package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mem/internal/note"
	"mem/internal/synclient"
)

func post(t *testing.T, method, url string, body any) (int, string) {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// The server refuses a note it would store as something else or that would spread a broken note,
// and says why; it stores nothing for it.
func TestServerRefusesMalformedNotes(t *testing.T) {
	srv, serverDir := testServer(t)
	for _, c := range []struct {
		req  map[string]any
		want string
	}{
		{map[string]any{"title": "a", "body": "b", "tags": []string{"Kafka Streams"}}, `tag \"Kafka Streams\" is not a slug`},
		{map[string]any{"title": "a", "body": "b", "sources": []string{"Some Book"}}, `source \"Some Book\" is not a slug`},
		{map[string]any{"title": "a", "body": "b", "date": "last tuesday"}, "date"},
		{map[string]any{"title": "a", "body": "b", "created": "yesterday"}, "created"},
		{map[string]any{"title": "a", "body": "---\ntags: [kb]\n\nbody\n"}, "closing --- is missing"},
	} {
		code, body := post(t, http.MethodPost, srv.URL+"/api/notes", c.req)
		if code != http.StatusBadRequest || !strings.Contains(body, c.want) {
			t.Errorf("%v: %d %s, want 400 with %q", c.req, code, body, c.want)
		}
	}
	if entries, _ := filepath.Glob(filepath.Join(serverDir, "*.md")); len(entries) != 0 {
		t.Errorf("refused notes were stored: %v", entries)
	}

	code, body := post(t, http.MethodPost, srv.URL+"/api/notes", map[string]any{"title": "ok", "body": "fine", "tags": []string{"kb"}})
	if code != http.StatusCreated {
		t.Fatalf("good note: %d %s", code, body)
	}
	var created struct{ ID string }
	_ = json.Unmarshal([]byte(body), &created)
	if code, body := post(t, http.MethodPatch, srv.URL+"/api/notes/"+created.ID, map[string]any{"body": "x", "tags": []string{"Bad Tag"}}); code != http.StatusBadRequest {
		t.Errorf("update with a bad tag: %d %s", code, body)
	}
	if n, _ := note.FindByID(serverDir, created.ID); n.Body != "fine" {
		t.Errorf("refused update changed the note: %q", n.Body)
	}
}

// Integrations name tags and sources as they like: ingest makes them slugs instead of refusing.
func TestIngestSlugsTagsAndSources(t *testing.T) {
	srv, serverDir := testServer(t)
	code, body := post(t, http.MethodPost, srv.URL+"/api/ingest", map[string]any{"title": "x", "body": "b", "tags": []string{"Read Later"}, "from": []string{"Todoist"}})
	if code != http.StatusCreated {
		t.Fatalf("%d %s", code, body)
	}
	notes, _ := note.List(serverDir)
	if len(notes) != 1 || strings.Join(notes[0].Tags, ",") != "read-later" || strings.Join(notes[0].Sources, ",") != "todoist" {
		t.Errorf("ingested %+v", notes)
	}
}

// A broken note stays on the machine it was made on, says why, and the other notes still sync.
func TestPushRefusesBrokenNoteAndSyncsTheRest(t *testing.T) {
	srv, serverDir := testServer(t)
	a := t.TempDir()
	ts := time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local)
	broken := note.FilePath(a, ts, "broken")
	if err := os.WriteFile(broken, []byte("---\ntags: [kb]\n\nthe body\n"), 0600); err != nil {
		t.Fatal(err)
	}
	good := note.FilePath(a, ts.Add(time.Second), "good")
	if err := note.WriteRaw(good, note.BuildFrontmatter("", []string{"kb"}, nil, nil, nil)+"fine\n"); err != nil {
		t.Fatal(err)
	}

	created, _, _, _, refused, err := synclient.New(srv.URL, "", a).Push()
	if err != nil || created != 1 {
		t.Fatalf("created %d: %v", created, err)
	}
	if len(refused) != 1 || refused[0].Path != broken || !strings.Contains(refused[0].Msg, "never closed") {
		t.Errorf("refused %v", refused)
	}
	if note.Exists(filepath.Join(serverDir, filepath.Base(broken))) {
		t.Error("broken note reached the server")
	}
	if !note.Exists(filepath.Join(serverDir, filepath.Base(good))) {
		t.Error("good note did not reach the server")
	}

	// Once synced, a note broken later is refused as an update, and is not deleted on the server.
	if err := os.WriteFile(good, []byte("---\ntags: [Kafka Streams]\n---\nfine, edited\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, updated, deleted, _, refused, err := synclient.New(srv.URL, "", a).Push()
	if err != nil || updated != 0 || deleted != 0 || len(refused) != 2 {
		t.Errorf("updated %d deleted %d refused %v: %v", updated, deleted, refused, err)
	}
	if n, _ := note.Parse(filepath.Join(serverDir, filepath.Base(good))); n.Body != "fine\n" && n.Body != "fine" {
		t.Errorf("server note changed to %q", n.Body)
	}
}
