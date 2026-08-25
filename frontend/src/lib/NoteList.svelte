<script>
  import { api, clearToken } from '../api.js'
  import { onMount } from 'svelte'

  let { onopen, onnew, theme, toggleTheme } = $props()

  let notes   = $state([])
  let loading = $state(true)
  let error   = $state('')
  let query   = $state('')

  let heroInput    = $state(null)
  let resultsInput = $state(null)

  onMount(load)

  async function load() {
    loading = true; error = ''
    try { notes = await api.notes() }
    catch (e) { error = e.message }
    finally { loading = false }
  }

  let hero = $derived(query === '')

  // Refocus the visible input whenever the mode switches
  $effect(() => {
    if (hero) heroInput?.focus()
    else resultsInput?.focus()
  })

  // Tag frequency list — top 20 by note count
  let tagList = $derived.by(() => {
    const counts = new Map()
    for (const n of notes) {
      for (const t of (n.tags || [])) {
        counts.set(t, (counts.get(t) || 0) + 1)
      }
    }
    return [...counts.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 20)
      .map(([tag, count]) => ({ tag, count }))
  })

  function effectiveDate(n) {
    return n.date || n.created || ''
  }

  // Full timeline when query is empty; filtered when not
  let grouped = $derived.by(() => {
    const q = query.trim().toLowerCase()
    const filtered = q
      ? notes.filter(n => n.slug.includes(q) || (n.tags || []).some(t => t.includes(q)))
      : notes
    const sorted = [...filtered].sort((a, b) => effectiveDate(b).localeCompare(effectiveDate(a)))
    const byDay = new Map()
    for (const n of sorted) {
      const day = effectiveDate(n).slice(0, 10)
      if (!byDay.has(day)) byDay.set(day, [])
      byDay.get(day).push(n)
    }
    return [...byDay.entries()].sort((a, b) => b[0].localeCompare(a[0]))
  })

  function time(n) {
    const iso = n.date || n.created
    if (!iso) return ''
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  function logout() { clearToken(); location.reload() }

  let isDark = $derived(theme !== 'light')
</script>

<div class="shell">

  <!-- ── Hero header ────────────────────────────────────────────── -->
  {#if hero}
    <div class="hero-header">
      <div class="corner-actions">
        <button class="action new-btn" onclick={onnew} aria-label="new note">
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true"><line x1="8" y1="2" x2="8" y2="14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><line x1="2" y1="8" x2="14" y2="8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
        </button>
        <button class="action icon-btn" onclick={toggleTheme} aria-label="toggle theme">
          {#if isDark}
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="3" stroke="currentColor" stroke-width="1.5"/><line x1="8" y1="1" x2="8" y2="3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="8" y1="13" x2="8" y2="15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="1" y1="8" x2="3" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="13" y1="8" x2="15" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="2.93" x2="4.34" y2="4.34" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="11.66" x2="13.07" y2="13.07" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="4.34" x2="13.07" y2="2.93" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="13.07" x2="4.34" y2="11.66" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
          {:else}
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M13.5 9a5.5 5.5 0 0 1-7-7 5.5 5.5 0 1 0 7 7Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>
          {/if}
        </button>
        <button class="action icon-btn" onclick={logout} aria-label="log out">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M6 3H3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><polyline points="10,5 13,8 10,11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="13" y1="8" x2="6" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
        </button>
      </div>

      <h1 class="wordmark"><span class="prompt" aria-hidden="true">❯</span> mem</h1>

      <div class="search-wrap">
        <svg class="search-icon" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.5"/>
          <line x1="10.5" y1="10.5" x2="14" y2="14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <input
          bind:this={heroInput}
          class="search hero-search"
          type="search"
          placeholder="search notes…"
          value={query}
          oninput={(e) => { query = e.currentTarget.value }}
        />
      </div>

      {#if tagList.length > 0}
        <div class="tag-pills">
          {#each tagList as { tag } (tag)}
            <button class="pill" onclick={() => { query = tag }}>#{tag}</button>
          {/each}
        </div>
      {/if}
    </div>

  <!-- ── Results header (sticky, compact) ──────────────────────── -->
  {:else}
    <header class="topbar">
      <button class="wordmark-sm" onclick={() => { query = '' }}>mem</button>

      <div class="search-wrap">
        <svg class="search-icon" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.5"/>
          <line x1="10.5" y1="10.5" x2="14" y2="14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <input
          bind:this={resultsInput}
          class="search"
          type="search"
          value={query}
          oninput={(e) => { query = e.currentTarget.value }}
        />
        <button class="clear" onclick={() => { query = '' }} aria-label="clear">✕</button>
      </div>

      <div class="actions">
        <button class="action new-btn" onclick={onnew} aria-label="new note">
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true"><line x1="8" y1="2" x2="8" y2="14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><line x1="2" y1="8" x2="14" y2="8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
        </button>
        <button class="action icon-btn" onclick={toggleTheme} aria-label="toggle theme">
          {#if isDark}
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="3" stroke="currentColor" stroke-width="1.5"/><line x1="8" y1="1" x2="8" y2="3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="8" y1="13" x2="8" y2="15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="1" y1="8" x2="3" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="13" y1="8" x2="15" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="2.93" x2="4.34" y2="4.34" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="11.66" x2="13.07" y2="13.07" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="4.34" x2="13.07" y2="2.93" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="13.07" x2="4.34" y2="11.66" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
          {:else}
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M13.5 9a5.5 5.5 0 0 1-7-7 5.5 5.5 0 1 0 7 7Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>
          {/if}
        </button>
        <button class="action icon-btn" onclick={logout} aria-label="log out">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M6 3H3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><polyline points="10,5 13,8 10,11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="13" y1="8" x2="6" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
        </button>
      </div>
    </header>
  {/if}

  <!-- ── Timeline / results (always rendered) ───────────────────── -->
  <main class="content">
    {#if loading}
      <p class="status">loading…</p>
    {:else if error}
      <p class="status err">{error}</p>
    {:else if grouped.length === 0 && !hero}
      <p class="status">no notes matching <em>{query}</em></p>
    {:else}
      {#each grouped as [day, dayNotes] (day)}
        <section class="day">
          <div class="day-label">{day}</div>
          {#each dayNotes as note (note.id)}
            <button class="note-row" onclick={() => onopen(note.id)}>
              <span class="n-time">{time(note)}</span>
              <span class="n-slug">{note.slug}</span>
              {#if note.tags?.length}
                <span class="n-tags">{note.tags.map(t => '#' + t).join(' ')}</span>
              {/if}
            </button>
          {/each}
        </section>
      {/each}
    {/if}
  </main>
</div>

<style>
  .shell {
    display: flex;
    flex-direction: column;
    min-height: 100dvh;
  }

  /* ── Hero header ────────────────────────────────────────────────── */
  .hero-header {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.1rem;
    padding: 4rem 1.5rem 2rem;
    width: 100%;
    max-width: 640px;
    margin: 0 auto;
  }

  .corner-actions {
    position: absolute;
    top: 1.25rem;
    right: 0;
    display: flex;
    gap: 0.5rem;
  }

  .wordmark {
    font-size: 3rem;
    font-weight: 800;
    letter-spacing: -0.03em;
    line-height: 1;
    font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    color: transparent;
    background: linear-gradient(120deg, #a78bfa 0%, #22d3ee 100%);
    -webkit-background-clip: text;
    background-clip: text;
  }

  .prompt {
    color: var(--muted);
    font-weight: 400;
    margin-right: 0.15em;
    background: none;
    -webkit-text-fill-color: var(--muted);
  }

  .hero-search {
    font-size: 1.05rem;
    padding: 0.85rem 1.25rem 0.85rem 2.6rem !important;
    box-shadow: 0 2px 12px rgba(0,0,0,0.15);
  }

  /* ── Tag pills ──────────────────────────────────────────────────── */
  .tag-pills {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    justify-content: center;
  }

  .pill {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 2rem;
    color: var(--muted);
    cursor: pointer;
    font-size: 0.78rem;
    padding: 0.25rem 0.65rem;
    transition: color 0.1s, border-color 0.1s;
  }
  .pill:hover {
    color: var(--accent);
    border-color: var(--accent);
  }

  /* ── Results topbar ─────────────────────────────────────────────── */
  .topbar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 1.25rem;
    border-bottom: 1px solid var(--border);
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
  }

  .wordmark-sm {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 0.95rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    padding: 0;
    flex-shrink: 0;
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  /* ── Shared search wrap ─────────────────────────────────────────── */
  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
    flex: 1;
    min-width: 0;
  }

  .hero-header .search-wrap {
    flex: none;
    width: 100%;
  }

  .search-icon {
    position: absolute;
    left: 0.85rem;
    width: 14px;
    height: 14px;
    color: var(--muted);
    pointer-events: none;
  }

  .search {
    width: 100%;
    background: var(--surface);
    border: 1.5px solid var(--border);
    border-radius: 2rem;
    color: var(--text);
    font-size: 0.9rem;
    padding: 0.4rem 2rem 0.4rem 2.2rem;
    transition: border-color 0.12s, box-shadow 0.12s;
  }
  .search:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent);
  }

  .clear {
    position: absolute;
    right: 0.55rem;
    background: none;
    border: none;
    color: var(--muted);
    cursor: pointer;
    font-size: 0.7rem;
    padding: 0.15rem 0.3rem;
    border-radius: 3px;
  }
  .clear:hover { color: var(--text); background: var(--border) }

  /* ── Shared action buttons ──────────────────────────────────────── */
  .action {
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--muted);
    cursor: pointer;
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
    letter-spacing: 0.02em;
  }
  .action:hover { color: var(--text); border-color: var(--muted) }
  .icon-btn { display: flex; align-items: center; justify-content: center; padding: 0.3rem }
  .new-btn  { display: flex; align-items: center; justify-content: center; padding: 0.3rem; color: var(--accent); border-color: var(--accent) }

  /* ── Content / timeline ─────────────────────────────────────────── */
  .content {
    width: 100%;
    max-width: 640px;
    margin: 0 auto;
    padding: 0 1.5rem 4rem;
    flex: 1;
  }

  .day { margin-top: 2rem }

  .day-label {
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.07em;
    text-transform: uppercase;
    color: var(--muted);
    padding-bottom: 0.4rem;
    margin-bottom: 0.1rem;
    border-bottom: 1px solid var(--border);
  }

  .note-row {
    display: flex;
    align-items: baseline;
    gap: 0.85rem;
    width: 100%;
    background: none;
    border: none;
    border-radius: 6px;
    color: var(--text);
    cursor: pointer;
    padding: 0.5rem 0.5rem;
    text-align: left;
    font-size: 0.9rem;
  }
  .note-row:hover { background: var(--surface) }

  .n-time {
    color: var(--muted);
    font-size: 0.8rem;
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
    min-width: 3.2rem;
  }
  .n-slug  { flex-shrink: 0; font-weight: 500 }
  .n-tags  { color: var(--muted); font-size: 0.8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap }

  .status { color: var(--muted); padding: 3rem 0.5rem; font-size: 0.9rem }
  .status.err { color: #f85149 }
  .status em { font-style: normal; color: var(--text) }
</style>
