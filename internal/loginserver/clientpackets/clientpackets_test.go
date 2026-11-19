package clientpackets

import (
	"bytes"
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

func field(s string, size int) []byte {
	b := make([]byte, size)
	copy(b, s)
	return b
}

// B4: short packets used to panic, and the zero padding was kept in the
// username and password.
func TestParseRequestAuthLogin(t *testing.T) {
	data := append(field("alice", 14), field("secret", 14)...)
	data = append(data, 0, 0, 0, 0) // trailing padding is ignored
	got, err := ParseRequestAuthLogin(data)
	if err != nil {
		t.Fatalf("ParseRequestAuthLogin: %v", err)
	}
	if got.Username != "alice" || got.Password != "secret" {
		t.Errorf("got %+v, want alice/secret without padding", got)
	}

	full, err := ParseRequestAuthLogin(append(field("abcdefghijklmn", 14), field("opqrstuvwxyz01", 14)...))
	if err != nil {
		t.Fatalf("ParseRequestAuthLogin: %v", err)
	}
	if full.Username != "abcdefghijklmn" || full.Password != "opqrstuvwxyz01" {
		t.Errorf("full-length fields: got %+v", full)
	}

	for _, n := range []int{0, 1, 13, 14, 27} {
		if _, err := ParseRequestAuthLogin(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

func TestParseRequestPlay(t *testing.T) {
	sid := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	got, err := ParseRequestPlay(append(sid, 3, 0, 0))
	if err != nil {
		t.Fatalf("ParseRequestPlay: %v", err)
	}
	if !bytes.Equal(got.SessionID, sid) || got.ServerID != 3 {
		t.Errorf("got %+v", got)
	}
	for _, n := range []int{0, 7, 8} {
		if _, err := ParseRequestPlay(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

func TestParseRequestServerList(t *testing.T) {
	sid := []byte{8, 7, 6, 5, 4, 3, 2, 1}
	got, err := ParseRequestServerList(append(sid, 0xff))
	if err != nil {
		t.Fatalf("ParseRequestServerList: %v", err)
	}
	if !bytes.Equal(got.SessionID, sid) {
		t.Errorf("SessionID = %x, want %x", got.SessionID, sid)
	}
	if _, err := ParseRequestServerList(make([]byte, 7)); !errors.Is(err, packets.ErrShortPacket) {
		t.Errorf("7 bytes: error = %v, want ErrShortPacket", err)
	}
}

func FuzzParsers(f *testing.F) {
	f.Add([]byte{})
	f.Add(append(field("user", 14), field("pass", 14)...))
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseRequestAuthLogin(data)
		_, _ = ParseRequestPlay(data)
		_, _ = ParseRequestServerList(data)
	})
}
