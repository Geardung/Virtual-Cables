package main

import (
	"io"
	"log"
	"strings"
	"testing"
)

func TestRestartFallsBackWhenPreferredAddressUnavailable(t *testing.T) {
	// 8.8.8.8 is never bindable locally, forcing Restart onto the fallback path.
	s := &appServer{address: "8.8.8.8:3240", bufferMS: 250, logger: log.New(io.Discard, "", 0)}
	if err := s.Restart(1); err != nil {
		t.Fatalf("Restart with unavailable preferred address: %v", err)
	}
	defer s.Stop()
	addr := s.Address()
	if addr == "8.8.8.8:3240" {
		t.Fatalf("server reported the unavailable address %q", addr)
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("bound %q, want a 127.0.0.1 fallback address", addr)
	}
	if s.ListenPort() < 1 || s.ListenPort() > 65535 {
		t.Fatalf("ListenPort() = %d, want a valid TCP port", s.ListenPort())
	}
	if s.Preferred() != "8.8.8.8:3240" {
		t.Fatalf("Preferred() = %q, want the configured address", s.Preferred())
	}
}

func TestListenCandidatesPreferConfiguredAddress(t *testing.T) {
	candidates := listenCandidates("127.0.0.1:3240")
	if candidates[0] != "127.0.0.1:3240" {
		t.Fatalf("first candidate = %q, want the configured address", candidates[0])
	}
	if candidates[len(candidates)-1] != "127.0.0.1:0" {
		t.Fatalf("last candidate = %q, want the ephemeral fallback", candidates[len(candidates)-1])
	}
	seen := make(map[string]bool)
	for _, addr := range candidates {
		if seen[addr] {
			t.Fatalf("duplicate candidate %q", addr)
		}
		seen[addr] = true
	}
	if len(candidates) != 21 {
		t.Fatalf("len(candidates) = %d, want 21 (preferred + 19 ports + ephemeral)", len(candidates))
	}
}

func TestListenCandidatesSkipDuplicatePreferred(t *testing.T) {
	candidates := listenCandidates("127.0.0.1:3245")
	count := 0
	for _, addr := range candidates {
		if addr == "127.0.0.1:3245" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("candidate 127.0.0.1:3245 appears %d times, want 1", count)
	}
}

func TestListenPort(t *testing.T) {
	cases := []struct {
		address string
		want    int
	}{
		{"127.0.0.1:3240", 3240},
		{"127.0.0.1:3241", 3241},
		{"localhost:3240", 3240},
		{"not-an-address", 3240},
		{"127.0.0.1:", 3240},
		{"127.0.0.1:99999", 3240},
	}
	for _, tc := range cases {
		if got := listenPort(tc.address); got != tc.want {
			t.Errorf("listenPort(%q) = %d, want %d", tc.address, got, tc.want)
		}
	}
}
