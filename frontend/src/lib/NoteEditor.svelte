<script>
  import { api } from '../api.js'
  import { marked } from 'marked'
  import DOMPurify from 'dompurify'
  import { onMount, tick } from 'svelte'

  let { id = null, onsave, oncancel } = $props()

  let isEdit = $derived(id !== null)

  let title      = $state('')
  let tagsRaw    = $state('')
  let dateRaw    = $state('')
  let body       = $state('')
  let tab        = $state('write')
  let saving     = $state(false)
  let loading    = $state(false)
  let error      = $state('')
  let etag       = $state('')
  let allTags    = $state([])
  let tagFocus   = $state(false)
  let accelIdx   = $state(-1)
  let tagsEl     = $state(null)

  onMount(async () => {
    try {
      const notes = await api.notes()
      const counts = new Map()
      for (const n of notes)
        for (const t of (n.tags || []))
          counts.set(t, (counts.get(t) || 0) + 1)
      allTags = [...counts.entries()].sort((a,b) => b[1]-a[1]).map(([t]) => t)
    } catch {}

    if (id) {
      loading = true
      try {
        const n = await api.note(id)
        title   = n.slug || n.id || ''
        body    = n.body || ''
        tagsRaw = (n.tags || []).join(' ')
        etag    = n.etag || ''
        if (n.date) {
          const d = new Date(n.date)
          dateRaw = d.toISOString().slice(0, 16)
        }
      } catch (e) {
        error = e.message
      } finally {
        loading = false
      }
    }
  })

  // The word currently being typed in the tags field
  let currentWord = $derived.by(() => {
    const parts = tagsRaw.split(/[\s,]+/)
    return parts[parts.length - 1].replace(/^#/, '').toLowerCase()
  })

  let suggestions = $derived.by(() => {
    if (!tagFocus || !currentWord) return []
    return allTags.filter(t => t.startsWith(currentWord) && t !== currentWord).slice(0, 8)
  })

  function pickSuggestion(tag) {
    const parts = tagsRaw.split(/[\s,]+/)
    parts[parts.length - 1] = tag
    tagsRaw = parts.join(' ') + ' '
    accelIdx = -1
    tagsEl?.focus()
  }

  function onTagsKeydown(e) {
    if (!suggestions.length) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      accelIdx = Math.min(accelIdx + 1, suggestions.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      accelIdx = Math.max(accelIdx - 1, -1)
    } else if ((e.key === 'Tab' || e.key === 'Enter') && accelIdx >= 0) {
      e.preventDefault()
      pickSuggestion(suggestions[accelIdx])
    } else if (e.key === 'Escape') {
      accelIdx = -1
      tagFocus = false
    }
  }

  marked.use({
    breaks: true,
    renderer: {
      code(token) {
        if (token.lang === 'mermaid') return `<div class="mermaid">${token.text}</div>`
        return false
      }
    }
  })

  let previewEl = $state(null)

  let html = $derived(
    tab === 'preview' && body
      ? DOMPurify.sanitize(marked.parse(body))
      : ''
  )

  $effect(() => {
    if (tab === 'preview' && html) {
      tick().then(async () => {
        const nodes = previewEl?.querySelectorAll('.mermaid')
        if (!nodes?.length) return
        const { default: mermaid } = await import('mermaid')
        mermaid.initialize({ startOnLoad: false, theme: 'neutral' })
        nodes.forEach(n => { n.removeAttribute('data-processed'); n.innerHTML = n.textContent ?? '' })
        mermaid.run({ nodes })
      })
    }
  })

  function parseTags(raw) {
    return raw.split(/[\s,]+/).map(t => t.replace(/^#/, '').trim()).filter(Boolean)
  }

  async function save() {
    saving = true; error = ''
    try {
      if (isEdit) {
        const payload = { body, tags: parseTags(tagsRaw), sources: [] }
        if (dateRaw.trim()) payload.date = new Date(dateRaw).toISOString()
        await api.update(id, payload, etag)
        onsave(id)
      } else {
        const slug = title.trim()
        if (!slug) { error = 'Title is required.'; saving = false; return }
        const payload = {
          title:   slug,
          body,
          tags:    parseTags(tagsRaw),
          sources: [],
          created: new Date().toISOString(),
        }
        if (dateRaw.trim()) payload.date = new Date(dateRaw).toISOString()
        const note = await api.create(payload)
        onsave(note.id)
      }
    } catch (e) {
      error = e.message
    } finally {
      saving = false
    }
  }

  $effect(() => {
    function handler(e) {
      // Ctrl+Shift+P (all platforms) — toggle write/preview
      if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'p') {
        e.preventDefault()
        tab = tab === 'write' ? 'preview' : 'write'
        return
      }
      // ⌘↵ / Ctrl+Enter — save
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault()
        save()
        return
      }
      // Escape — cancel (only when tag dropdown is closed)
      if (e.key === 'Escape' && !suggestions.length) oncancel()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  })
</script>

<div class="shell" role="none">
  <header>
    <button class="back" onclick={oncancel}>← back</button>
    <span class="heading">{isEdit ? 'edit note' : 'new note'}</span>
    <div class="spacer"></div>
    {#if error}<span class="err">{error}</span>{/if}
    <button class="save-btn" onclick={save} disabled={saving || loading}>
      {saving ? 'saving…' : 'save'}
    </button>
  </header>

  <main>
    {#if isEdit}
      <span class="slug-label">{title}</span>
    {:else}
      <input
        class="title-input"
        type="text"
        placeholder="title"
        bind:value={title}
        autocomplete="off"
        spellcheck="false"
      />
    {/if}

    <input
      class="date-input"
      type="datetime-local"
      bind:value={dateRaw}
      title="override display date (leave blank to use now)"
    />

    <div class="tags-wrap">
      <input
        bind:this={tagsEl}
        class="tags-input"
        type="text"
        placeholder="tags  (space or comma separated)"
        bind:value={tagsRaw}
        autocomplete="off"
        spellcheck="false"
        onfocus={() => { tagFocus = true }}
        onblur={() => setTimeout(() => { tagFocus = false; accelIdx = -1 }, 150)}
        onkeydown={onTagsKeydown}
      />
      {#if suggestions.length > 0}
        <ul class="suggestions" role="listbox">
          {#each suggestions as tag, i (tag)}
            <li
              class="suggestion"
              class:highlighted={i === accelIdx}
              role="option"
              aria-selected={i === accelIdx}
              onmousedown={() => pickSuggestion(tag)}
            >#{tag}</li>
          {/each}
        </ul>
      {/if}
    </div>

    <div class="tabs">
      <button class="tab" class:active={tab === 'write'}   onclick={() => tab = 'write'}>write</button>
      <button class="tab" class:active={tab === 'preview'} onclick={() => tab = 'preview'}>preview</button>
      <span class="hint">⌘↵ save · ^⇧P toggle preview</span>
    </div>

    {#if tab === 'write'}
      <textarea
        class="body-input"
        placeholder="write in Markdown…"
        bind:value={body}
        spellcheck="true"
      ></textarea>
    {:else}
      <div class="preview md-body" bind:this={previewEl}>
        {#if html}
          {@html html}
        {:else}
          <p class="empty">nothing to preview yet</p>
        {/if}
      </div>
    {/if}
  </main>
</div>

<style>
  .shell { display: flex; flex-direction: column; min-height: 100dvh }

  header {
    position: sticky; top: 0; z-index: 10;
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px);
    border-bottom: 1px solid var(--border);
    display: flex; align-items: center; gap: 0.75rem; padding: 0.65rem 1.25rem;
  }
  .back { background: none; border: none; color: var(--accent); cursor: pointer; font-size: 0.9rem; padding: 0; flex-shrink: 0 }
  .back:hover { text-decoration: underline }
  .heading { color: var(--muted); font-size: 0.85rem }
  .spacer  { flex: 1 }
  .err { color: #f85149; font-size: 0.82rem; flex-shrink: 0 }
  .save-btn {
    background: var(--accent); border: none; border-radius: 6px;
    color: #fff; cursor: pointer; font-size: 0.85rem; font-weight: 600;
    padding: 0.35rem 0.9rem; flex-shrink: 0;
  }
  .save-btn:disabled { opacity: 0.5; cursor: default }
  .save-btn:not(:disabled):hover { filter: brightness(1.1) }

  main {
    display: flex; flex-direction: column; flex: 1;
    padding: 1.25rem 1.5rem 2rem;
    max-width: 720px; margin: 0 auto; width: 100%; gap: 0.6rem;
  }

  .date-input {
    background: transparent;
    border: none;
    border-bottom: 1px solid var(--border);
    color: var(--muted);
    font-family: inherit;
    font-size: 0.82rem;
    padding: 0.4rem 0;
    width: auto;
    color-scheme: dark light;
  }
  .date-input:focus { outline: none; border-bottom-color: var(--accent) }

  .slug-label {
    display: block;
    font-size: 1.3rem;
    font-weight: 700;
    letter-spacing: -0.01em;
    color: var(--accent);
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }

  .title-input, .tags-input {
    background: transparent; border: none;
    border-bottom: 1px solid var(--border);
    color: var(--text); font-family: inherit; padding: 0.4rem 0; width: 100%;
  }
  .title-input { font-size: 1.3rem; font-weight: 700; letter-spacing: -0.01em }
  .title-input::placeholder { color: var(--muted) }
  .tags-input { font-size: 0.85rem; color: var(--muted) }
  .tags-input::placeholder { color: var(--border) }
  .title-input:focus, .tags-input:focus { outline: none; border-bottom-color: var(--accent) }

  /* ── Tag autocomplete ─────────────────────────────────────────── */
  .tags-wrap { position: relative }

  .suggestions {
    position: absolute; top: calc(100% + 2px); left: 0;
    background: var(--surface); border: 1px solid var(--border); border-radius: 6px;
    list-style: none; margin: 0; padding: 0.25rem 0;
    width: 100%; max-height: 220px; overflow-y: auto;
    z-index: 20; box-shadow: 0 4px 16px rgba(0,0,0,0.2);
  }
  .suggestion {
    color: var(--muted); cursor: pointer;
    font-size: 0.82rem; padding: 0.35rem 0.9rem;
  }
  .suggestion:hover, .suggestion.highlighted {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
  }

  /* ── Tabs ─────────────────────────────────────────────────────── */
  .tabs {
    display: flex; align-items: center; gap: 0;
    margin-top: 0.4rem; border-bottom: 1px solid var(--border);
  }
  .tab {
    background: none; border: none; color: var(--muted); cursor: pointer;
    font-size: 0.82rem; padding: 0.4rem 0.9rem 0.4rem 0;
    border-bottom: 2px solid transparent; margin-bottom: -1px;
    transition: color 0.1s, border-color 0.1s;
  }
  .tab.active { color: var(--accent); border-bottom-color: var(--accent) }
  .tab:hover:not(.active) { color: var(--text) }
  .hint { margin-left: auto; font-size: 0.75rem; color: var(--border) }

  /* ── Body ─────────────────────────────────────────────────────── */
  .body-input {
    flex: 1; min-height: 400px; background: transparent; border: none;
    color: var(--text);
    font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 0.9rem; line-height: 1.7; padding: 0.75rem 0; resize: none; width: 100%;
  }
  .body-input:focus { outline: none }
  .body-input::placeholder { color: var(--border) }

  .preview { flex: 1; min-height: 400px; padding: 0.75rem 0 }
  .empty { color: var(--border); font-size: 0.9rem }

  /* ── Markdown preview ─────────────────────────────────────────── */
  .md-body :global(h1),.md-body :global(h2),.md-body :global(h3),.md-body :global(h4) {
    color: var(--text); font-weight: 600; line-height: 1.3; margin: 1.5em 0 0.5em;
  }
  .md-body :global(h1) { font-size: 1.4rem }
  .md-body :global(h2) { font-size: 1.15rem }
  .md-body :global(h3) { font-size: 1rem }
  .md-body :global(p)  { margin: 0.75em 0; font-size: 0.95rem; line-height: 1.7; color: var(--text) }
  .md-body :global(ul),.md-body :global(ol) { padding-left: 1.5em; margin: 0.75em 0 }
  .md-body :global(li) { margin: 0.25em 0; font-size: 0.95rem; color: var(--text) }
  .md-body :global(a)  { color: var(--accent); text-decoration: none }
  .md-body :global(a:hover) { text-decoration: underline }
  .md-body :global(blockquote) {
    border-left: 3px solid var(--border); margin: 0.75em 0;
    padding: 0.25em 0 0.25em 1em; color: var(--muted);
  }
  .md-body :global(code) {
    background: var(--surface); border: 1px solid var(--border); border-radius: 3px;
    font-family: ui-monospace,'Fira Code',monospace; font-size: 0.85em; padding: 0.1em 0.35em;
  }
  .md-body :global(pre) {
    background: var(--surface); border: 1px solid var(--border); border-radius: 6px;
    font-family: ui-monospace,'Fira Code',monospace; font-size: 0.85rem;
    line-height: 1.55; margin: 1em 0; overflow-x: auto; padding: 1em 1.25em;
  }
  .md-body :global(pre code) { background: none; border: none; padding: 0 }
  .md-body :global(table) {
    border-collapse: collapse; font-size: 0.88rem; margin: 1em 0;
    overflow-x: auto; display: block; width: max-content; max-width: 100%;
  }
  .md-body :global(th),.md-body :global(td) { border: 1px solid var(--border); padding: 0.4em 0.75em }
  .md-body :global(th) { background: var(--surface); font-weight: 600; text-align: left }
  .md-body :global(hr) { border: none; border-top: 1px solid var(--border); margin: 1.5em 0 }
</style>
