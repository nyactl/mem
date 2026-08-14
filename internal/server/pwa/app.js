'use strict';

// ── IndexedDB ─────────────────────────────────────────────────────────────

const DB_NAME = 'mem-db';
const DB_VER  = 1;

let db;

function openDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VER);
    req.onupgradeneeded = e => {
      const d = e.target.result;
      if (!d.objectStoreNames.contains('notes')) {
        const s = d.createObjectStore('notes', { keyPath: 'id' });
        s.createIndex('created', 'created');
        s.createIndex('syncState', 'syncState');
      }
      if (!d.objectStoreNames.contains('outbox')) {
        d.createObjectStore('outbox', { keyPath: 'id', autoIncrement: true });
      }
    };
    req.onsuccess = e => resolve(e.target.result);
    req.onerror   = e => reject(e.target.error);
  });
}

function tx(store, mode, fn) {
  return new Promise((resolve, reject) => {
    const t = db.transaction(store, mode);
    t.onerror = e => reject(e.target.error);
    const s = t.objectStore(store);
    const req = fn(s);
    if (req && req.onsuccess !== undefined) {
      req.onsuccess = e => resolve(e.target.result);
      req.onerror   = e => reject(e.target.error);
    } else {
      t.oncomplete = () => resolve();
    }
  });
}

const store = {
  allNotes: () => new Promise((resolve, reject) => {
    const t = db.transaction('notes', 'readonly');
    const req = t.objectStore('notes').getAll();
    req.onsuccess = e => resolve(e.target.result);
    req.onerror   = e => reject(e.target.error);
  }),
  getNote: id => tx('notes', 'readonly', s => s.get(id)),
  putNote: note => tx('notes', 'readwrite', s => { s.put(note); }),
  deleteNote: id => tx('notes', 'readwrite', s => { s.delete(id); }),
  enqueue: item => tx('outbox', 'readwrite', s => { s.add(item); }),
  allOutbox: () => new Promise((resolve, reject) => {
    const t = db.transaction('outbox', 'readonly');
    const req = t.objectStore('outbox').getAll();
    req.onsuccess = e => resolve(e.target.result);
    req.onerror   = e => reject(e.target.error);
  }),
  clearOutbox: () => tx('outbox', 'readwrite', s => { s.clear(); }),
  deleteOutboxItem: id => tx('outbox', 'readwrite', s => { s.delete(id); }),
};

// ── API client ────────────────────────────────────────────────────────────

