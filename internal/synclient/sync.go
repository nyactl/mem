// Package synclient implements the mem sync pull/push client.
// It speaks the server's REST API and writes plain note files locally.
package synclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mem-cli/internal/note"
)

// Client communicates with a mem serve instance.
type Client struct {
	base  string // e.g. "http://192.168.1.10:4747"
	token string
	http  *http.Client
	dir   string // local notes directory
}

// New creates a Client. base is the server URL, dir the local notes dir.
func New(base, token, dir string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
		dir:   dir,
	}
}

// ── sync state ────────────────────────────────────────────────────────────
// Tracks per-note ETags from the last sync so pull can distinguish
// "server changed, local untouched" from "both changed" (true conflict).
//
// We store both the server ETag and the local ETag observed right after
// writing the file. Comparing against the server ETag alone is insufficient:
// writeLocal reconstructs frontmatter from the JSON response, so the local
// file is rarely byte-identical to the server file — ETags diverge after
// every pull, producing false conflicts on the next one.

type noteState struct {
	ServerETag string `json:"server_etag"`
	LocalETag  string `json:"local_etag"`
}

type syncState struct {
	Notes    map[string]noteState `json:"notes"`
	LastSync string               `json:"last_sync,omitempty"`
}

func (c *Client) statePath() string {
	return filepath.Join(c.dir, ".mem-sync-state.json")
}

func (c *Client) loadState() syncState {
	data, err := os.ReadFile(c.statePath())
	if err != nil {
		return syncState{Notes: make(map[string]noteState)}
	}
	var s syncState
	if err := json.Unmarshal(data, &s); err != nil || s.Notes == nil {
		return syncState{Notes: make(map[string]noteState)}
	}
	return s
}

func (c *Client) saveState(s syncState) {
	s.LastSync = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.statePath(), data, 0600)
}

// ── remote note types (mirrors server's JSON) ─────────────────────────────

