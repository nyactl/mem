package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mem/internal/note"
	"mem/internal/synclient"
)

func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	mux, err := New(dir, "").buildMux()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, dir
}

// A title travels with its note through the server to another machine byte for byte, so neither
// side sees a change on the next sync.
func TestTitleSyncsBetweenMachines(t *testing.T) {
	srv, serverDir := testServer(t)
	a, b := t.TempDir(), t.TempDir()
	ts := time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local)
	path := note.FilePath(a, ts, "c")
	written := note.BuildFrontmatter("C++: templates", []string{"glossary"}, nil, nil, nil) + "a language\n"
	if err := note.WriteRaw(path, written); err != nil {
		t.Fatal(err)
	}
	old := note.FilePath(a, ts.Add(time.Second), "kafka-rebalance")
	if err := note.WriteRaw(old, "---\ntags: [kb]\n---\nno title\n"); err != nil {
		t.Fatal(err)
	}

	if created, _, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil || created != 2 {
		t.Fatalf("push created %d: %v", created, err)
	}
	if got, _ := os.ReadFile(filepath.Join(serverDir, filepath.Base(path))); string(got) != written {
		t.Errorf("server file\n%s\nwant\n%s", got, written)
	}
	if got, _ := os.ReadFile(filepath.Join(serverDir, filepath.Base(old))); string(got) != "---\ntags: [kb]\n---\nno title\n" {
		t.Errorf("note without a title came out as\n%s", got)
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil || updated != 0 {
		t.Errorf("second push updated %d: %v", updated, err)
	}
	if created, _, _, err := synclient.New(srv.URL, "", b).Pull(); err != nil || created != 2 {
		t.Fatalf("pull created %d: %v", created, err)
	}
	if got, _ := os.ReadFile(filepath.Join(b, filepath.Base(path))); strings.TrimRight(string(got), "\n") != strings.TrimRight(written, "\n") {
		t.Errorf("pulled file\n%s\nwant\n%s", got, written)
	}

	fixed := note.BuildFrontmatter("C++ templates", []string{"glossary"}, nil, nil, nil) + "a language\n"
	if err := note.WriteRaw(path, fixed); err != nil {
		t.Fatal(err)
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil || updated != 1 {
		t.Fatalf("push of a corrected title updated %d: %v", updated, err)
	}
	if got, _ := os.ReadFile(filepath.Join(serverDir, filepath.Base(path))); string(got) != fixed {
		t.Errorf("server file after the correction %q, want %q", got, fixed)
	}
}

// Clients from before titles send none; their update must not drop the title, and their create
// sends the slug as title, which must not become the note's title.
func TestClientWithoutTitlesKeepsThem(t *testing.T) {
	srv, serverDir := testServer(t)
	send := func(method, path string, body any) *http.Response {
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	created := "2026-10-28T09:30:00Z"
	if r := send(http.MethodPost, "/api/notes", map[string]any{"title": "c", "body": "old client\n", "created": created}); r.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", r.StatusCode)
	}
	if r := send(http.MethodPost, "/api/notes", map[string]any{"title": "c-2", "display_title": "C++", "body": "new client\n", "created": created}); r.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", r.StatusCode)
	}
	id := note.FormatTS(time.Date(2026, 10, 28, 9, 30, 0, 0, time.UTC).Local())
	n, _ := note.FindByID(serverDir, id+"-c")
	if n.Title != "" {
		t.Errorf("old client's slug became the title %q", n.Title)
	}
	if r := send(http.MethodPatch, "/api/notes/"+id+"-c-2", map[string]any{"body": "edited\n", "tags": []string{}}); r.StatusCode != http.StatusOK {
		t.Fatalf("update: %d", r.StatusCode)
	}
	if n, _ := note.FindByID(serverDir, id+"-c-2"); n.Title != "C++" || n.Body != "edited" {
		t.Errorf("after an update without a title: title %q body %q", n.Title, n.Body)
	}
}

