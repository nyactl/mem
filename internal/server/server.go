package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mem-cli/internal/gitops"
	"mem-cli/internal/note"
)

//go:embed pwa/*
var pwaFiles embed.FS

// Server hosts the sync REST API and serves the embedded PWA.
type Server struct {
	notesDir string
	token    string // empty = no auth required
}

// New creates a Server that reads and writes notes in notesDir.
func New(notesDir, token string) *Server {
	return &Server{notesDir: notesDir, token: token}
}

// Start initialises the notes dir, ensures git, and listens on addr.
func (s *Server) Start(addr string) error {
	if err := os.MkdirAll(s.notesDir, 0700); err != nil {
		return fmt.Errorf("notes dir: %w", err)
	}
	if !gitops.IsRepo(s.notesDir) {
		if err := gitops.Init(s.notesDir); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}
	gitops.EnsureAuthor(s.notesDir)

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/notes", s.auth(s.handleNotes))
	mux.HandleFunc("/api/notes/", s.auth(s.handleNote))
	mux.HandleFunc("/api/delta", s.auth(s.handleDelta))
	mux.HandleFunc("/api/ingest", s.auth(s.handleIngest))

	// PWA — strip the "pwa/" prefix so /app.js works
	sub, err := fs.Sub(pwaFiles, "pwa")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	log.Printf("mem serve  %s  notes=%s", addr, s.notesDir)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
}

