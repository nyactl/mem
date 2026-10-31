package synclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A server with auth_token set rejects a client that has none. The message has
// to name the fix, because nothing else in the client's output does.
func TestPullWithoutTokenNamesTheMissingConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, _, _, err := New(srv.URL, "", t.TempDir()).Pull()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"none is configured", "mem auth login", "MEM_AUTH_TOKEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// With a token configured the cause is different -- it is wrong, not absent --
// and the message must not tell the user to set one they already set.
func TestPullWithWrongTokenSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, _, _, err := New(srv.URL, "wrong-token", t.TempDir()).Pull()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "rejected the token") {
		t.Errorf("error %q does not report a rejected token", err)
	}
	if strings.Contains(err.Error(), "none is configured") {
		t.Errorf("error %q tells the user to set a token they already have", err)
	}
}

func TestPullTreats403LikeAnAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	_, _, _, err := New(srv.URL, "", t.TempDir()).Pull()
	if err == nil || !strings.Contains(err.Error(), "none is configured") {
		t.Errorf("403 error = %v, want the auth hint", err)
	}
}

// Non-auth failures keep the server's own explanation, which is the only clue
// available for them.
func TestPushKeepsServerMessageForOtherStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("[]"))
			return
		}
		http.Error(w, "disk is full", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "20260930T101500-a-note.md")
	if err := os.WriteFile(path, []byte("---\ntags: []\n---\nbody\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, _, _, _, _, err := New(srv.URL, "", dir).Push()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "disk is full") {
		t.Errorf("error %q drops the server's message", err)
	}
	if strings.Contains(err.Error(), "auth_token") {
		t.Errorf("error %q blames auth for a 500", err)
	}
}

// A server that turns one note down (400) does not stop the push: the note is refused with the
// server's reason and the next one is still sent.
func TestPushGoesOnWhenServerRefusesANote(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost:
			posts++
			if posts == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"tag \"x\" is not a slug"}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"etag":"e"}`))
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	for _, name := range []string{"20261028T100000-a.md", "20261028T100001-b.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("---\ntags: []\n---\nbody\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	created, _, _, _, refused, err := New(srv.URL, "", dir).Push()
	if err != nil || created != 1 || posts != 2 {
		t.Fatalf("created %d posts %d: %v", created, posts, err)
	}
	if len(refused) != 1 || !strings.Contains(refused[0].Msg, `refused by the server: tag "x" is not a slug`) {
		t.Errorf("refused %v", refused)
	}
}

// The same holds for an update the server turns down.
func TestPushGoesOnWhenServerRefusesAnUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"20261028T100000-a","etag":"old"},{"id":"20261028T100001-b","etag":"old"}]`))
		case http.MethodPatch:
			if strings.HasSuffix(r.URL.Path, "-a") {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"no"}`))
				return
			}
			_, _ = w.Write([]byte(`{"etag":"new"}`))
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	for _, name := range []string{"20261028T100000-a.md", "20261028T100001-b.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("---\ntags: []\n---\nbody\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, updated, _, _, refused, err := New(srv.URL, "", dir).Push()
	if err != nil || updated != 1 || len(refused) != 1 || refused[0].Msg != "refused by the server: no" {
		t.Errorf("updated %d refused %v: %v", updated, refused, err)
	}
}