type remoteNote struct {
	ID      string   `json:"id"`
	Slug    string   `json:"slug"`
	Created string   `json:"created"`
	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`
	Body    string   `json:"body"`
	ETag    string   `json:"etag"`
}

// ── Pull ──────────────────────────────────────────────────────────────────

// Pull downloads all notes from the server that are missing or changed
// relative to the local store. Uses the sync state file to detect true
// conflicts (both sides changed) vs safe overwrites (only remote changed).
func (c *Client) Pull() (created, updated int, conflicts []string, err error) {
	if err = os.MkdirAll(c.dir, 0700); err != nil {
		return
	}

	state := c.loadState()

	remote, err := c.listRemote()
	if err != nil {
		return
	}

	for _, rn := range remote {
		localPath, localETag, exists := c.localInfo(rn.ID)
		ns := state.Notes[rn.ID]

		if !exists {
			full, fetchErr := c.getRemote(rn.ID)
			if fetchErr != nil {
				err = fetchErr
				return
			}
			if writeErr := c.writeLocal(full); writeErr != nil {
				err = writeErr
				return
			}
			localETagAfter, _ := note.ETag(localPath)
			state.Notes[rn.ID] = noteState{ServerETag: rn.ETag, LocalETag: localETagAfter}
			created++
			continue
		}

		serverChanged := rn.ETag != ns.ServerETag
		localChanged := ns.LocalETag != "" && localETag != ns.LocalETag

		if !serverChanged {
			// server unchanged since last sync; local edits handled by push
			continue
		}

		if localChanged {
			// both sides changed since last sync: true conflict
			conflicts = append(conflicts, rn.ID)
			continue
		}

		// only server changed: safe overwrite
		full, fetchErr := c.getRemote(rn.ID)
		if fetchErr != nil {
			err = fetchErr
			return
		}
		if writeErr := c.writeLocal(full); writeErr != nil {
			err = writeErr
			return
		}
		localETagAfter, _ := note.ETag(localPath)
		state.Notes[rn.ID] = noteState{ServerETag: rn.ETag, LocalETag: localETagAfter}
		updated++
	}

	c.saveState(state)
	return
}

// ── Push ──────────────────────────────────────────────────────────────────

// Push uploads local notes that the server does not have or that differ
// from the server's version.
func (c *Client) Push() (created, updated int, conflicts []string, err error) {
	local, err := note.List(c.dir)
	if err != nil {
		return
	}

	state := c.loadState()

	remote, err := c.listRemote()
	if err != nil {
		return
	}
	remoteByID := make(map[string]remoteNote, len(remote))
	for _, rn := range remote {
		remoteByID[rn.ID] = rn
	}

	for _, n := range local {
		localETag, _ := note.ETag(n.Path)
		rn, exists := remoteByID[n.ID()]

		if !exists {
			if pushErr := c.createRemote(n); pushErr != nil {
				err = pushErr
				return
			}
			created++
			continue
		}

		if localETag == rn.ETag {
			continue // identical content
		}

		if pushErr := c.updateRemote(n, rn.ETag); pushErr != nil {
			if isConflict(pushErr) {
				conflicts = append(conflicts, n.ID())
				continue
			}
			err = pushErr
			return
		}
		state.Notes[n.ID()] = noteState{ServerETag: localETag, LocalETag: localETag}
		updated++
	}

	c.saveState(state)
	return
}

// ── HTTP helpers ──────────────────────────────────────────────────────────

func (c *Client) req(method, path string, body any, extra map[string]string) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	return c.http.Do(req)
}

func (c *Client) listRemote() ([]remoteNote, error) {
	resp, err := c.req(http.MethodGet, "/api/notes", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list remote: server returned %d", resp.StatusCode)
	}
	var notes []remoteNote
	return notes, json.NewDecoder(resp.Body).Decode(&notes)
}

func (c *Client) getRemote(id string) (remoteNote, error) {
	resp, err := c.req(http.MethodGet, "/api/notes/"+url.PathEscape(id), nil, nil)
	if err != nil {
		return remoteNote{}, fmt.Errorf("get remote %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return remoteNote{}, fmt.Errorf("get remote %s: server returned %d", id, resp.StatusCode)
	}
	var n remoteNote
	return n, json.NewDecoder(resp.Body).Decode(&n)
}

func (c *Client) createRemote(n note.Note) error {
	raw, err := os.ReadFile(n.Path)
	if err != nil {
		return fmt.Errorf("read %s: %w", n.Path, err)
	}
	_, bodyText := splitFrontmatter(string(raw))
	payload := map[string]any{
		"title":   n.Slug,
		"body":    bodyText,
		"tags":    orEmpty(n.Tags),
		"sources": orEmpty(n.Sources),
		"created": n.Created.UTC().Format(time.RFC3339),
	}
	resp, err := c.req(http.MethodPost, "/api/notes", payload, nil)
	if err != nil {
		return fmt.Errorf("push create %s: %w", n.ID(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push create %s: server returned %d: %s", n.ID(), resp.StatusCode, body)
	}
	return nil
}

func (c *Client) updateRemote(n note.Note, serverETag string) error {
	raw, err := os.ReadFile(n.Path)
	if err != nil {
		return fmt.Errorf("read %s: %w", n.Path, err)
	}
	_, bodyText := splitFrontmatter(string(raw))
	payload := map[string]any{
		"body":    bodyText,
		"tags":    orEmpty(n.Tags),
		"sources": orEmpty(n.Sources),
	}
	resp, err := c.req(http.MethodPatch, "/api/notes/"+url.PathEscape(n.ID()), payload,
		map[string]string{"If-Match": serverETag})
	if err != nil {
		return fmt.Errorf("push update %s: %w", n.ID(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return conflictErr(n.ID())
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push update %s: server returned %d: %s", n.ID(), resp.StatusCode, body)
	}
	return nil
}

// ── local file helpers ────────────────────────────────────────────────────

func (c *Client) localInfo(id string) (path, etag string, exists bool) {
	// ID is "{timestamp}-{slug}" — exact match against filename (id + ".md")
	p := filepath.Join(c.dir, id+".md")
	et, err := note.ETag(p)
	if err != nil {
		return "", "", false
	}
	return p, et, true
}

func (c *Client) writeLocal(rn remoteNote) error {
	ts, err := time.ParseInLocation(time.RFC3339, rn.Created, time.UTC)
	if err != nil {
		return fmt.Errorf("parse created for %s: %w", rn.ID, err)
	}
	ts = ts.Local()

	slug := rn.Slug
	if slug == "" {
		slug = "note"
	}

	filename := note.Filename(ts, slug)
	path := filepath.Join(c.dir, filename)

	content := note.BuildFrontmatter(rn.Tags, rn.Sources, nil) + rn.Body
	return note.WriteRaw(path, content)
}

// ── utilities ─────────────────────────────────────────────────────────────

type conflictError struct{ id string }

func (e conflictError) Error() string { return "conflict: " + e.id }
func conflictErr(id string) error     { return conflictError{id} }
func isConflict(err error) bool       { _, ok := err.(conflictError); return ok }

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func splitFrontmatter(content string) (fm, body string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}
	idx := strings.Index(content[4:], "\n---\n")
	if idx < 0 {
		return "", content
	}
	end := 4 + idx + 5
	return content[:end], content[end:]
}