// StartContext is like Start but honours context cancellation.
func (s *Server) StartContext(ctx context.Context, addr string) error {
	if err := os.MkdirAll(s.notesDir, 0700); err != nil {
		return fmt.Errorf("notes dir: %w", err)
	}
	if !gitops.IsRepo(s.notesDir) {
		if err := gitops.Init(s.notesDir); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}
	gitops.EnsureAuthor(s.notesDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/notes", s.auth(s.handleNotes))
	mux.HandleFunc("/api/notes/", s.auth(s.handleNote))
	mux.HandleFunc("/api/delta", s.auth(s.handleDelta))
	mux.HandleFunc("/api/ingest", s.auth(s.handleIngest))

	sub, err := fs.Sub(pwaFiles, "pwa")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	log.Printf("mem serve  %s  notes=%s", addr, s.notesDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ── auth middleware ────────────────────────────────────────────────────────

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			next(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ── API types ──────────────────────────────────────────────────────────────

type noteItem struct {
	ID      string   `json:"id"`
	Slug    string   `json:"slug"`
	Created string   `json:"created"`
	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`
	ETag    string   `json:"etag"`
}

type noteDetail struct {
	noteItem
	Body string `json:"body"`
}

type createRequest struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`
	Created string   `json:"created"` // ISO8601; use client time for offline capture
}

type updateRequest struct {
	Body    string   `json:"body"`
	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`
}

type ingestRequest struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Tags    []string `json:"tags"`
	From    []string `json:"from"` // stored as sources for compatibility
	Created string   `json:"created"`
}

// ── GET /api/notes  POST /api/notes ───────────────────────────────────────

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listNotes(w, r)
	case http.MethodPost:
		s.createNote(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := note.List(s.notesDir)
	if err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}
	items := make([]noteItem, 0, len(notes))
	for _, n := range notes {
		etag, _ := note.ETag(n.Path)
		items = append(items, noteItem{
			ID:      n.ID(),
			Slug:    n.Slug,
			Created: n.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n.Tags),
			Sources: orEmpty(n.Sources),
			ETag:    etag,
		})
	}
	jsonOK(w, items)
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err, http.StatusBadRequest)
		return
	}

	ts := time.Now()
	if req.Created != "" {
		if t, err := time.Parse(time.RFC3339, req.Created); err == nil {
			ts = t.Local() // filename timestamps are always in local time
		}
	}

	slug := note.Slugify(req.Title)
	if slug == "" {
		slug = "note"
	}

	// avoid collisions — append a counter suffix if needed
	slug = uniqueSlug(s.notesDir, ts, slug)

	// merge inline tags/sources from body with explicit ones
	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.Sources, note.ExtractInlineSources(req.Body))

	filename := note.Filename(ts, slug)
	path := filepath.Join(s.notesDir, filename)

	content := note.BuildFrontmatter(allTags, allSources, nil) + req.Body
	if err := note.WriteRaw(path, content); err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}

	go func() {
		_ = gitops.Commit(s.notesDir, "note: add "+note.FormatTS(ts)+ifSlug(slug))
	}()

	n, _ := note.Parse(path)
	etag, _ := note.ETag(path)
	w.Header().Set("Location", "/api/notes/"+n.ID())
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, noteDetail{
		noteItem: noteItem{
			ID:      n.ID(),
			Slug:    n.Slug,
			Created: n.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n.Tags),
			Sources: orEmpty(n.Sources),
			ETag:    etag,
		},
		Body: n.Body,
	})
}

// ── GET /api/notes/{id}  PATCH /api/notes/{id}  DELETE /api/notes/{id} ───

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		http.Error(w, "missing note id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getNote(w, r, id)
	case http.MethodPatch:
		s.updateNote(w, r, id)
	case http.MethodDelete:
		s.deleteNote(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getNote(w http.ResponseWriter, _ *http.Request, id string) {
	n, err := note.FindByID(s.notesDir, id)
	if err != nil {
		jsonErr(w, err, http.StatusNotFound)
		return
	}
	etag, _ := note.ETag(n.Path)
	w.Header().Set("ETag", etag)
	jsonOK(w, noteDetail{
		noteItem: noteItem{
			ID:      n.ID(),
			Slug:    n.Slug,
			Created: n.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n.Tags),
			Sources: orEmpty(n.Sources),
			ETag:    etag,
		},
		Body: n.Body,
	})
}

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request, id string) {
	n, err := note.FindByID(s.notesDir, id)
	if err != nil {
		jsonErr(w, err, http.StatusNotFound)
		return
	}

	// optimistic concurrency: If-Match must match current ETag
	clientETag := r.Header.Get("If-Match")
	if clientETag != "" {
		serverETag, _ := note.ETag(n.Path)
		if clientETag != serverETag {
			w.Header().Set("ETag", serverETag)
			jsonErr(w, fmt.Errorf("conflict: note was modified since etag %q", clientETag), http.StatusConflict)
			return
		}
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err, http.StatusBadRequest)
		return
	}

	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.Sources, note.ExtractInlineSources(req.Body))

	content := note.BuildFrontmatter(allTags, allSources, n.Attachments) + req.Body
	if err := note.WriteRaw(n.Path, content); err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}

	go func() {
		_ = gitops.Commit(s.notesDir, "note: update "+id+ifSlug(n.Slug))
	}()

	etag, _ := note.ETag(n.Path)
	w.Header().Set("ETag", etag)
	n2, _ := note.Parse(n.Path)
	jsonOK(w, noteDetail{
		noteItem: noteItem{
			ID:      n2.ID(),
			Slug:    n2.Slug,
			Created: n2.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n2.Tags),
			Sources: orEmpty(n2.Sources),
			ETag:    etag,
		},
		Body: n2.Body,
	})
}

func (s *Server) deleteNote(w http.ResponseWriter, _ *http.Request, id string) {
	n, err := note.FindByID(s.notesDir, id)
	if err != nil {
		jsonErr(w, err, http.StatusNotFound)
		return
	}
	if err := os.Remove(n.Path); err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}
	go func() {
		_ = gitops.Commit(s.notesDir, "note: delete "+id+ifSlug(n.Slug))
	}()
	w.WriteHeader(http.StatusNoContent)
}

// ── GET /api/delta?since=RFC3339 ──────────────────────────────────────────

func (s *Server) handleDelta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var since time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			since = t
		}
	}

	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}

	var items []noteItem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !since.IsZero() && !info.ModTime().After(since) {
			continue
		}
		n, err := note.Parse(filepath.Join(s.notesDir, e.Name()))
		if err != nil {
			continue
		}
		etag, _ := note.ETag(n.Path)
		items = append(items, noteItem{
			ID:      n.ID(),
			Slug:    n.Slug,
			Created: n.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n.Tags),
			Sources: orEmpty(n.Sources),
			ETag:    etag,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Created > items[j].Created })
	jsonOK(w, items)
}

// ── POST /api/ingest ──────────────────────────────────────────────────────

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err, http.StatusBadRequest)
		return
	}

	ts := time.Now()
	if req.Created != "" {
		if t, err := time.Parse(time.RFC3339, req.Created); err == nil {
			ts = t.Local()
		}
	}

	slug := note.Slugify(req.Title)
	if slug == "" {
		slug = "ingest"
	}
	slug = uniqueSlug(s.notesDir, ts, slug)

	// from → sources for storage compatibility
	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.From, note.ExtractInlineSources(req.Body))

	filename := note.Filename(ts, slug)
	path := filepath.Join(s.notesDir, filename)

	content := note.BuildFrontmatter(allTags, allSources, nil) + req.Body
	if err := note.WriteRaw(path, content); err != nil {
		jsonErr(w, err, http.StatusInternalServerError)
		return
	}

	go func() {
		_ = gitops.Commit(s.notesDir, "ingest: "+note.FormatTS(ts)+ifSlug(slug))
	}()

	n, _ := note.Parse(path)
	etag, _ := note.ETag(path)
	w.Header().Set("Location", "/api/notes/"+n.ID())
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, noteDetail{
		noteItem: noteItem{
			ID:      n.ID(),
			Slug:    n.Slug,
			Created: n.Created.UTC().Format(time.RFC3339),
			Tags:    orEmpty(n.Tags),
			Sources: orEmpty(n.Sources),
			ETag:    etag,
		},
		Body: n.Body,
	})
}

// ── helpers ────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func ifSlug(slug string) string {
	if slug == "" {
		return ""
	}
	return " " + slug
}

// uniqueSlug appends a numeric suffix if a file with that slug already exists
// at the given timestamp.
func uniqueSlug(dir string, ts time.Time, slug string) string {
	candidate := slug
	for i := 2; ; i++ {
		path := filepath.Join(dir, note.Filename(ts, candidate))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", slug, i)
	}
}
