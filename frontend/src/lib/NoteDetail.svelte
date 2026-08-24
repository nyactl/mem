<script>
  import { api } from '../api.js'

  let { id, onback } = $props()

  let note    = $state(null)
  let loading = $state(true)
  let error   = $state('')

  async function load() {
    loading = true
    error   = ''
    note    = null
    try {
      note = await api.note(id)
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  $effect(() => { id; load() })

  function formatDate(iso) {
    if (!iso) return ''
    return new Date(iso).toLocaleString([], { dateStyle: 'medium', timeStyle: 'medium' })
  }
</script>

<div class="shell">
  <header>
    <button class="back" onclick={onback}>← back</button>
    {#if note}
      <span class="slug">{note.slug}</span>
    {/if}
  </header>

  <main>
    {#if loading}
      <p class="status">loading…</p>
    {:else if error}
      <p class="status error">{error}</p>
    {:else if note}
      <div class="meta">
        <span class="date">{formatDate(note.created)}</span>
        {#each (note.tags || []) as tag}
          <span class="tag">#{tag}</span>
        {/each}
      </div>
      <pre class="body">{note.body}</pre>
    {/if}
  </main>
</div>

<style>
  .shell { display: flex; flex-direction: column; min-height: 100dvh }

  header {
    position: sticky;
    top: 0;
    z-index: 10;
    background: var(--bg);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.6rem 1rem;
  }
  .back {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 0.9rem;
    padding: 0;
    flex-shrink: 0;
  }
  .back:hover { text-decoration: underline }
  .slug { color: var(--muted); font-size: 0.9rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap }

  main { padding: 1.25rem 1rem 2rem; max-width: 720px }

  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
  }
  .date { color: var(--muted); font-size: 0.82rem }
  .tag  { color: var(--muted); font-size: 0.82rem }

  .body {
    white-space: pre-wrap;
    word-break: break-word;
    font-family: inherit;
    font-size: 0.95rem;
    line-height: 1.65;
    color: var(--text);
  }

  .status { color: var(--muted); padding: 2rem 0.5rem; font-size: 0.9rem }
  .status.error { color: #f85149 }
</style>
