package logincrypt

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/crypt/blowfish"
)

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// words builds a byte slice from little-endian 32-bit words.
func words(ws ...uint32) []byte {
	b := make([]byte, 0, 4*len(ws))
	for _, w := range ws {
		b = binary.LittleEndian.AppendUint32(b, w)
	}
	return b
}

// rng returns a deterministic random source for keys and test data.
func rng(seed byte) *rand.ChaCha8 {
	var s [32]byte
	s[0] = seed
	return rand.NewChaCha8(s)
}

func randBytes(r io.Reader, n int) []byte {
	b := make([]byte, n)
	_, _ = io.ReadFull(r, b)
	return b
}

func TestStaticKey(t *testing.T) {
	want := unhex(t, "6b60cb5b82ce90b1cc2b6c556c6c6c6c")
	if !bytes.Equal(StaticKey[:], want) {
		t.Errorf("StaticKey = %x, want %x", StaticKey, want)
	}
}

// Sizes worked out by hand from Mobius LoginEncryption.encryptedSize:
// size = n + header; size += 8 - size%8; size += 8.
func TestPacketSizes(t *testing.T) {
	static := []struct{ n, want int }{
		{0, 24},    // 8 -> 16 -> 24
		{1, 24},    // 9 -> 16 -> 24
		{7, 24},    // 15 -> 16 -> 24
		{8, 32},    // 16 -> 24 -> 32
		{170, 192}, // Init body: 178 -> 184 -> 192
	}
	for _, tt := range static {
		if got := StaticPacketSize(tt.n); got != tt.want {
			t.Errorf("StaticPacketSize(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
	session := []struct{ n, want int }{
		{1, 16},  // 5 -> 8 -> 16
		{3, 16},  // 7 -> 8 -> 16
		{4, 24},  // 8 -> 16 -> 24
		{12, 32}, // 16 -> 24 -> 32
		{13, 32}, // 17 -> 24 -> 32
		{20, 40}, // 24 -> 32 -> 40
	}
	for _, tt := range session {
		if got := SessionPacketSize(tt.n); got != tt.want {
			t.Errorf("SessionPacketSize(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
	for n := range 300 {
		s := StaticPacketSize(n)
		if s%BlockSize != 0 || s-n < 17 || s-n > 24 {
			t.Errorf("StaticPacketSize(%d) = %d: not aligned or overhead out of 17..24", n, s)
		}
		s = SessionPacketSize(n)
		if s%BlockSize != 0 || s-n < 13 || s-n > 20 {
			t.Errorf("SessionPacketSize(%d) = %d: not aligned or overhead out of 13..20", n, s)
		}
	}
}

func TestAppendChecksumGolden(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"one word", words(0x12345678, 0), words(0x12345678, 0x12345678)},
		{"two words", words(1, 2, 0), words(1, 2, 3)},
		{"carry free", words(0xff000000, 0x00ff0000, 0x0000ffff, 0xdeadbeef), words(0xff000000, 0x00ff0000, 0x0000ffff, 0xffffffff)},
		{"zero padding", words(5, 0, 0, 0), words(5, 0, 0, 5)},
	}
	for _, tt := range tests {
		b := slices.Clone(tt.in)
		if err := AppendChecksum(b); err != nil {
			t.Fatalf("%s: AppendChecksum: %v", tt.name, err)
		}
		if !bytes.Equal(b, tt.want) {
			t.Errorf("%s: AppendChecksum = %x, want %x", tt.name, b, tt.want)
		}
		if !VerifyChecksum(b) {
			t.Errorf("%s: VerifyChecksum = false after AppendChecksum", tt.name)
		}
	}
}

// Interlude puts the checksum in the last word (C1 L2Go used len-8). The
// word before it is plain data and is covered by the checksum.
func TestChecksumIsInLastWord(t *testing.T) {
	b := words(1, 2, 4, 0)
	if err := AppendChecksum(b); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(b[8:]); got != 4 {
		t.Errorf("word at len-8 = %#x, want it unchanged (4)", got)
	}
	if got := binary.LittleEndian.Uint32(b[12:]); got != 1^2^4 {
		t.Errorf("checksum at len-4 = %#x, want %#x", got, 1^2^4)
	}
}

func TestVerifyChecksum(t *testing.T) {
	good := words(0xcafebabe, 0x01020304, 0xcafebabe^0x01020304)
	if !VerifyChecksum(good) {
		t.Fatal("VerifyChecksum(good) = false")
	}
	for i := range len(good) * 8 {
		b := slices.Clone(good)
		b[i/8] ^= 1 << (i % 8)
		if VerifyChecksum(b) {
			t.Errorf("VerifyChecksum accepted a flip of bit %d", i)
		}
	}
	// Mobius rejects sizes that are not a multiple of 4 or not longer
	// than the checksum.
	for _, n := range []int{0, 1, 3, 4, 5, 6, 7, 9} {
		if VerifyChecksum(make([]byte, n)) {
			t.Errorf("VerifyChecksum(%d zero bytes) = true", n)
		}
	}
	if !VerifyChecksum(make([]byte, 8)) {
		t.Error("VerifyChecksum(8 zero bytes) = false, want true")
	}
	for _, n := range []int{0, 4, 6, 10} {
		if err := AppendChecksum(make([]byte, n)); !errors.Is(err, ErrSize) {
			t.Errorf("AppendChecksum(%d bytes) = %v, want ErrSize", n, err)
		}
	}
}

// XOR pass vectors worked out by hand from NewCrypt.encXORPass: for each
// word from offset 4 to len-8, key += word; word ^= key. The key is then
// stored at len-8.
func TestEncXORPassGolden(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		key  uint32
		want []byte
	}{
		{
			// key = 1+2 = 3, word = 2^3 = 1; key 3 stored at 8; first
			// and last words untouched.
			name: "one word",
			in:   words(0xaabbccdd, 2, 0x11111111, 0x22222222),
			key:  1,
			want: words(0xaabbccdd, 1, 3, 0x22222222),
		},
		{
			// 0x100+0x10=0x110 -> 0x10^0x110=0x100
			// 0x110+0x20=0x130 -> 0x20^0x130=0x110
			// 0x130+0x30=0x160 -> 0x30^0x160=0x150
			name: "three words",
			in:   words(0, 0x10, 0x20, 0x30, 0, 0),
			key:  0x100,
			want: words(0, 0x100, 0x110, 0x150, 0x160, 0),
		},
		{
			// 0xffffffff + 1 wraps to 0; 1^0 = 1.
			name: "wraps",
			in:   words(0, 1, 0, 0),
			key:  0xffffffff,
			want: words(0, 1, 0, 0),
		},
		{
			// No data words: the key goes to offset 4 (= len-8).
			name: "minimum size",
			in:   words(7, 0, 0),
			key:  0x01020304,
			want: words(7, 0x01020304, 0),
		},
	}
	for _, tt := range tests {
		b := slices.Clone(tt.in)
		if err := EncXORPass(b, tt.key); err != nil {
			t.Fatalf("%s: EncXORPass: %v", tt.name, err)
		}
		if !bytes.Equal(b, tt.want) {
			t.Errorf("%s: EncXORPass = %x, want %x", tt.name, b, tt.want)
		}
		if err := DecXORPass(b); err != nil {
			t.Fatalf("%s: DecXORPass: %v", tt.name, err)
		}
		// Everything except the key word comes back.
		n := len(b) - 8
		if !bytes.Equal(b[:n], tt.in[:n]) || !bytes.Equal(b[n+4:], tt.in[n+4:]) {
			t.Errorf("%s: DecXORPass = %x, want %x outside the key word", tt.name, b, tt.in)
		}
	}
}

func TestXORPassSizeErrors(t *testing.T) {
	for _, n := range []int{0, 4, 8, 13, 14, 15} {
		if err := EncXORPass(make([]byte, n), 1); !errors.Is(err, ErrSize) {
			t.Errorf("EncXORPass(%d bytes) = %v, want ErrSize", n, err)
		}
		if err := DecXORPass(make([]byte, n)); !errors.Is(err, ErrSize) {
			t.Errorf("DecXORPass(%d bytes) = %v, want ErrSize", n, err)
		}
	}
}

// The first server packet is the body after the XOR pass, encrypted with
// the static key. Decrypting with the plain Blowfish wrapper must show the
// XOR pass layout worked out by hand.
func TestFirstServerPacketLayout(t *testing.T) {
	key := randBytes(rng(1), KeySize)
	s, err := NewSession(key, bytes.NewReader([]byte{1, 0, 0, 0})) // XOR key 1
	if err != nil {
		t.Fatal(err)
	}
	body := []byte{0x00, 0x2a, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}
	enc, err := s.EncryptServerPacket(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 32 {
		t.Fatalf("len = %d, want 32", len(enc))
	}
	bf, err := blowfish.NewCipher(StaticKey[:])
	if err != nil {
		t.Fatal(err)
	}
	if err := bf.Decrypt(enc, enc); err != nil {
		t.Fatal(err)
	}
	// Word 0 is plain. Word 1: key 1+2 = 3, 2^3 = 1. The zero padding
	// words up to len-8 each become 0^3 = 3. Key 3 at len-8, last word 0.
	want := words(0x00002a00, 1, 3, 3, 3, 3, 3, 0)
	if !bytes.Equal(enc, want) {
		t.Errorf("static packet after Blowfish = %x, want %x", enc, want)
	}
	if !bytes.Equal(body, []byte{0x00, 0x2a, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) {
		t.Error("EncryptServerPacket modified body")
	}
}

// After the first packet, a server packet is the body, zero padding and the
// checksum in the last word, encrypted with the session key.
func TestSessionPacketLayout(t *testing.T) {
	key := randBytes(rng(2), KeySize)
	s, err := NewSession(key, rng(3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncryptServerPacket([]byte{0}); err != nil {
		t.Fatal(err)
	}
	enc, err := s.EncryptServerPacket([]byte{0x05})
	if err != nil {
		t.Fatal(err)
	}
	bf, err := blowfish.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := bf.Decrypt(enc, enc); err != nil {
		t.Fatal(err)
	}
	if want := words(5, 0, 0, 5); !bytes.Equal(enc, want) {
		t.Errorf("session packet after Blowfish = %x, want %x", enc, want)
	}
}

// initBody returns a fake Init body with the session key at offset 153, as
// in the real packet (opcode, session ID, protocol, modulus, GG data).
func initBody(key []byte) []byte {
	body := make([]byte, 170)
	body[0] = 0x00
	copy(body[1:5], []byte{0x11, 0x22, 0x33, 0x44})
	binary.LittleEndian.PutUint32(body[5:], 0xc621)
	for i := 9; i < 153; i++ {
		body[i] = byte(i)
	}
	copy(body[153:169], key)
	return body
}

func TestServerClientRoundTrip(t *testing.T) {
	src := rng(4)
	sessionKey, err := NewSessionKey(src)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewSession(sessionKey[:], src)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClientSession()

	// Init: static key and XOR pass.
	body := initBody(sessionKey[:])
	enc, err := server.EncryptServerPacket(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != StaticPacketSize(len(body)) {
		t.Fatalf("Init is %d bytes, want %d", len(enc), StaticPacketSize(len(body)))
	}
	if err := client.DecryptServerPacket(enc); err != nil {
		t.Fatalf("client DecryptServerPacket(Init): %v", err)
	}
	if !bytes.Equal(enc[:len(body)], body) {
		t.Fatalf("client read Init = %x, want %x", enc[:len(body)], body)
	}
	if err := client.SetSessionKey(enc[153:169]); err != nil {
		t.Fatal(err)
	}

	// Many packets in both directions with the session key.
	data := rand.New(rand.NewPCG(5, 6))
	for i := range 300 {
		body := randBytes(src, 1+data.IntN(400))

		up, err := client.EncryptClientPacket(body)
		if err != nil {
			t.Fatalf("packet %d: EncryptClientPacket: %v", i, err)
		}
		if len(up) != SessionPacketSize(len(body)) {
			t.Fatalf("packet %d: client packet is %d bytes, want %d", i, len(up), SessionPacketSize(len(body)))
		}
		if err := server.DecryptClientPacket(up); err != nil {
			t.Fatalf("packet %d: DecryptClientPacket: %v", i, err)
		}
		if !bytes.Equal(up[:len(body)], body) {
			t.Fatalf("packet %d: server read %x, want %x", i, up[:len(body)], body)
		}

		down, err := server.EncryptServerPacket(body)
		if err != nil {
			t.Fatalf("packet %d: EncryptServerPacket: %v", i, err)
		}
		if err := client.DecryptServerPacket(down); err != nil {
			t.Fatalf("packet %d: client DecryptServerPacket: %v", i, err)
		}
		if !bytes.Equal(down[:len(body)], body) {
			t.Fatalf("packet %d: client read %x, want %x", i, down[:len(body)], body)
		}
	}
}

// Each first packet gets a fresh XOR key, so equal bodies encrypt
// differently while still decrypting to the same body.
func TestFirstPacketUsesRandomXORKey(t *testing.T) {
	key := randBytes(rng(7), KeySize)
	body := initBody(key)
	var encs [][]byte
	for i := range 2 {
		s, err := NewSession(key, bytes.NewReader([]byte{byte(i + 1), 0, 0, 0}))
		if err != nil {
			t.Fatal(err)
		}
		enc, err := s.EncryptServerPacket(body)
		if err != nil {
			t.Fatal(err)
		}
		encs = append(encs, slices.Clone(enc))
		c := NewClientSession()
		if err := c.DecryptServerPacket(enc); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(enc[:len(body)], body) {
			t.Errorf("XOR key %d: client read a different body", i+1)
		}
	}
	if bytes.Equal(encs[0], encs[1]) {
		t.Error("different XOR keys gave the same ciphertext")
	}
}

func TestSessionErrors(t *testing.T) {
	key := randBytes(rng(8), KeySize)
	for _, n := range []int{0, 8, 15, 17, 56} {
		if _, err := NewSession(make([]byte, n), nil); err == nil {
			t.Errorf("NewSession(%d-byte key) succeeded", n)
		}
		if err := NewClientSession().SetSessionKey(make([]byte, n)); err == nil {
			t.Errorf("SetSessionKey(%d-byte key) succeeded", n)
		}
	}

	s, err := NewSession(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncryptServerPacket(nil); !errors.Is(err, ErrSize) {
		t.Errorf("EncryptServerPacket(empty) = %v, want ErrSize", err)
	}
	if _, err := s.EncryptServerPacket([]byte{0}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EncryptServerPacket(nil); !errors.Is(err, ErrSize) {
		t.Errorf("EncryptServerPacket(empty) after Init = %v, want ErrSize", err)
	}
	for _, n := range []int{0, 1, 7, 9, 23} {
		if err := s.DecryptClientPacket(make([]byte, n)); !errors.Is(err, ErrSize) {
			t.Errorf("DecryptClientPacket(%d bytes) = %v, want ErrSize", n, err)
		}
	}

	c := NewClientSession()
	if err := c.SetSessionKey(key); err != nil {
		t.Fatal(err)
	}
	up, err := c.EncryptClientPacket([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	up[0] ^= 0x80
	if err := s.DecryptClientPacket(up); !errors.Is(err, ErrChecksum) {
		t.Errorf("DecryptClientPacket(tampered) = %v, want ErrChecksum", err)
	}

	// A packet made with another key fails the checksum.
	other := NewClientSession()
	if err := other.SetSessionKey(randBytes(rng(9), KeySize)); err != nil {
		t.Fatal(err)
	}
	up, err = other.EncryptClientPacket([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DecryptClientPacket(up); !errors.Is(err, ErrChecksum) {
		t.Errorf("DecryptClientPacket(wrong key) = %v, want ErrChecksum", err)
	}

	// A failing random source stops the first packet.
	broken, err := NewSession(key, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broken.EncryptServerPacket([]byte{0}); err == nil {
		t.Error("EncryptServerPacket with an empty random source succeeded")
	}
}

func TestClientSessionErrors(t *testing.T) {
	c := NewClientSession()
	if _, err := c.EncryptClientPacket([]byte{1}); !errors.Is(err, ErrNoSessionKey) {
		t.Errorf("EncryptClientPacket before key = %v, want ErrNoSessionKey", err)
	}
	for _, n := range []int{0, 7, 8, 9} {
		if err := c.DecryptServerPacket(make([]byte, n)); !errors.Is(err, ErrSize) {
			t.Errorf("DecryptServerPacket(%d bytes) = %v, want ErrSize", n, err)
		}
	}
	// The first packet decrypts without the session key; the second needs it.
	if err := c.DecryptServerPacket(make([]byte, 24)); err != nil {
		t.Fatalf("DecryptServerPacket(first) = %v", err)
	}
	if err := c.DecryptServerPacket(make([]byte, 16)); !errors.Is(err, ErrNoSessionKey) {
		t.Errorf("DecryptServerPacket(second) before key = %v, want ErrNoSessionKey", err)
	}
}

func TestNewSessionKey(t *testing.T) {
	a, err := NewSessionKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSessionKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two random session keys are equal")
	}
	if _, err := NewSessionKey(bytes.NewReader(make([]byte, KeySize-1))); err == nil {
		t.Error("NewSessionKey with a short random source succeeded")
	}
	got, err := NewSessionKey(bytes.NewReader(bytes.Repeat([]byte{0xab}, KeySize)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:], bytes.Repeat([]byte{0xab}, KeySize)) {
		t.Errorf("NewSessionKey = %x, want the bytes of the source", got)
	}
}

func FuzzDecryptClientPacket(f *testing.F) {
	key := randBytes(rng(10), KeySize)
	f.Add([]byte{})
	f.Add(make([]byte, 16))
	f.Add(make([]byte, 13))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := NewSession(key, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.DecryptClientPacket(data); err != nil {
			return
		}
		if !VerifyChecksum(data) {
			t.Fatal("DecryptClientPacket accepted a packet without a valid checksum")
		}
	})
}

func FuzzClientDecryptServerPacket(f *testing.F) {
	key := randBytes(rng(11), KeySize)
	f.Add([]byte{}, false)
	f.Add(make([]byte, 24), true)
	f.Add(make([]byte, 12), false)
	f.Fuzz(func(t *testing.T, data []byte, afterInit bool) {
		c := NewClientSession()
		if afterInit {
			if err := c.DecryptServerPacket(make([]byte, 24)); err != nil {
				t.Fatal(err)
			}
			if err := c.SetSessionKey(key); err != nil {
				t.Fatal(err)
			}
		}
		_ = c.DecryptServerPacket(data)
	})
}

func FuzzXORPass(f *testing.F) {
	f.Add(make([]byte, 16), uint32(1))
	f.Add([]byte{1, 2, 3}, uint32(0))
	f.Fuzz(func(t *testing.T, data []byte, key uint32) {
		_ = DecXORPass(slices.Clone(data))
		b := slices.Clone(data)
		if err := EncXORPass(b, key); err != nil {
			return
		}
		if got := binary.LittleEndian.Uint32(b[len(b)-8:]); len(b) == 12 && got != key {
			t.Fatalf("key word = %#x, want %#x", got, key)
		}
		if err := DecXORPass(b); err != nil {
			t.Fatal(err)
		}
		n := len(b) - 8
		if !bytes.Equal(b[:n], data[:n]) || !bytes.Equal(b[n+4:], data[n+4:]) {
			t.Fatalf("DecXORPass(EncXORPass(x)) = %x, want %x outside the key word", b, data)
		}
	})
}

func FuzzChecksum(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 8))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = VerifyChecksum(data)
		b := slices.Clone(data)
		if err := AppendChecksum(b); err != nil {
			return
		}
		if !VerifyChecksum(b) {
			t.Fatalf("VerifyChecksum(AppendChecksum(%x)) = false", data)
		}
	})
}

func BenchmarkSessionPacket(b *testing.B) {
	key := randBytes(rng(12), KeySize)
	s, err := NewSession(key, nil)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := s.EncryptServerPacket([]byte{0}); err != nil {
		b.Fatal(err)
	}
	body := make([]byte, 256)
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		if _, err := s.EncryptServerPacket(body); err != nil {
			b.Fatal(err)
		}
	}
}
