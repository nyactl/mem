package auth

import "testing"

func TestEnvWins(t *testing.T) {
	t.Setenv(EnvVar, "  env-token  ")
	token, src := Token()
	if token != "env-token" {
		t.Errorf("token = %q, want env-token (trimmed)", token)
	}
	if src != SourceEnv {
		t.Errorf("source = %q, want %q", src, SourceEnv)
	}
}

// An empty token is a valid state: a server started without a token accepts
// unauthenticated requests, so this must not be reported as an error.
func TestEmptyEnvFallsThrough(t *testing.T) {
	t.Setenv(EnvVar, "   ")
	if _, src := Token(); src == SourceEnv {
		t.Error("whitespace-only env var was treated as a token")
	}
}

func TestSaveRejectsEmpty(t *testing.T) {
	if err := Save("  "); err == nil {
		t.Error("expected an error for an empty token")
	}
}
