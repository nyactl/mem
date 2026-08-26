<script>
  import { api } from '../api.js'
  import { marked } from 'marked'
  import DOMPurify from 'dompurify'
  import { tick } from 'svelte'

  let { id, onback, onedit, theme, toggleTheme } = $props()
  let isDark = $derived(theme !== 'light')

  let note    = $state(null)
  let loading = $state(true)
  let error   = $state('')
  let raw     = $state(false)
  let bodyEl  = $state(null)

  marked.use({
    breaks: true,
    renderer: {
      code(token) {
        if (token.lang === 'mermaid') {
          return `<div class="mermaid">${token.text}</div>`
        }
        return false
      }
    }
  })

  async function load() {
    loading = true
    error   = ''
    note    = null
    try { note = await api.note(id) }
    catch (e) { error = e.message }
    finally { loading = false }
  }

  $effect(() => { id; load() })

  let html = $derived.by(() => {
    if (!note?.body) return ''
    return DOMPurify.sanitize(marked.parse(note.body))
  })

  async function renderMermaid() {
    const nodes = bodyEl?.querySelectorAll('.mermaid')
    if (!nodes?.length) return
    const mermaidTheme = isDark ? 'dark' : 'neutral'
    const { default: mermaid } = await import('mermaid')
    mermaid.initialize({ startOnLoad: false, theme: mermaidTheme })
    nodes.forEach(n => { n.removeAttribute('data-processed'); n.innerHTML = n.textContent ?? '' })
    mermaid.run({ nodes })
  }

  $effect(() => {
    if (!raw && html) tick().then(renderMermaid)
  })

  function formatDate(iso) {
    if (!iso) return ''
    return new Date(iso).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
  }

  let displayDate = $derived(note ? (note.date || note.created) : null)
</script>

<div class="shell">
  <header>
    <button class="back" onclick={onback}>← back</button>
    {#if note}
      <span class="slug">{note.slug}</span>
    {/if}
    <div class="spacer"></div>
    {#if note}
      <button class="action" onclick={onedit} aria-label="edit note">edit</button>
    {/if}
    <button class="action" class:active={raw} onclick={() => raw = !raw} aria-label="toggle raw">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
        <polyline points="5,4 1,8 5,12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        <polyline points="11,4 15,8 11,12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </button>
    <button class="action icon-btn" onclick={toggleTheme} aria-label="toggle theme">
      {#if isDark}
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="3" stroke="currentColor" stroke-width="1.5"/><line x1="8" y1="1" x2="8" y2="3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="8" y1="13" x2="8" y2="15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="1" y1="8" x2="3" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="13" y1="8" x2="15" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="2.93" x2="4.34" y2="4.34" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="11.66" x2="13.07" y2="13.07" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="11.66" y1="4.34" x2="13.07" y2="2.93" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><line x1="2.93" y1="13.07" x2="4.34" y2="11.66" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M13.5 9a5.5 5.5 0 0 1-7-7 5.5 5.5 0 1 0 7 7Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>
      {/if}
    </button>
  </header>

  <main>
    {#if loading}
      <p class="status">loading…</p>
    {:else if error}
      <p class="status error">{error}</p>
    {:else if note}
      <div class="meta">
        <span class="date">{formatDate(displayDate)}</span>
        {#each (note.tags || []) as tag}
          <span class="tag">#{tag}</span>
        {/each}
      </div>
      {#if raw}
        <pre class="raw-body">{note.body}</pre>
      {:else}
        <div class="md-body" bind:this={bodyEl}>{@html html}</div>
      {/if}
    {/if}
  </main>
</div>

<style>
  .shell { display: flex; flex-direction: column; min-height: 100dvh }

  header {
    position: sticky;
    top: 0;
    z-index: 10;
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 1.25rem;
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
  .spacer { flex: 1 }
  .action {
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--muted);
    cursor: pointer;
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
  }
  .action:hover { color: var(--text); border-color: var(--muted) }
  .action.active { color: var(--accent); border-color: var(--accent) }
  .icon-btn { display: flex; align-items: center; justify-content: center; padding: 0.3rem }
  .slug { color: var(--accent); font-size: 0.9rem; font-weight: 600; letter-spacing: 0.05em; overflow: hidden; text-overflow: ellipsis; white-space: nowrap }

  main { padding: 1.25rem 1.5rem 3rem; max-width: 720px; margin: 0 auto; width: 100% }

  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
  }
  .date { color: var(--muted); font-size: 0.82rem }
  .tag  { color: var(--muted); font-size: 0.82rem }

  .raw-body {
    white-space: pre-wrap;
    word-break: break-word;
    font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 0.88rem;
    line-height: 1.65;
    color: var(--text);
  }

  /* ── Markdown rendered output ───────────────────────────────────── */
  .md-body {
    font-size: 0.95rem;
    line-height: 1.7;
    color: var(--text);
  }

  .md-body :global(h1),
  .md-body :global(h2),
  .md-body :global(h3),
  .md-body :global(h4) {
    color: var(--text);
    font-weight: 600;
    line-height: 1.3;
    margin: 1.5em 0 0.5em;
  }
  .md-body :global(h1) { font-size: 1.4rem }
  .md-body :global(h2) { font-size: 1.15rem }
  .md-body :global(h3) { font-size: 1rem }

  .md-body :global(p)  { margin: 0.75em 0 }
  .md-body :global(ul),
  .md-body :global(ol) { padding-left: 1.5em; margin: 0.75em 0 }
  .md-body :global(li) { margin: 0.25em 0 }

  .md-body :global(a)  { color: var(--accent); text-decoration: none }
  .md-body :global(a:hover) { text-decoration: underline }

  .md-body :global(blockquote) {
    border-left: 3px solid var(--border);
    margin: 0.75em 0;
    padding: 0.25em 0 0.25em 1em;
    color: var(--muted);
  }

  .md-body :global(code) {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 3px;
    font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 0.85em;
    padding: 0.1em 0.35em;
  }

  .md-body :global(pre) {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 0.85rem;
    line-height: 1.55;
    margin: 1em 0;
    overflow-x: auto;
    padding: 1em 1.25em;
  }
  .md-body :global(pre code) {
    background: none;
    border: none;
    padding: 0;
    font-size: inherit;
  }

  .md-body :global(table) {
    border-collapse: collapse;
    font-size: 0.88rem;
    margin: 1em 0;
    overflow-x: auto;
    display: block;
    width: max-content;
    max-width: 100%;
  }
  .md-body :global(th),
  .md-body :global(td) {
    border: 1px solid var(--border);
    padding: 0.4em 0.75em;
  }
  .md-body :global(th) {
    background: var(--surface);
    font-weight: 600;
    text-align: left;
  }

  .md-body :global(hr) {
    border: none;
    border-top: 1px solid var(--border);
    margin: 1.5em 0;
  }

  /* Mermaid diagrams */
  .md-body :global(.mermaid) {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    margin: 1em 0;
    overflow-x: auto;
    padding: 1em;
    text-align: center;
  }
  .md-body :global(.mermaid svg) { max-width: 100% }

  .status { color: var(--muted); padding: 2rem 0.5rem; font-size: 0.9rem }
  .status.error { color: #f85149 }
</style>
