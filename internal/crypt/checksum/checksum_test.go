package checksum

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestPutGolden(t *testing.T) {
	b := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x00, 0x00, 0x00, 0x00, // checksum goes here
		0x00, 0x00, 0x00, 0x00,
	}
	if err := Put(b); err != nil {
		t.Fatal(err)
	}
	// 0x04030201 ^ 0x08070605 = 0x0c040404, little-endian.
	want := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x04, 0x04, 0x04, 0x0c,
		0x00, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(b, want) {
		t.Errorf("Put =\n% x\nwant\n% x", b, want)
	}
	if !Verify(b) {
		t.Error("Verify = false for a packet just checksummed")
	}
}

func TestChangedByteFails(t *testing.T) {
	b := []byte("abcdefghijklmnop\x00\x00\x00\x00\x00\x00\x00\x00")
	if err := Put(b); err != nil {
		t.Fatal(err)
	}
	// Every byte up to and including the checksum is covered. The last word
	// is not.
	for i := range len(b) - Size {
		c := bytes.Clone(b)
		c[i] ^= 0x01
		if Verify(c) {
			t.Errorf("Verify = true after changing byte %d", i)
		}
	}
}

func TestLengths(t *testing.T) {
	tests := []struct {
		n     int
		valid bool
	}{
		{0, false},
		{4, false},
		{7, false},
		{8, true}, // no data words, checksum 0
		{10, false},
		{12, true}, // multiple of 4 but not of 8
		{14, false},
		{16, true},
		{24, true},
	}
	for _, tt := range tests {
		b := make([]byte, tt.n)
		for i := range b {
			b[i] = byte(i * 7)
		}
		err := Put(b)
		if tt.valid != (err == nil) {
			t.Errorf("Put(%d bytes) = %v, want valid=%v", tt.n, err, tt.valid)
		}
		if !tt.valid && !errors.Is(err, ErrLength) {
			t.Errorf("Put(%d bytes) = %v, want ErrLength", tt.n, err)
		}
		if got := Verify(b); got != tt.valid {
			t.Errorf("Verify(%d bytes) after Put = %v, want %v", tt.n, got, tt.valid)
		}
	}
}

func TestEightZeroBytes(t *testing.T) {
	if !Verify(make([]byte, 8)) {
		t.Error("Verify(8 zero bytes) = false")
	}
	if Verify([]byte{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Error("checksum word must be zero when there is no data")
	}
}

// mobiusVerify is NewCrypt.verifyChecksum from L2J Mobius: the checksum is
// the last word.
func mobiusVerify(b []byte) bool {
	if len(b)%4 != 0 || len(b) <= 4 {
		return false
	}
	var sum uint32
	i := 0
	for ; i < len(b)-4; i += 4 {
		sum ^= binary.LittleEndian.Uint32(b[i:])
	}
	return sum == binary.LittleEndian.Uint32(b[i:])
}

// Packets checksummed by Put, with a zero last word, also pass the Mobius
// check that expects the checksum in the last word.
func TestCompatibleWithLastWordChecksum(t *testing.T) {
	for n := 8; n <= 64; n += 8 {
		b := make([]byte, n)
		for i := range n - 8 {
			b[i] = byte(i*13 + n)
		}
		if err := Put(b); err != nil {
			t.Fatal(err)
		}
		if !mobiusVerify(b) {
			t.Errorf("%d bytes: Mobius verify failed for % x", n, b)
		}
	}
}

func FuzzVerify(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 16))
	f.Fuzz(func(t *testing.T, data []byte) {
		Verify(data)
		b := bytes.Clone(data)
		if Put(b) == nil && !Verify(b) {
			t.Fatalf("Verify = false right after Put: % x", b)
		}
	})
}
