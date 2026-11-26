package clientpackets

import "testing"

// Mobius RequestSSQStatus.readImpl: c page (one unsigned byte).
func TestParseRequestSSQStatus(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want int
	}{
		{"page 1", []byte{1}, 1},
		{"page 4", []byte{4}, 4},
		{"not checked", []byte{0xff}, 255},
		{"trailing data", []byte{2, 7, 8}, 2},
	}
	for _, tt := range tests {
		got, err := ParseRequestSSQStatus(tt.body)
		m67Parsed(t, tt.name, got, err, RequestSSQStatus{Page: tt.want})
	}
	m67Truncations(t, "RequestSSQStatus", []byte{3}, ParseRequestSSQStatus)
}
