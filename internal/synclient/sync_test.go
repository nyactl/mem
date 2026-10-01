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
	for _, want := range []string{"no auth_token is configured", "MEM_AUTH_TOKEN"} {
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
	if !strings.Contains(err.Error(), "rejected the configured auth_token") {
		t.Errorf("error %q does not report a rejected token", err)
	}
	if strings.Contains(err.Error(), "no auth_token is configured") {
		t.Errorf("error %q tells the user to set a token they already have", err)
	}
}

func TestPullTreats403LikeAnAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	_, _, _, err := New(srv.URL, "", t.TempDir()).Pull()
	if err == nil || !strings.Contains(err.Error(), "no auth_token is configured") {
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

	_, _, _, _, err := New(srv.URL, "", dir).Push()
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
