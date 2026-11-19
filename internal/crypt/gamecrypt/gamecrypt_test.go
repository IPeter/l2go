package gamecrypt

import (
	"bytes"
	"encoding/hex"
	"math/rand/v2"
	"slices"
	"testing"
)

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// testKey is A0 01 02 ... 0F: key[i] = i except key[0], so that a run of
// zero bytes encrypts to the running XOR of the key bytes.
func testKey() [KeySize]byte {
	var k [KeySize]byte
	for i := range k {
		k[i] = byte(i)
	}
	k[0] = 0xa0
	return k
}

func TestNewKey(t *testing.T) {
	key, err := NewKey(bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	if err != nil {
		t.Fatal(err)
	}
	want := unhex(t, "0102030405060708c8279301a16c3197")
	if !bytes.Equal(key[:], want) {
		t.Errorf("NewKey = %x, want %x", key, want)
	}
	if _, err := NewKey(bytes.NewReader([]byte{1, 2, 3})); err == nil {
		t.Error("NewKey with a short random source succeeded")
	}
	a, err := NewKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two random keys are equal")
	}
	if !bytes.Equal(a[HalfSize:], KeyTail[:]) {
		t.Errorf("random key tail = %x, want %x", a[HalfSize:], KeyTail)
	}
}

func TestKeyFromHalf(t *testing.T) {
	key, err := KeyFromHalf([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	if err != nil {
		t.Fatal(err)
	}
	if want := unhex(t, "0102030405060708c8279301a16c3197"); !bytes.Equal(key[:], want) {
		t.Errorf("KeyFromHalf = %x, want %x", key, want)
	}
	for _, n := range []int{0, 7, 9, 16} {
		if _, err := KeyFromHalf(make([]byte, n)); err == nil {
			t.Errorf("KeyFromHalf(%d bytes) succeeded", n)
		}
	}
}

// Worked out by hand from Encryption.encrypt: e[i] = p[i] ^ key[i&15] ^ e[i-1].
func TestEncryptGolden(t *testing.T) {
	c := NewClientCipher(testKey())

	// 10^A0 = B0; 20^01^B0 = 91; 30^02^91 = A3.
	b := []byte{0x10, 0x20, 0x30}
	c.Encrypt(b)
	if want := []byte{0xb0, 0x91, 0xa3}; !bytes.Equal(b, want) {
		t.Errorf("Encrypt = %x, want %x", b, want)
	}
	// Offset 0x0b0a0908 + 3.
	if want := unhex(t, "0b090a0b"); !bytes.Equal(c.out[8:12], want) {
		t.Errorf("out key[8:12] = %x, want %x", c.out[8:12], want)
	}

	d := NewClientCipher(testKey())
	d.Decrypt(b)
	if want := []byte{0x10, 0x20, 0x30}; !bytes.Equal(b, want) {
		t.Errorf("Decrypt = %x, want %x", b, want)
	}
}

// Zero bytes encrypt to the running XOR of the key. Byte 16 wraps to
// key[0]. After 17 bytes the offset 0x0b0a0908 becomes 0x0b0a0919, and the
// next packet uses 0x19 as key[8].
func TestKeyWrapAndOffset(t *testing.T) {
	c := NewClientCipher(testKey())

	first := make([]byte, 17)
	c.Encrypt(first)
	want := unhex(t, "a0a1a3a0a4a1a7a0a8a1aba0aca1afa000")
	if !bytes.Equal(first, want) {
		t.Errorf("first packet = %x, want %x", first, want)
	}
	if want := unhex(t, "19090a0b"); !bytes.Equal(c.out[8:12], want) {
		t.Errorf("out key[8:12] after 17 bytes = %x, want %x", c.out[8:12], want)
	}

	second := make([]byte, 9)
	c.Encrypt(second)
	// Same as before up to byte 7; byte 8 is 0x19 ^ 0xa0 = 0xb9.
	if want := unhex(t, "a0a1a3a0a4a1a7a0b9"); !bytes.Equal(second, want) {
		t.Errorf("second packet = %x, want %x", second, want)
	}
	if want := unhex(t, "22090a0b"); !bytes.Equal(c.out[8:12], want) {
		t.Errorf("out key[8:12] after 26 bytes = %x, want %x", c.out[8:12], want)
	}
	// Only bytes 8-11 change, and only the output key.
	k := testKey()
	if !bytes.Equal(c.out[:8], k[:8]) || !bytes.Equal(c.out[12:], k[12:]) {
		t.Errorf("out key = %x: bytes outside 8-11 changed", c.out)
	}
	if c.in != k {
		t.Errorf("in key = %x, want it unchanged (%x)", c.in, k)
	}
}

func TestOffsetOnRealKeyAndWrap(t *testing.T) {
	key, err := KeyFromHalf(make([]byte, HalfSize))
	if err != nil {
		t.Fatal(err)
	}
	c := NewClientCipher(key)
	c.Decrypt(make([]byte, 10))
	// c8 27 93 01 = 0x019327c8; + 10 = 0x019327d2.
	if want := unhex(t, "d2279301"); !bytes.Equal(c.in[8:12], want) {
		t.Errorf("in key[8:12] = %x, want %x", c.in[8:12], want)
	}

	var k [KeySize]byte
	copy(k[8:12], []byte{0xff, 0xff, 0xff, 0xff})
	c = NewClientCipher(k)
	c.Encrypt(make([]byte, 2))
	if want := unhex(t, "01000000"); !bytes.Equal(c.out[8:12], want) {
		t.Errorf("offset after wrap = %x, want %x", c.out[8:12], want)
	}
}

// The server sends the KeyPacket plain; that call only turns the cipher on
// and does not move the key. Client packets before that stay plain too.
func TestServerFirstPacketIsPlain(t *testing.T) {
	s := NewServerCipher(testKey())
	if s.Enabled() {
		t.Fatal("server cipher is on before the KeyPacket")
	}

	early := []byte{1, 2, 3}
	s.Decrypt(early)
	if !bytes.Equal(early, []byte{1, 2, 3}) || s.in != testKey() {
		t.Error("Decrypt changed data or key before the KeyPacket")
	}

	keyPacket := []byte{0x00, 0x01, 1, 2, 3, 4, 5, 6, 7, 8}
	s.Encrypt(keyPacket)
	if !bytes.Equal(keyPacket, []byte{0x00, 0x01, 1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("KeyPacket was encrypted: %x", keyPacket)
	}
	if !s.Enabled() {
		t.Fatal("server cipher is off after the KeyPacket")
	}
	if s.out != testKey() {
		t.Errorf("out key moved on the KeyPacket: %x", s.out)
	}

	// The next packet is encrypted with the unchanged key.
	next := make([]byte, 17)
	s.Encrypt(next)
	if want := unhex(t, "a0a1a3a0a4a1a7a0a8a1aba0aca1afa000"); !bytes.Equal(next, want) {
		t.Errorf("first encrypted packet = %x, want %x", next, want)
	}
}

func TestEmptyPacket(t *testing.T) {
	c := NewClientCipher(testKey())
	c.Encrypt(nil)
	c.Decrypt([]byte{})
	if c.in != testKey() || c.out != testKey() {
		t.Error("an empty packet moved a key")
	}
	s := NewServerCipher(testKey())
	s.Encrypt(nil) // the first call turns the cipher on even when empty
	if !s.Enabled() {
		t.Error("first Encrypt(nil) did not turn the cipher on")
	}
}

func TestServerClientRoundTrip(t *testing.T) {
	key, err := NewKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServerCipher(key)

	keyPacket := append([]byte{0x00, 0x01}, key[:HalfSize]...)
	server.Encrypt(keyPacket)
	clientKey, err := KeyFromHalf(keyPacket[2 : 2+HalfSize])
	if err != nil {
		t.Fatal(err)
	}
	client := NewClientCipher(clientKey)

	r := rand.New(rand.NewPCG(1, 2))
	for i := range 500 {
		plain := make([]byte, 1+r.IntN(1000))
		for j := range plain {
			plain[j] = byte(r.Uint32())
		}

		up := slices.Clone(plain)
		client.Encrypt(up)
		if len(plain) >= KeySize && bytes.Equal(up, plain) {
			t.Fatalf("packet %d: client packet not encrypted", i)
		}
		server.Decrypt(up)
		if !bytes.Equal(up, plain) {
			t.Fatalf("packet %d: server read %x, want %x", i, up, plain)
		}

		down := slices.Clone(plain)
		server.Encrypt(down)
		client.Decrypt(down)
		if !bytes.Equal(down, plain) {
			t.Fatalf("packet %d: client read %x, want %x", i, down, plain)
		}
	}
	if server.in != client.out || server.out != client.in {
		t.Error("server and client keys are out of step")
	}
	if server.in == key || server.out == key {
		t.Error("keys did not move")
	}
}

func FuzzDecrypt(f *testing.F) {
	f.Add([]byte{}, []byte{})
	f.Add(make([]byte, KeySize), make([]byte, 40))
	f.Fuzz(func(t *testing.T, rawKey, data []byte) {
		var key [KeySize]byte
		copy(key[:], rawKey)
		c := NewClientCipher(key)
		b := slices.Clone(data)
		c.Decrypt(b)
		NewClientCipher(key).Encrypt(b)
		if !bytes.Equal(b, data) {
			t.Fatalf("Encrypt(Decrypt(x)) = %x, want %x", b, data)
		}
		s := NewServerCipher(key)
		s.Decrypt(slices.Clone(data))
		s.Encrypt(slices.Clone(data))
		s.Decrypt(slices.Clone(data))
	})
}

func BenchmarkEncrypt(b *testing.B) {
	key, err := NewKey(nil)
	if err != nil {
		b.Fatal(err)
	}
	c := NewClientCipher(key)
	buf := make([]byte, 256)
	b.SetBytes(int64(len(buf)))
	for b.Loop() {
		c.Encrypt(buf)
	}
}
