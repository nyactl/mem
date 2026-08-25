package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mem/internal/gitops"
	"mem/internal/note"
	"mem/internal/web"
)

const maxBodyBytes = 1 << 20 // 1 MB

// Server hosts the sync REST API and serves the embedded PWA.
type Server struct {
	notesDir string
	token    string // empty = no auth required
	commitMu sync.Mutex
}

// New creates a Server that reads and writes notes in notesDir.
func New(notesDir, token string) *Server {
	return &Server{notesDir: notesDir, token: token}
}

// StartContext initialises the notes dir, ensures git, and listens on addr.
// It shuts down cleanly when ctx is cancelled.
func (s *Server) StartContext(ctx context.Context, addr string) error {
	if err := s.setup(); err != nil {
		return err
	}
	mux, err := s.buildMux()
	if err != nil {
		return err
	}
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

func (s *Server) setup() error {
	if err := os.MkdirAll(s.notesDir, 0700); err != nil {
		return fmt.Errorf("notes dir: %w", err)
	}
	if !gitops.IsRepo(s.notesDir) {
		if err := gitops.Init(s.notesDir); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}
	gitops.EnsureAuthor(s.notesDir)
	return nil
}

func (s *Server) buildMux() (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/notes", s.auth(s.handleNotes))
	mux.HandleFunc("/api/notes/", s.auth(s.handleNote))
	mux.HandleFunc("/api/delta", s.auth(s.handleDelta))
	mux.HandleFunc("/api/ingest", s.auth(s.handleIngest))

	sub, err := fs.Sub(web.FS, "dist")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	return mux, nil
}

// commit serialises git commits so concurrent writes don't race.
func (s *Server) commit(msg string) {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	_ = gitops.Commit(s.notesDir, msg)
}

// ── auth middleware ────────────────────────────────────────────────────────

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			next(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		// constant-time compare prevents timing-based token guessing
		if !eqConstant(got, s.token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// eqConstant compares two strings in constant time.
func eqConstant(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// ── API types ──────────────────────────────────────────────────────────────

type noteItem struct {
	ID      string   `json:"id"`
	Slug    string   `json:"slug"`
	Created string   `json:"created"`
	Date    string   `json:"date,omitempty"`
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
	Created string   `json:"created"` // RFC3339 UTC; use client time for offline capture
	Date    string   `json:"date,omitempty"`
}

type updateRequest struct {
	Body    string   `json:"body"`
	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`
	Date    string   `json:"date,omitempty"`
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

func (s *Server) listNotes(w http.ResponseWriter, _ *http.Request) {
	notes, err := note.List(s.notesDir)
	if err != nil {
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}
	items := make([]noteItem, 0, len(notes))
	for _, n := range notes {
		etag, _ := note.ETag(n.Path)
		items = append(items, toItem(n, etag))
	}
	jsonWrite(w, http.StatusOK, items)
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonWrite(w, http.StatusBadRequest, errBody(err))
		return
	}

	ts := time.Now()
	if req.Created != "" {
		if t, err := time.Parse(time.RFC3339, req.Created); err == nil {
			ts = t.Local() // filenames use local time
		}
	}

	slug := note.Slugify(req.Title)
	if slug == "" {
		slug = "note"
	}
	slug = uniqueSlug(s.notesDir, ts, slug)

	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.Sources, note.ExtractInlineSources(req.Body))
	var createDate *time.Time
	if req.Date != "" {
		if d, err := time.Parse(time.RFC3339, req.Date); err == nil {
			createDate = &d
		}
	}

	path := filepath.Join(s.notesDir, note.Filename(ts, slug))
	content := note.BuildFrontmatter(allTags, allSources, nil, createDate) + req.Body
	if err := note.WriteRaw(path, content); err != nil {
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}

	n, _ := note.Parse(path)
	etag, _ := note.ETag(path)

	w.Header().Set("Location", "/api/notes/"+n.ID())
	setETag(w, etag)
	jsonWrite(w, http.StatusCreated, toDetail(n, etag))

	go s.commit("note: add " + note.FormatTS(ts) + ifSlug(slug))
}

// ── GET /api/notes/{id}  PATCH /api/notes/{id}  DELETE /api/notes/{id} ───

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	id = strings.TrimSuffix(id, "/")
	if !isValidID(id) {
		jsonWrite(w, http.StatusBadRequest, errBody(fmt.Errorf("invalid note id %q", id)))
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
		jsonWrite(w, http.StatusNotFound, errBody(err))
		return
	}
	etag, _ := note.ETag(n.Path)
	setETag(w, etag)
	jsonWrite(w, http.StatusOK, toDetail(n, etag))
}

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request, id string) {
	n, err := note.FindByID(s.notesDir, id)
	if err != nil {
		jsonWrite(w, http.StatusNotFound, errBody(err))
		return
	}

	// optimistic concurrency: If-Match must match current ETag
	if clientETag := r.Header.Get("If-Match"); clientETag != "" {
		serverETag, _ := note.ETag(n.Path)
		if !eqConstant(clientETag, serverETag) {
			setETag(w, serverETag)
			jsonWrite(w, http.StatusConflict,
				errBody(fmt.Errorf("conflict: modified since etag %q", clientETag)))
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonWrite(w, http.StatusBadRequest, errBody(err))
		return
	}

	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.Sources, note.ExtractInlineSources(req.Body))
	var updateDate *time.Time
	if req.Date != "" {
		if d, err := time.Parse(time.RFC3339, req.Date); err == nil {
			updateDate = &d
		}
	} else {
		updateDate = n.Date
	}
	content := note.BuildFrontmatter(allTags, allSources, n.Attachments, updateDate) + req.Body
	if err := note.WriteRaw(n.Path, content); err != nil {
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}

	n2, _ := note.Parse(n.Path)
	etag, _ := note.ETag(n.Path)
	setETag(w, etag)
	jsonWrite(w, http.StatusOK, toDetail(n2, etag))

	go s.commit("note: update " + id + ifSlug(n.Slug))
}

func (s *Server) deleteNote(w http.ResponseWriter, _ *http.Request, id string) {
	n, err := note.FindByID(s.notesDir, id)
	if err != nil {
		jsonWrite(w, http.StatusNotFound, errBody(err))
		return
	}
	if err := os.Remove(n.Path); err != nil {
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
	go s.commit("note: delete " + id + ifSlug(n.Slug))
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
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}

	items := make([]noteItem, 0) // never return null
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
		items = append(items, toItem(n, etag))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Created > items[j].Created })
	jsonWrite(w, http.StatusOK, items)
}

// ── POST /api/ingest ──────────────────────────────────────────────────────

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonWrite(w, http.StatusBadRequest, errBody(err))
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

	allTags := note.MergeTags(req.Tags, note.ExtractInlineTags(req.Body))
	allSources := note.MergeSources(req.From, note.ExtractInlineSources(req.Body)) // from → sources

	path := filepath.Join(s.notesDir, note.Filename(ts, slug))
	content := note.BuildFrontmatter(allTags, allSources, nil, nil) + req.Body
	if err := note.WriteRaw(path, content); err != nil {
		jsonWrite(w, http.StatusInternalServerError, errBody(err))
		return
	}

	n, _ := note.Parse(path)
	etag, _ := note.ETag(path)

	w.Header().Set("Location", "/api/notes/"+n.ID())
	jsonWrite(w, http.StatusCreated, toDetail(n, etag))

	go s.commit("ingest: " + note.FormatTS(ts) + ifSlug(slug))
}

// ── helpers ────────────────────────────────────────────────────────────────

// jsonWrite sets Content-Type, writes status, then encodes v as JSON.
// All three happen in order before any body bytes are written.
func jsonWrite(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func errBody(err error) map[string]string { return map[string]string{"error": err.Error()} }

func toItem(n note.Note, etag string) noteItem {
	item := noteItem{
		ID:      n.ID(),
		Slug:    n.Slug,
		Created: n.Created.UTC().Format(time.RFC3339),
		Tags:    orEmpty(n.Tags),
		Sources: orEmpty(n.Sources),
		ETag:    etag,
	}
	if n.Date != nil {
		item.Date = n.Date.UTC().Format(time.RFC3339)
	}
	return item
}

func toDetail(n note.Note, etag string) noteDetail {
	return noteDetail{noteItem: toItem(n, etag), Body: n.Body}
}

// setETag writes an ETag header preserving RFC 7232 casing ("ETag", not "Etag").
// Go's textproto.CanonicalMIMEHeaderKey would convert "ETag" → "Etag", so we
// write directly to the underlying map to bypass canonicalisation.
func setETag(w http.ResponseWriter, etag string) {
	w.Header()["ETag"] = []string{etag}
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

// isValidID accepts note IDs of the form "{timestamp}-{slug}" where timestamp
// is the 15-char YYYYMMDDTHHMMSS string and slug is one or more lowercase
// alphanumeric/hyphen chars. Rejects empty slugs, path traversal, and
// malformed timestamps.
func isValidID(id string) bool {
	if len(id) < 17 { // minimum: 15-char ts + "-" + 1-char slug
		return false
	}
	if id[15] != '-' {
		return false
	}
	if _, err := time.Parse("20060102T150405", id[:15]); err != nil {
		return false
	}
	slug := id[16:]
	if slug == "" {
		return false
	}
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

// uniqueSlug appends a numeric suffix if a file with that slug already exists
// at the given timestamp second.
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
