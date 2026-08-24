<script>
  import { api, clearToken } from '../api.js'

  let { onopen } = $props()

  let notes    = $state([])
  let loading  = $state(true)
  let error    = $state('')
  let query    = $state('')

  async function load() {
    loading = true
    error   = ''
    try {
      notes = await api.notes()
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  $effect(() => { load() })

  // group notes by day (YYYY-MM-DD), newest day first
  let grouped = $derived.by(() => {
    const q = query.trim().toLowerCase()
    const filtered = q
      ? notes.filter(n =>
          n.slug.includes(q) ||
          (n.tags || []).some(t => t.includes(q))
        )
      : notes

    const byDay = new Map()
    for (const n of [...filtered].reverse()) {
      const day = (n.created || '').slice(0, 10)
      if (!byDay.has(day)) byDay.set(day, [])
      byDay.get(day).push(n)
    }
    return [...byDay.entries()].sort((a, b) => b[0].localeCompare(a[0]))
  })

  function time(created) {
    if (!created) return ''
    const d = new Date(created)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  }

  function logout() {
    clearToken()
    location.reload()
  }
</script>

<div class="shell">
  <header>
    <span class="logo">mem</span>
    <input class="search" type="search" placeholder="filter by slug or tag…" bind:value={query} />
    <button class="logout" onclick={logout} title="disconnect">⏻</button>
  </header>

  <main>
    {#if loading}
      <p class="status">loading…</p>
    {:else if error}
      <p class="status error">{error}</p>
    {:else if grouped.length === 0}
      <p class="status">no notes{query ? ' matching filter' : ''}</p>
    {:else}
      {#each grouped as [day, dayNotes]}
        <div class="day-group">
          <div class="day-header">{day}</div>
          {#each dayNotes as note}
            <button class="note-row" onclick={() => onopen(note.id)}>
              <span class="note-time">{time(note.created)}</span>
              <span class="note-slug">{note.slug}</span>
              <span class="note-tags">
                {#each (note.tags || []) as tag}
                  <span class="tag">#{tag}</span>
                {/each}
              </span>
            </button>
          {/each}
        </div>
      {/each}
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
    gap: 0.75rem;
    padding: 0.6rem 1rem;
  }
  .logo { font-weight: 700; letter-spacing: 0.08em; color: var(--accent); flex-shrink: 0 }
  .search {
    flex: 1;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text);
    font-size: 0.9rem;
    padding: 0.35rem 0.65rem;
  }
  .search:focus { outline: none; border-color: var(--accent) }
  .logout {
    background: none;
    border: none;
    color: var(--muted);
    cursor: pointer;
    font-size: 1rem;
    padding: 0.2rem;
  }
  .logout:hover { color: var(--text) }

  main { padding: 0 1rem 2rem }

  .day-group { margin-top: 1.5rem }
  .day-header {
    color: var(--muted);
    font-size: 0.78rem;
    font-weight: 600;
    letter-spacing: 0.06em;
    padding: 0.25rem 0;
    border-bottom: 1px solid var(--border);
    margin-bottom: 0.25rem;
    text-transform: uppercase;
  }

  .note-row {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
    width: 100%;
    background: none;
    border: none;
    border-radius: 6px;
    color: var(--text);
    cursor: pointer;
    padding: 0.4rem 0.5rem;
    text-align: left;
    font-size: 0.9rem;
  }
  .note-row:hover { background: var(--surface) }

  .note-time { color: var(--muted); font-variant-numeric: tabular-nums; flex-shrink: 0; font-size: 0.82rem }
  .note-slug { flex-shrink: 0 }
  .note-tags { display: flex; flex-wrap: wrap; gap: 0.3rem; min-width: 0 }
  .tag { color: var(--muted); font-size: 0.8rem }

  .status { color: var(--muted); padding: 2rem 0.5rem; font-size: 0.9rem }
  .status.error { color: #f85149 }
</style>