const api = {
  async list(since) {
    const path = since
      ? '/api/delta?since=' + encodeURIComponent(since)
      : '/api/notes';
    const r = await fetch(path);
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  },
  async get(id) {
    const r = await fetch('/api/notes/' + id);
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  },
  async create(payload) {
    const r = await fetch('/api/notes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  },
  async update(id, payload, etag) {
    const headers = { 'Content-Type': 'application/json' };
    if (etag) headers['If-Match'] = etag;
    const r = await fetch('/api/notes/' + id, {
      method: 'PATCH',
      headers,
      body: JSON.stringify(payload),
    });
    if (r.status === 409) {
      const serverEtag = r.headers.get('ETag');
      const err = new Error('conflict');
      err.conflict = true;
      err.serverEtag = serverEtag;
      throw err;
    }
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  },
  async delete(id) {
    const r = await fetch('/api/notes/' + id, { method: 'DELETE' });
    if (!r.ok) throw new Error(await r.text());
  },
};

// ── Sync engine ───────────────────────────────────────────────────────────

let syncInProgress = false;
let lastSyncTime   = null;

async function sync() {
  if (syncInProgress || !navigator.onLine) return;
  syncInProgress = true;
  try {
    // 1. Drain outbox (pending local creates/updates)
    const outbox = await store.allOutbox();
    for (const item of outbox) {
      try {
        if (item.action === 'create') {
          const note = await api.create(item.payload);
          // replace the local pending note with the server's canonical version
          await store.deleteNote(item.localId);
          await store.putNote({ ...note, syncState: 'synced' });
        } else if (item.action === 'update') {
          const note = await api.update(item.id, item.payload, item.etag);
          await store.putNote({ ...note, syncState: 'synced' });
        } else if (item.action === 'delete') {
          await api.delete(item.id);
          await store.deleteNote(item.id);
        }
        await store.deleteOutboxItem(item.id);
      } catch (err) {
        if (err.conflict) {
          // mark local note as conflicted — user must resolve
          const local = await store.getNote(item.id);
          if (local) {
            await store.putNote({ ...local, syncState: 'conflict' });
          }
          await store.deleteOutboxItem(item.id);
        }
        // non-conflict errors: leave in outbox, retry next sync
      }
    }

    // 2. Pull new/changed notes from server
    const sinceParam = lastSyncTime
      ? '?since=' + encodeURIComponent(lastSyncTime)
      : '';
    const serverNotes = await api.list();
    for (const sn of serverNotes) {
      const local = await store.getNote(sn.id);
      if (!local || (local.syncState === 'synced' && local.etag !== sn.etag)) {
        // fetch full content only if we don't have it or it changed
        const full = await api.get(sn.id);
        await store.putNote({ ...full, syncState: 'synced' });
      }
    }
    lastSyncTime = new Date().toISOString();
  } finally {
    syncInProgress = false;
    render();
  }
}

// ── State ─────────────────────────────────────────────────────────────────

let state = {
  view:    'list',  // 'list' | 'new' | 'note' | 'edit'
  notes:   [],
  noteId:  null,
  search:  '',
  online:  navigator.onLine,
  pending: 0,
  saving:  false,
  saveMsg: '',
};

function setState(patch) {
  state = { ...state, ...patch };
  render();
}

// ── Routing (hash) ────────────────────────────────────────────────────────

function route() {
  const hash = location.hash || '#list';
  if (hash === '#list' || hash === '#') {
    setState({ view: 'list', noteId: null });
  } else if (hash === '#new') {
    setState({ view: 'new', noteId: null });
  } else if (hash.startsWith('#note/')) {
    setState({ view: 'note', noteId: hash.slice(6) });
  } else if (hash.startsWith('#edit/')) {
    setState({ view: 'edit', noteId: hash.slice(6) });
  }
}

window.addEventListener('hashchange', () => { route(); loadNotes(); });

// ── Data loading ──────────────────────────────────────────────────────────

async function loadNotes() {
  const all   = await store.allNotes();
  const outbox = await store.allOutbox();
  state = { ...state, notes: all, pending: outbox.length };
  render();
}

// ── Save note ─────────────────────────────────────────────────────────────

function tsNow() {
  const d = new Date();
  const pad = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}${pad(d.getMonth()+1)}${pad(d.getDate())}T${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
}

async function saveNew(title, body) {
  const localId  = tsNow() + '-pending-' + Math.random().toString(36).slice(2, 6);
  const created  = new Date().toISOString();
  const payload  = { title, body, created };

  // optimistic local write
  const localNote = {
    id: localId, slug: title || '(untitled)', created, tags: [], sources: [],
    body, etag: '', syncState: 'pending',
  };
  await store.putNote(localNote);
  await store.enqueue({ action: 'create', localId, payload });
  await loadNotes();
  setState({ saveMsg: 'saved' });

  if (navigator.onLine) {
    await sync();
  }
}

async function saveEdit(id, body, etag) {
  const local = await store.getNote(id);
  if (!local) return;
  const updated = { ...local, body, syncState: 'pending' };
  await store.putNote(updated);
  await store.enqueue({ action: 'update', id, payload: { body, tags: local.tags, sources: local.sources }, etag });
  setState({ saveMsg: 'saved' });
  if (navigator.onLine) await sync();
}

// ── Render ────────────────────────────────────────────────────────────────

const app = document.getElementById('app');

function render() {
  const { view, notes, noteId, search, online, pending } = state;

  const dotClass = !online ? 'offline' : pending > 0 ? 'pending' : '';
  const statusText = !online
    ? `offline · ${pending} queued`
    : pending > 0
      ? `syncing ${pending} note${pending > 1 ? 's' : ''}…`
      : 'synced';

  let html = '';

  if (view === 'list') {
    const filtered = notes.filter(n => {
      if (!search) return true;
      const q = search.toLowerCase();
      return (n.slug || '').includes(q) ||
             (n.body || '').toLowerCase().includes(q) ||
             (n.tags || []).some(t => t.includes(q));
    }).sort((a, b) => b.created.localeCompare(a.created));

    html = `
      <div class="topbar">
        <span class="topbar-logo">mem</span>
        <span class="topbar-title"></span>
        <button class="btn-icon" onclick="location.hash='#new'" aria-label="New note">+</button>
      </div>
      <div class="scroll">
        <div class="search-wrap">
          <input class="search-input" type="search" placeholder="search…"
            value="${esc(search)}"
            oninput="handleSearch(this.value)"
            autocomplete="off" autocorrect="off" spellcheck="false">
        </div>
        ${filtered.length === 0 ? '<div class="empty">No notes yet.<br>Tap + to capture one.</div>' : ''}
        <ul class="note-list">
          ${filtered.map(n => `
            <li class="note-item" onclick="location.hash='#note/${esc(n.id)}'">
              <div class="note-item-title">${esc(displayTitle(n))}</div>
              <div class="note-item-meta">
                <span>${fmtDate(n.created)}</span>
                ${(n.tags || []).slice(0, 3).map(t => `<span class="tag">#${esc(t)}</span>`).join('')}
                ${n.syncState === 'pending' ? '<span class="pending-badge">⟳ pending</span>' : ''}
                ${n.syncState === 'conflict' ? '<span class="pending-badge" style="color:var(--danger)">⚠ conflict</span>' : ''}
              </div>
            </li>
          `).join('')}
        </ul>
      </div>
      <div class="status">
        <div class="status-dot ${dotClass}"></div>
        <span>${statusText}</span>
      </div>
    `;

  } else if (view === 'new') {
    html = `
      <div class="topbar">
        <button class="btn-icon" onclick="location.hash='#list'" aria-label="Back">←</button>
        <span class="topbar-title">New note</span>
      </div>
      <div class="scroll">
        <div class="field-title">
          <input id="inp-title" class="input-title" type="text"
            placeholder="Title (optional)" autocorrect="on" autocapitalize="sentences">
        </div>
        <div class="field-body">
          <textarea id="inp-body" class="textarea-body"
            placeholder="Write your note…&#10;Use #tags and @sources inline."
            autocorrect="on" autocapitalize="sentences"></textarea>
        </div>
        <div class="capture-hints">
          Use #tag to tag · @name to reference a person or source
        </div>
      </div>
      <div class="save-bar">
        <span class="save-status" id="save-status">${esc(state.saveMsg)}</span>
        <button class="btn-primary" onclick="handleSaveNew()">Save</button>
      </div>
    `;

  } else if (view === 'note') {
    const n = notes.find(x => x.id === noteId);
    if (!n) {
      html = `
        <div class="topbar">
          <button class="btn-icon" onclick="location.hash='#list'">←</button>
          <span class="topbar-title">Not found</span>
        </div>
        <div class="scroll"><div class="empty">Note not found.</div></div>
      `;
    } else {
      html = `
        <div class="topbar">
          <button class="btn-icon" onclick="location.hash='#list'">←</button>
          <span class="topbar-title"></span>
          <button class="btn-text" onclick="location.hash='#edit/${esc(n.id)}'">Edit</button>
        </div>
        <div class="scroll">
          <div class="note-header">
            <div class="note-view-title">${esc(displayTitle(n))}</div>
            <div class="note-view-meta">
              <span>${fmtDateFull(n.created)}</span>
              ${(n.tags || []).map(t => `<span class="tag">#${esc(t)}</span>`).join('')}
              ${n.syncState === 'conflict'
                ? `<span style="color:var(--danger);font-size:11px">⚠ conflict — edit to resolve</span>`
                : ''}
            </div>
          </div>
          <div class="note-body">${esc(n.body || '')}</div>
        </div>
        <div class="status">
          <div class="status-dot ${dotClass}"></div>
          <span>${statusText}</span>
        </div>
      `;
    }

  } else if (view === 'edit') {
    const n = notes.find(x => x.id === noteId);
    if (!n) {
      location.hash = '#list';
      return;
    }
    html = `
      <div class="topbar">
        <button class="btn-icon" onclick="location.hash='#note/${esc(n.id)}'">←</button>
        <span class="topbar-title">Edit</span>
      </div>
      <div class="scroll">
        <div class="field-body">
          <textarea id="inp-body" class="textarea-body"
            autocorrect="on" autocapitalize="sentences">${esc(n.body || '')}</textarea>
        </div>
      </div>
      <div class="save-bar">
        <span class="save-status" id="save-status">${esc(state.saveMsg)}</span>
        <button class="btn-primary" onclick="handleSaveEdit('${esc(n.id)}','${esc(n.etag || '')}')">Save</button>
      </div>
    `;
  }

  app.innerHTML = html;

  // autofocus body on new/edit
  if (view === 'new') {
    const ta = document.getElementById('inp-body');
    if (ta) setTimeout(() => ta.focus(), 50);
  }
  if (view === 'edit') {
    const ta = document.getElementById('inp-body');
    if (ta) {
      setTimeout(() => {
        ta.focus();
        ta.setSelectionRange(ta.value.length, ta.value.length);
      }, 50);
    }
  }
}

// ── Event handlers (called from inline html) ──────────────────────────────

window.handleSearch = function(v) {
  state = { ...state, search: v };
  render();
};

window.handleSaveNew = async function() {
  const title = (document.getElementById('inp-title')?.value || '').trim();
  const body  = (document.getElementById('inp-body')?.value || '').trim();
  if (!body && !title) return;
  setState({ saving: true, saveMsg: 'saving…' });
  await saveNew(title, body);
  location.hash = '#list';
};

window.handleSaveEdit = async function(id, etag) {
  const body = (document.getElementById('inp-body')?.value || '').trim();
  setState({ saving: true, saveMsg: 'saving…' });
  await saveEdit(id, body, etag);
  location.hash = '#note/' + id;
};

// ── Helpers ───────────────────────────────────────────────────────────────

function esc(s) {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function displayTitle(n) {
  if (n.slug && n.slug !== 'note' && n.slug !== 'ingest' && !n.slug.startsWith('pending')) {
    return n.slug.replace(/-/g, ' ');
  }
  const first = (n.body || '').split('\n').find(l => l.trim());
  return first ? first.replace(/^#+\s*/, '').slice(0, 60) : '(untitled)';
}

function fmtDate(iso) {
  const d = new Date(iso);
  const now = new Date();
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

function fmtDateFull(iso) {
  const d = new Date(iso);
  return d.toLocaleString([], {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

// ── Bootstrap ─────────────────────────────────────────────────────────────

window.addEventListener('online',  () => { setState({ online: true });  sync(); });
window.addEventListener('offline', () => { setState({ online: false }); });

(async function init() {
  db = await openDB();
  route();
  await loadNotes();
  await sync();
  // background sync every 30s while online
  setInterval(() => { if (navigator.onLine) sync(); }, 30_000);
})();
