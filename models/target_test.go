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

func TestAllowedByGroups(t *testing.T) {
	target := Target{
		Name:        "test",
		AllowGroups: []string{"admin", "dev"},
	}

	tests := []struct {
		name       string
		userGroups []string
		want       bool
	}{
		{"matching group", []string{"admin"}, true},
		{"second matching group", []string{"dev"}, true},
		{"multiple groups one matches", []string{"ops", "dev"}, true},
		{"no matching group", []string{"ops", "qa"}, false},
		{"empty user groups", []string{}, false},
		{"nil user groups", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := target.AllowedByGroups(tt.userGroups)
			if got != tt.want {
				t.Errorf("AllowedByGroups(%v) = %v, want %v", tt.userGroups, got, tt.want)
			}
		})
	}
}
