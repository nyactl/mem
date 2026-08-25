<script>
  import { getToken, setToken } from './api.js'
  import NoteList from './lib/NoteList.svelte'
  import NoteDetail from './lib/NoteDetail.svelte'
  import NoteEditor from './lib/NoteEditor.svelte'

  let token      = $state(getToken())
  let tokenInput = $state('')
  let authError  = $state('')

  // ── theme ──────────────────────────────────────────────────────
  let theme = $state(localStorage.getItem('mem_theme') || 'system')

  $effect(() => {
    const root = document.documentElement
    if (theme === 'system') root.removeAttribute('data-theme')
    else root.setAttribute('data-theme', theme)
    localStorage.setItem('mem_theme', theme)
  })

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark'
  }

  // ── routing ────────────────────────────────────────────────────
  let hash = $state(location.hash)
  $effect(() => {
    const update = () => { hash = location.hash }
    window.addEventListener('hashchange', update)
    return () => window.removeEventListener('hashchange', update)
  })

  let view = $derived(parseHash(hash))

  function parseHash(h) {
    const s = h.replace(/^#\/?/, '')
    if (s.startsWith('note/')) return { name: 'detail', id: decodeURIComponent(s.slice(5)) }
    if (s === 'new') return { name: 'new' }
    return { name: 'list' }
  }

  function navigate(path) { location.hash = path }

  async function saveToken() {
    const t = tokenInput.trim()
    if (!t) return
    authError = ''
    try {
      const res = await fetch('/api/notes', { headers: { 'Authorization': 'Bearer ' + t } })
      if (res.status === 401) { authError = 'Invalid token.'; return }
      if (!res.ok) { authError = `Server error: ${res.status}`; return }
    } catch {
      authError = 'Could not reach server.'
      return
    }
    setToken(t)
    token = t
    tokenInput = ''
  }
</script>

{#if !token}
  <div class="auth">
    <div class="auth-card">
      <h1>mem</h1>
      <p>Enter your auth token to continue.</p>
      <form onsubmit={(e) => { e.preventDefault(); saveToken() }}>
        <input type="password" placeholder="auth token" bind:value={tokenInput} />
        <button type="submit">Connect</button>
        {#if authError}<p class="auth-error">{authError}</p>{/if}
      </form>
    </div>
  </div>
{:else if view.name === 'detail'}
  <NoteDetail
    id={view.id}
    onback={() => navigate('/')}
    {theme}
    {toggleTheme}
  />
{:else if view.name === 'new'}
  <NoteEditor
    onsave={(id) => navigate(id ? `/note/${encodeURIComponent(id)}` : '/')}
    oncancel={() => navigate('/')}
  />
{:else}
  <NoteList
    onopen={(id) => navigate(`/note/${encodeURIComponent(id)}`)}
    onnew={() => navigate('/new')}
    {theme}
    {toggleTheme}
  />
{/if}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0 }

  /* dark-first defaults */
  :global(:root) {
    --bg:      #0d1117;
    --surface: #161b22;
    --border:  #30363d;
    --text:    #e6edf3;
    --muted:   #7d8590;
    --accent:  #818cf8;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    font-size: 16px;
    background: var(--bg);
    color: var(--text);
  }

  /* system light */
  @media (prefers-color-scheme: light) {
    :global(:root:not([data-theme="dark"])) {
      --bg:      #ffffff;
      --surface: #f6f8fa;
      --border:  #d0d7de;
      --text:    #1f2328;
      --muted:   #636c76;
      --accent:  #4f46e5;
    }
  }

  /* explicit overrides */
  :global(:root[data-theme="dark"]) {
    --bg:      #0d1117;
    --surface: #161b22;
    --border:  #30363d;
    --text:    #e6edf3;
    --muted:   #7d8590;
    --accent:  #818cf8;
  }
  :global(:root[data-theme="light"]) {
    --bg:      #ffffff;
    --surface: #f6f8fa;
    --border:  #d0d7de;
    --text:    #1f2328;
    --muted:   #636c76;
    --accent:  #4f46e5;
  }

  :global(body) { background: var(--bg); min-height: 100dvh }

  .auth {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100dvh;
    padding: 1rem;
  }
  .auth-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 2rem;
    width: 100%;
    max-width: 360px;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .auth-card h1 { font-size: 1.5rem; letter-spacing: 0.05em; color: var(--accent) }
  .auth-card p  { color: var(--muted); font-size: 0.9rem }
  .auth-error   { color: #f85149; font-size: 0.85rem }
  form { display: flex; flex-direction: column; gap: 0.5rem }
  input {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text);
    padding: 0.5rem 0.75rem;
    font-size: 0.95rem;
    width: 100%;
  }
  input:focus { outline: none; border-color: var(--accent) }
  button {
    background: var(--accent);
    border: none;
    border-radius: 6px;
    color: #fff;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 600;
    padding: 0.5rem;
  }
  button:hover { filter: brightness(1.1) }
</style>
