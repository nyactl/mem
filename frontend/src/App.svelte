<script>
  import { getToken, setToken } from './api.js'
  import NoteList from './lib/NoteList.svelte'
  import NoteDetail from './lib/NoteDetail.svelte'

  let token = $state(getToken())
  let tokenInput = $state('')

  // hash-based routing: '' or '#/' → list, '#/note/<id>' → detail
  let hash = $state(location.hash)
  $effect(() => {
    const update = () => { hash = location.hash }
    window.addEventListener('hashchange', update)
    return () => window.removeEventListener('hashchange', update)
  })

  function navigate(path) {
    location.hash = path
  }

  let view = $derived(parseHash(hash))

  function parseHash(h) {
    const s = h.replace(/^#\/?/, '')
    if (s.startsWith('note/')) return { name: 'detail', id: decodeURIComponent(s.slice(5)) }
    return { name: 'list' }
  }

  function saveToken() {
    const t = tokenInput.trim()
    if (!t) return
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
        <input
          type="password"
          placeholder="auth token"
          bind:value={tokenInput}
          autofocus
        />
        <button type="submit">Connect</button>
      </form>
    </div>
  </div>
{:else if view.name === 'detail'}
  <NoteDetail id={view.id} onback={() => navigate('/')} />
{:else}
  <NoteList onopen={(id) => navigate(`/note/${encodeURIComponent(id)}`)} />
{/if}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0 }
  :global(:root) {
    --bg:      #0d1117;
    --surface: #161b22;
    --border:  #30363d;
    --text:    #e6edf3;
    --muted:   #7d8590;
    --accent:  #3fb950;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    font-size: 15px;
    background: var(--bg);
    color: var(--text);
  }
  @media (prefers-color-scheme: light) {
    :global(:root:not([data-theme="dark"])) {
      --bg:      #ffffff;
      --surface: #f6f8fa;
      --border:  #d0d7de;
      --text:    #1f2328;
      --muted:   #636c76;
      --accent:  #1a7f37;
    }
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
  .auth-card h1 { font-size: 1.5rem; letter-spacing: 0.05em }
  .auth-card p  { color: var(--muted); font-size: 0.9rem }
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