// A machine that pulled or edited a note with a client older than titles holds it without its
// title. Its next push takes the title back from the server instead of pushing the note forever.
func TestPushTakesBackTitleOlderClientDropped(t *testing.T) {
	srv, serverDir := testServer(t)
	a, b := t.TempDir(), t.TempDir()
	ts := time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local)
	name := filepath.Base(note.FilePath(a, ts, "c"))
	if err := note.WriteRaw(filepath.Join(a, name), note.BuildFrontmatter("C++", []string{"glossary"}, nil, nil, nil)+"a language\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := synclient.New(srv.URL, "", b).Pull(); err != nil {
		t.Fatal(err)
	}
	// what v0.5.0's mem edit leaves on b
	if err := note.WriteRaw(filepath.Join(b, name), "---\ntags: [glossary]\n---\na language, edited\n"); err != nil {
		t.Fatal(err)
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", b).Push(); err != nil || updated != 1 {
		t.Fatalf("push updated %d: %v", updated, err)
	}
	want := note.BuildFrontmatter("C++", []string{"glossary"}, nil, nil, nil) + "a language, edited\n"
	for _, dir := range []string{b, serverDir} {
		if got, _ := os.ReadFile(filepath.Join(dir, name)); string(got) != want {
			t.Errorf("%s holds %q, want %q", dir, got, want)
		}
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", b).Push(); err != nil || updated != 0 {
		t.Errorf("second push updated %d: %v", updated, err)
	}

	// Unedited, the title alone makes it match the server: nothing is pushed, and the sync state
	// follows, so a later change on the server is pulled rather than taken for a conflict.
	if err := note.WriteRaw(filepath.Join(b, name), "---\ntags: [glossary]\n---\na language, edited\n"); err != nil {
		t.Fatal(err)
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", b).Push(); err != nil || updated != 0 {
		t.Errorf("push after the title came back updated %d: %v", updated, err)
	}
	if err := note.WriteRaw(filepath.Join(a, name), note.BuildFrontmatter("C++", []string{"glossary"}, nil, nil, nil)+"from a\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := synclient.New(srv.URL, "", a).Pull(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil {
		t.Fatal(err)
	}
	if _, updated, conflicts, err := synclient.New(srv.URL, "", b).Pull(); err != nil || updated != 1 || len(conflicts) != 0 {
		t.Errorf("pull updated %d conflicts %v: %v", updated, conflicts, err)
	}
}

// A pulled note is the server's file byte for byte, final newline or not, so the first push after
// a pull sends nothing back and the server keeps the note as it was stored.
func TestPulledNoteMatchesServerByteForByte(t *testing.T) {
	srv, serverDir := testServer(t)
	a, b := t.TempDir(), t.TempDir()
	ts := time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local)
	files := map[string]string{
		filepath.Base(note.FilePath(a, ts, "newline")):                  note.BuildFrontmatter("", []string{"kb"}, nil, nil, nil) + "ends with a newline\n",
		filepath.Base(note.FilePath(a, ts.Add(time.Second), "none")):    note.BuildFrontmatter("", []string{"kb"}, nil, nil, nil) + "ends without one",
		filepath.Base(note.FilePath(a, ts.Add(2*time.Second), "blank")): note.BuildFrontmatter("T", nil, nil, nil, nil) + "two\n\n",
	}
	for name, content := range files {
		if err := note.WriteRaw(filepath.Join(a, name), content); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, _, _, _, err := synclient.New(srv.URL, "", a).Push(); err != nil {
		t.Fatal(err)
	}
	if created, _, _, err := synclient.New(srv.URL, "", b).Pull(); err != nil || created != 3 {
		t.Fatalf("pull created %d: %v", created, err)
	}
	if _, updated, _, _, _, err := synclient.New(srv.URL, "", b).Push(); err != nil || updated != 0 {
		t.Errorf("push after pull updated %d: %v", updated, err)
	}
	for name, content := range files {
		for _, dir := range []string{serverDir, b} {
			if got, _ := os.ReadFile(filepath.Join(dir, name)); string(got) != content {
				t.Errorf("%s in %s is %q, want %q", name, dir, got, content)
			}
		}
	}
}
