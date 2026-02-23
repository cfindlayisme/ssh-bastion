package models

import "testing"

func TestTargetAddr(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected string
	}{
		{
			name:     "standard host and port",
			target:   Target{Host: "192.168.1.10", Port: "22"},
			expected: "192.168.1.10:22",
		},
		{
			name:     "hostname with non-standard port",
			target:   Target{Host: "example.com", Port: "2222"},
			expected: "example.com:2222",
		},
		{
			name:     "IPv6 address",
			target:   Target{Host: "::1", Port: "22"},
			expected: "[::1]:22",
		},
		{
			name:     "empty host",
			target:   Target{Host: "", Port: "22"},
			expected: ":22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.Addr()
			if got != tt.expected {
				t.Errorf("Addr() = %q, want %q", got, tt.expected)
			}
		})
	}
}
