package main

import "testing"

func TestResolveAccessControlOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   string
	}{
		{name: "localhost dev", origin: "http://localhost:8443", want: "http://localhost:8443"},
		{name: "universearch api", origin: "https://api-lunetterie.universearch.com", want: "https://api-lunetterie.universearch.com"},
		{name: "render subdomain", origin: "https://frontend.onrender.com", want: "https://frontend.onrender.com"},
		{name: "unknown origin", origin: "https://evil.example.com", want: "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAccessControlOrigin(tt.origin); got != tt.want {
				t.Fatalf("resolveAccessControlOrigin(%q) = %q, want %q", tt.origin, got, tt.want)
			}
		})
	}
}
