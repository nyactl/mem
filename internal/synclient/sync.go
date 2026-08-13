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
	base   string // e.g. "http://192.168.1.10:4747"
	token  string
	http   *http.Client
	dir    string // local notes directory
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
// compared to the local store. Returns counts of created and updated notes.
func (c *Client) Pull() (created, updated int, conflicts []string, err error) {
	if err = os.MkdirAll(c.dir, 0700); err != nil {
		return
	}

	remote, err := c.listRemote()
	if err != nil {
		return
	}

	for _, rn := range remote {
		localPath, localETag, exists := c.localInfo(rn.ID)
		if !exists {
			// fetch full note and write locally
			full, fetchErr := c.getRemote(rn.ID)
			if fetchErr != nil {
				err = fetchErr
				return
			}
			if writeErr := c.writeLocal(full); writeErr != nil {
				err = writeErr
				return
			}
			created++
			continue
		}

		// already exists locally
		if rn.ETag == localETag {
			continue // identical content, skip
		}

		// ETags differ — check if local was also modified (conflict)
		localETagNow, _ := note.ETag(localPath)
		if localETagNow != localETag {
			// both local and remote changed since last sync: conflict
			conflicts = append(conflicts, rn.ID)
			continue
		}

		// only remote changed: safe to overwrite
		full, fetchErr := c.getRemote(rn.ID)
		if fetchErr != nil {
			err = fetchErr
			return
		}
		if writeErr := c.writeLocal(full); writeErr != nil {
			err = writeErr
			return
		}
		updated++
	}
	return
}

// ── Push ──────────────────────────────────────────────────────────────────

// Push uploads local notes that the server does not have or has an older
// version of. Returns counts of created and updated notes on the server.
func (c *Client) Push() (created, updated int, conflicts []string, err error) {
	local, err := note.List(c.dir)
	if err != nil {
		return
	}

	remote, err := c.listRemote()
	if err != nil {
		return
	}
	remoteMap := make(map[string]string, len(remote)) // id → etag
	for _, rn := range remote {
		remoteMap[rn.ID] = rn.ETag
	}

	for _, n := range local {
		localETag, _ := note.ETag(n.Path)
		remoteETag, exists := remoteMap[n.ID()]

		if !exists {
			if pushErr := c.createRemote(n); pushErr != nil {
				err = pushErr
				return
			}
			created++
			continue
		}

		if localETag == remoteETag {
			continue // in sync
		}

		// try to update; server will 409 if it was concurrently modified
		if pushErr := c.updateRemote(n, remoteETag); pushErr != nil {
			if isConflict(pushErr) {
				conflicts = append(conflicts, n.ID())
				continue
			}
			err = pushErr
			return
		}
		updated++
	}
	return
}

// ── HTTP helpers ──────────────────────────────────────────────────────────

func (c *Client) req(method, path string, body any) (*http.Response, error) {
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
	return c.http.Do(req)
}

func (c *Client) listRemote() ([]remoteNote, error) {
	resp, err := c.req(http.MethodGet, "/api/notes", nil)
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
	resp, err := c.req(http.MethodGet, "/api/notes/"+url.PathEscape(id), nil)
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
	body, _ := io.ReadAll(mustOpen(n.Path))
	_, bodyText := splitFrontmatter(string(body))
	payload := map[string]any{
		"title":   n.Slug,
		"body":    bodyText,
		"tags":    n.Tags,
		"sources": n.Sources,
		"created": n.Created.UTC().Format(time.RFC3339),
	}
	resp, err := c.req(http.MethodPost, "/api/notes", payload)
	if err != nil {
		return fmt.Errorf("push create %s: %w", n.ID(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("push create %s: server returned %d", n.ID(), resp.StatusCode)
	}
	return nil
}

func (c *Client) updateRemote(n note.Note, serverETag string) error {
	raw, err := os.ReadFile(n.Path)
	if err != nil {
		return err
	}
	_, bodyText := splitFrontmatter(string(raw))
	payload := map[string]any{
		"body":    bodyText,
		"tags":    n.Tags,
		"sources": n.Sources,
	}
	req, err := http.NewRequest(http.MethodPatch, c.base+"/api/notes/"+url.PathEscape(n.ID()), mustJSON(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", serverETag)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("push update %s: %w", n.ID(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return conflictErr(n.ID())
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("push update %s: server returned %d", n.ID(), resp.StatusCode)
	}
	return nil
}

// ── local file helpers ────────────────────────────────────────────────────

func (c *Client) localInfo(id string) (path, etag string, exists bool) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return "", "", false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id) && strings.HasSuffix(e.Name(), ".md") {
			p := filepath.Join(c.dir, e.Name())
			et, _ := note.ETag(p)
			return p, et, true
		}
	}
	return "", "", false
}

func (c *Client) writeLocal(rn remoteNote) error {
	ts, err := time.ParseInLocation(time.RFC3339, rn.Created, time.UTC)
	if err != nil {
		return fmt.Errorf("parse created time for %s: %w", rn.ID, err)
	}
	ts = ts.Local()

	slug := rn.Slug
	if slug == "" {
		slug = "note"
	}

	filename := note.Filename(ts, slug)
	path := filepath.Join(c.dir, filename)

	// if a file with this timestamp exists but different slug (rename), remove old
	entries, _ := os.ReadDir(c.dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), rn.ID) && e.Name() != filename {
			_ = os.Remove(filepath.Join(c.dir, e.Name()))
		}
	}

	content := note.BuildFrontmatter(rn.Tags, rn.Sources, nil) + rn.Body
	return note.WriteRaw(path, content)
}

// ── small utilities ───────────────────────────────────────────────────────

type conflictError struct{ id string }

func (e conflictError) Error() string { return "conflict: " + e.id }
func conflictErr(id string) error     { return conflictError{id} }
func isConflict(err error) bool       { _, ok := err.(conflictError); return ok }

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

func mustOpen(path string) io.Reader {
	f, err := os.Open(path)
	if err != nil {
		return strings.NewReader("")
	}
	return f
}

func mustJSON(v any) io.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}
