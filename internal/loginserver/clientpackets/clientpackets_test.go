package clientpackets

import (
	"bytes"
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

func seq(start byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = start + byte(i)
	}
	return b
}

// AuthGameGuard.readImpl: d session ID, then four ints that are ignored.
func TestParseAuthGameGuard(t *testing.T) {
	data := []byte{
		0x78, 0x56, 0x34, 0x12, // session ID
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, // GameGuard data
		0, 0, 0, 0, 0, 0, 0, // padding and checksum are ignored
	}
	got, err := ParseAuthGameGuard(data)
	if err != nil {
		t.Fatalf("ParseAuthGameGuard: %v", err)
	}
	if got.SessionID != 0x12345678 {
		t.Errorf("SessionID = %#x, want 0x12345678", got.SessionID)
	}

	neg, err := ParseAuthGameGuard(append([]byte{0xff, 0xff, 0xff, 0xff}, make([]byte, 16)...))
	if err != nil || neg.SessionID != -1 {
		t.Errorf("negative session ID: got %+v, %v", neg, err)
	}

	for _, n := range []int{0, 3, 4, 19} {
		if _, err := ParseAuthGameGuard(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// RequestAuthLogin.readImpl: 256 bytes or more left → two blocks, 128 or
// more → one block, otherwise the packet is dropped.
func TestParseRequestAuthLogin(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantLen int
	}{
		{"exactly one block", 128, 128},
		{"one block with padding and checksum", 143, 128}, // what a 128-byte client sends
		{"just under two blocks", 255, 128},
		{"exactly two blocks", 256, 256},
		{"two blocks with padding and checksum", 271, 256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := seq(1, tt.size)
			got, err := ParseRequestAuthLogin(data)
			if err != nil {
				t.Fatalf("ParseRequestAuthLogin: %v", err)
			}
			if !bytes.Equal(got.Encrypted, data[:tt.wantLen]) {
				t.Errorf("Encrypted = %d bytes %x..., want the first %d bytes", len(got.Encrypted), got.Encrypted[:4], tt.wantLen)
			}
			if got.NewAuth() != (tt.wantLen == 256) {
				t.Errorf("NewAuth = %v", got.NewAuth())
			}
		})
	}

	// The block is a copy: changing the packet buffer must not change it.
	data := seq(0, 128)
	got, err := ParseRequestAuthLogin(data)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 0xee
	if got.Encrypted[0] != 0 {
		t.Error("Encrypted aliases the packet buffer")
	}

	for _, n := range []int{0, 1, 64, 127} {
		if _, err := ParseRequestAuthLogin(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// RequestServerList.readImpl: d loginOk1, d loginOk2 (8 bytes needed).
func TestParseRequestServerList(t *testing.T) {
	data := []byte{0x04, 0x03, 0x02, 0x01, 0xfe, 0xff, 0xff, 0xff, 0x05, 0, 0, 0}
	got, err := ParseRequestServerList(data)
	if err != nil {
		t.Fatalf("ParseRequestServerList: %v", err)
	}
	if got.LoginOK1 != 0x01020304 || got.LoginOK2 != -2 {
		t.Errorf("got %+v, want 0x01020304 and -2", got)
	}
	for _, n := range []int{0, 4, 7} {
		if _, err := ParseRequestServerList(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// RequestServerLogin.readImpl: d loginOk1, d loginOk2, c server ID (9 bytes
// needed).
func TestParseRequestServerLogin(t *testing.T) {
	data := []byte{0x0d, 0x0c, 0x0b, 0x0a, 0x01, 0x00, 0x00, 0x80, 0x03, 0, 0, 0, 0, 0, 0}
	got, err := ParseRequestServerLogin(data)
	if err != nil {
		t.Fatalf("ParseRequestServerLogin: %v", err)
	}
	want := RequestServerLogin{LoginOK1: 0x0a0b0c0d, LoginOK2: -0x7fffffff, ServerID: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	high, err := ParseRequestServerLogin(append(make([]byte, 8), 0xff))
	if err != nil || high.ServerID != 255 {
		t.Errorf("server ID 255: got %+v, %v", high, err)
	}

	for _, n := range []int{0, 7, 8} {
		if _, err := ParseRequestServerLogin(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// Garbage of any length must never panic, and a parser either fails or
// returns a value; it never succeeds on less than its minimum size.
func FuzzParsers(f *testing.F) {
	f.Add([]byte{})
	f.Add(seq(0, 20))
	f.Add(seq(0, 128))
	f.Add(seq(0, 271))
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		if _, err := ParseAuthGameGuard(data); (err == nil) != (len(data) >= 20) {
			t.Errorf("AuthGameGuard on %d bytes: %v", len(data), err)
		}
		if p, err := ParseRequestAuthLogin(data); (err == nil) != (len(data) >= 128) {
			t.Errorf("RequestAuthLogin on %d bytes: %v", len(data), err)
		} else if err == nil && len(p.Encrypted) != 128 && len(p.Encrypted) != 256 {
			t.Errorf("RequestAuthLogin block is %d bytes", len(p.Encrypted))
		}
		if _, err := ParseRequestServerList(data); (err == nil) != (len(data) >= 8) {
			t.Errorf("RequestServerList on %d bytes: %v", len(data), err)
		}
		if _, err := ParseRequestServerLogin(data); (err == nil) != (len(data) >= 9) {
			t.Errorf("RequestServerLogin on %d bytes: %v", len(data), err)
		}
	})
}
