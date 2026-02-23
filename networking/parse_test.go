package networking

import (
	"encoding/binary"
	"testing"
)

// Helper to build a pty-req payload:
//
//	uint32 term_len, []byte term, uint32 width, uint32 height, uint32 pix_w, uint32 pix_h, string modes
func buildPtyPayload(term string, width, height uint32) []byte {
	termBytes := []byte(term)
	// 4 (term_len) + len(term) + 4*4 (w, h, pix_w, pix_h) + 4 (modes_len) + 0 (modes)
	buf := make([]byte, 4+len(termBytes)+16+4)
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(termBytes)))
	copy(buf[4:4+len(termBytes)], termBytes)
	off := 4 + len(termBytes)
	binary.BigEndian.PutUint32(buf[off:off+4], width)
	binary.BigEndian.PutUint32(buf[off+4:off+8], height)
	// pix_w, pix_h, modes_len left as zero
	return buf
}

func buildWindowChangePayload(width, height uint32) []byte {
	buf := make([]byte, 16) // w, h, pix_w, pix_h
	binary.BigEndian.PutUint32(buf[0:4], width)
	binary.BigEndian.PutUint32(buf[4:8], height)
	return buf
}

func TestParsePtyRequest(t *testing.T) {
	tests := []struct {
		name           string
		payload        []byte
		expectWidth    int
		expectHeight   int
	}{
		{
			name:         "standard 80x24",
			payload:      buildPtyPayload("xterm-256color", 80, 24),
			expectWidth:  80,
			expectHeight: 24,
		},
		{
			name:         "large terminal",
			payload:      buildPtyPayload("xterm", 200, 50),
			expectWidth:  200,
			expectHeight: 50,
		},
		{
			name:         "empty term string",
			payload:      buildPtyPayload("", 120, 40),
			expectWidth:  120,
			expectHeight: 40,
		},
		{
			name:         "nil payload defaults",
			payload:      nil,
			expectWidth:  80,
			expectHeight: 24,
		},
		{
			name:         "empty payload defaults",
			payload:      []byte{},
			expectWidth:  80,
			expectHeight: 24,
		},
		{
			name:         "truncated payload defaults",
			payload:      []byte{0, 0, 0, 5, 'x', 't'},
			expectWidth:  80,
			expectHeight: 24,
		},
		{
			name:         "zero width defaults to 80",
			payload:      buildPtyPayload("xterm", 0, 30),
			expectWidth:  80,
			expectHeight: 30,
		},
		{
			name:         "zero height defaults to 24",
			payload:      buildPtyPayload("xterm", 100, 0),
			expectWidth:  100,
			expectHeight: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := ParsePtyRequest(tt.payload)
			if w != tt.expectWidth {
				t.Errorf("width = %d, want %d", w, tt.expectWidth)
			}
			if h != tt.expectHeight {
				t.Errorf("height = %d, want %d", h, tt.expectHeight)
			}
		})
	}
}

func TestParseWindowChange(t *testing.T) {
	tests := []struct {
		name         string
		payload      []byte
		expectWidth  int
		expectHeight int
	}{
		{
			name:         "standard resize",
			payload:      buildWindowChangePayload(120, 40),
			expectWidth:  120,
			expectHeight: 40,
		},
		{
			name:         "small terminal",
			payload:      buildWindowChangePayload(40, 10),
			expectWidth:  40,
			expectHeight: 10,
		},
		{
			name:         "nil payload defaults",
			payload:      nil,
			expectWidth:  80,
			expectHeight: 24,
		},
		{
			name:         "short payload defaults",
			payload:      []byte{0, 0, 0},
			expectWidth:  80,
			expectHeight: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := ParseWindowChange(tt.payload)
			if w != tt.expectWidth {
				t.Errorf("width = %d, want %d", w, tt.expectWidth)
			}
			if h != tt.expectHeight {
				t.Errorf("height = %d, want %d", h, tt.expectHeight)
			}
		})
	}
}
