package coze

import "testing"

func TestResolveUploadHostFallsBackToRequestHostForLoopbackConfig(t *testing.T) {
	got := resolveUploadHost("http://localhost:8888", "10.10.10.226:8896", "")
	if got != "10.10.10.226:8896" {
		t.Fatalf("resolveUploadHost() = %q, want %q", got, "10.10.10.226:8896")
	}
}

func TestResolveUploadHostUsesForwardedHost(t *testing.T) {
	got := resolveUploadHost("http://127.0.0.1:8888", "127.0.0.1:8888", "agent.example.com")
	if got != "agent.example.com" {
		t.Fatalf("resolveUploadHost() = %q, want %q", got, "agent.example.com")
	}
}

func TestResolveUploadHostKeepsNonLoopbackConfig(t *testing.T) {
	got := resolveUploadHost("https://agent.example.com", "10.10.10.226:8896", "")
	if got != "agent.example.com" {
		t.Fatalf("resolveUploadHost() = %q, want %q", got, "agent.example.com")
	}
}
