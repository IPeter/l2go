package testclient

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
)

// wire builds packet bytes field by field, the way Mobius WriteBuffer
// does. It does not use internal/packets, so the client's encoders and
// decoders are checked against an independent encoding.
type wire struct{ b []byte }

// msg starts a packet with its opcode.
func msg(op byte) *wire { return &wire{b: []byte{op}} }

func (w *wire) u8(v byte) *wire    { w.b = append(w.b, v); return w }
func (w *wire) u16(v uint16) *wire { w.b = binary.LittleEndian.AppendUint16(w.b, v); return w }
func (w *wire) u32(v uint32) *wire { w.b = binary.LittleEndian.AppendUint32(w.b, v); return w }
func (w *wire) i32(v int32) *wire  { return w.u32(uint32(v)) }
func (w *wire) i64(v int64) *wire  { w.b = binary.LittleEndian.AppendUint64(w.b, uint64(v)); return w }
func (w *wire) f64(v float64) *wire {
	w.b = binary.LittleEndian.AppendUint64(w.b, math.Float64bits(v))
	return w
}
func (w *wire) raw(b ...byte) *wire { w.b = append(w.b, b...); return w }

// str writes a Java String as WriteBuffer.writeString does: UTF-16LE code
// units and a zero terminator.
func (w *wire) str(s string) *wire {
	for _, u := range utf16.Encode([]rune(s)) {
		w.u16(u)
	}
	return w.u16(0)
}

func b2i(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func b2u8(v bool) byte {
	if v {
		return 1
	}
	return 0
}

// Test values for the login side.
const (
	testSessionID int32 = 0x12345678
	testLoginOK1  int32 = 0x11111111
	testLoginOK2  int32 = -0x22222222
	testPlayOK1   int32 = 0x33333333
	testPlayOK2   int32 = 0x44444444
)

var testBlowfishKey = [16]byte{
	0x10, 0x21, 0x32, 0x43, 0x54, 0x65, 0x76, 0x87,
	0x98, 0xa9, 0xba, 0xcb, 0xdc, 0xed, 0xfe, 0x0f,
}

// testKeyPair is shared by all tests, because generating RSA keys is slow.
var testKeyPair = sync.OnceValues(rsakey.Generate)

// The Mobius login server packets, transcribed from their writeImpl.

func mobiusInit(sessionID int32, scrambled, blowfishKey []byte) []byte {
	return msg(0x00).
		i32(sessionID).
		u32(0x0000c621).
		raw(scrambled...).
		u32(0x29dd954e).u32(0x77c39cfc).u32(0x97adb620).u32(0x07bde0f7).
		raw(blowfishKey...).
		u8(0).b
}

func mobiusGGAuth(response int32) []byte {
	return msg(0x0b).i32(response).i32(0).i32(0).i32(0).i32(0).b
}

func mobiusLoginOk(k1, k2 int32) []byte {
	return msg(0x03).i32(k1).i32(k2).
		i32(0).i32(0).i32(0x3ea).i32(0).i32(0).i32(0).
		raw(make([]byte, 16)...).b
}

func mobiusLoginFail(reason byte) []byte { return msg(0x01).u8(reason).b }

func mobiusAccountKicked(reason int32) []byte { return msg(0x02).i32(reason).b }

func mobiusPlayOk(p1, p2 int32) []byte { return msg(0x07).i32(p1).i32(p2).b }

func mobiusPlayFail(reason byte) []byte { return msg(0x06).u8(reason).b }

func mobiusServerList(last byte, servers []ServerInfo, chars []ServerCharacters) []byte {
	w := msg(0x04).u8(byte(len(servers))).u8(last)
	for _, s := range servers {
		ip := s.IP.As4()
		w.u8(s.ID).raw(ip[:]...).i32(s.Port).u8(s.AgeLimit).u8(b2u8(s.PvP)).
			u16(s.CurrentPlayers).u16(s.MaxPlayers).u8(b2u8(s.Up)).i32(s.ServerType).u8(b2u8(s.Brackets))
	}
	w.u16(0)
	w.u8(byte(len(chars)))
	for _, c := range chars {
		w.u8(c.ServerID).u8(c.Characters).u8(byte(len(c.DeleteTimers)))
		for _, d := range c.DeleteTimers {
			w.i32(d)
		}
	}
	return w.b
}

// sealedLogin returns body as the client sees it after decrypting a
// session packet: padded to Mobius's size, with a checksum.
func sealedLogin(t *testing.T, body []byte) []byte {
	t.Helper()
	buf := make([]byte, logincrypt.SessionPacketSize(len(body)))
	copy(buf, body)
	if err := logincrypt.AppendChecksum(buf); err != nil {
		t.Fatal(err)
	}
	return buf
}

// fakeLogin is the server end of one login connection, built from the
// server half of logincrypt.
type fakeLogin struct {
	t    *testing.T
	conn net.Conn
	sess *logincrypt.Session
	key  *rsakey.KeyPair
}

// startFakeLogin listens on a local port and runs script for the first
// connection. It returns the address to dial. The test waits for the
// script before it ends.
func startFakeLogin(t *testing.T, script func(f *fakeLogin)) string {
	t.Helper()
	kp, err := testKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	t.Cleanup(func() { <-done })
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		sess, err := logincrypt.NewSession(testBlowfishKey[:], nil)
		if err != nil {
			t.Errorf("fake login server: %v", err)
			return
		}
		script(&fakeLogin{t: t, conn: conn, sess: sess, key: kp})
	}()
	return ln.Addr().String()
}

func (f *fakeLogin) fatalf(format string, args ...any) {
	f.t.Errorf("fake login server: "+format, args...)
	runtime.Goexit()
}

// send encrypts body (the first call as Init) and writes it.
func (f *fakeLogin) send(body []byte) {
	enc, err := f.sess.EncryptServerPacket(body)
	if err != nil {
		f.fatalf("encrypt: %v", err)
	}
	writeTestFrame(f.t, f.conn, enc, f.fatalf)
}

// sendInit sends Mobius's Init for this connection.
func (f *fakeLogin) sendInit() {
	f.send(mobiusInit(testSessionID, f.key.ScrambledModulus(), testBlowfishKey[:]))
}

// recv reads and decrypts a client packet. It returns the whole decrypted
// packet: body, padding and checksum.
func (f *fakeLogin) recv() []byte {
	b := readTestFrame(f.conn, f.fatalf)
	if err := f.sess.DecryptClientPacket(b); err != nil {
		f.fatalf("decrypt client packet: %v", err)
	}
	return b
}

// expect reads a client packet and checks that its body is want, padded as
// Mobius LoginEncryption pads a body of that size.
func (f *fakeLogin) expect(want []byte) {
	got := f.recv()
	if len(got) != logincrypt.SessionPacketSize(len(want)) {
		f.fatalf("client packet is %d bytes, want %d for a %d-byte body\n got % x\nwant % x",
			len(got), logincrypt.SessionPacketSize(len(want)), len(want), got, want)
	}
	if !bytes.Equal(got[:len(want)], want) {
		f.fatalf("client packet body\n got % x\nwant % x", got[:len(want)], want)
	}
	for i, b := range got[len(want) : len(got)-logincrypt.ChecksumSize] {
		if b != 0 {
			f.fatalf("padding byte %d is 0x%02x", len(want)+i, b)
		}
	}
}

// drain reads until the client closes the connection.
func (f *fakeLogin) drain() { _, _ = io.Copy(io.Discard, f.conn) }

// fakeGame is the server end of one game connection, built from the server
// half of gamecrypt.
type fakeGame struct {
	t      *testing.T
	conn   net.Conn
	cipher *gamecrypt.Cipher
}

// startFakeGame works like startFakeLogin for the game side.
func startFakeGame(t *testing.T, script func(f *fakeGame)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	t.Cleanup(func() { <-done })
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		script(&fakeGame{t: t, conn: conn})
	}()
	return ln.Addr().String()
}

func (f *fakeGame) fatalf(format string, args ...any) {
	f.t.Errorf("fake game server: "+format, args...)
	runtime.Goexit()
}

// testKeyHalf is the random half of the game key in the fake server.
var testKeyHalf = [8]byte{0x9a, 0x3c, 0x51, 0xe7, 0x02, 0xb4, 0x6d, 0xf8}

// mobiusKeyPacket is Mobius KeyPacket.writeImpl.
func mobiusKeyPacket(result byte, half [8]byte, encryption bool, serverID int32) []byte {
	return msg(0x00).u8(result).raw(half[:]...).i32(b2i(encryption)).i32(serverID).u8(1).i32(0).b
}

// sendKey sends KeyPacket in plain text and, with encryption on, turns on
// the server cipher the way Mobius does: the first Encrypt call leaves
// KeyPacket plain.
func (f *fakeGame) sendKey(result byte, encryption bool) {
	body := mobiusKeyPacket(result, testKeyHalf, encryption, 1)
	if encryption {
		key, err := gamecrypt.NewKey(bytes.NewReader(testKeyHalf[:]))
		if err != nil {
			f.fatalf("key: %v", err)
		}
		f.cipher = gamecrypt.NewServerCipher(key)
		f.cipher.Encrypt(body)
	}
	writeTestFrame(f.t, f.conn, body, f.fatalf)
}

// send encrypts body when the cipher is on and writes it.
func (f *fakeGame) send(body []byte) {
	b := bytes.Clone(body)
	if f.cipher != nil {
		f.cipher.Encrypt(b)
	}
	writeTestFrame(f.t, f.conn, b, f.fatalf)
}

// recv reads a client packet and decrypts it when the cipher is on.
func (f *fakeGame) recv() []byte {
	b := readTestFrame(f.conn, f.fatalf)
	if f.cipher != nil {
		f.cipher.Decrypt(b)
	}
	return b
}

// expect reads a client packet and checks that it is exactly want.
func (f *fakeGame) expect(want []byte) {
	got := f.recv()
	if !bytes.Equal(got, want) {
		f.fatalf("client packet\n got % x\nwant % x", got, want)
	}
}

// drain reads until the client closes the connection.
func (f *fakeGame) drain() { _, _ = io.Copy(io.Discard, f.conn) }

// writeTestFrame writes body with its 2-byte length prefix, which counts
// itself.
func writeTestFrame(t *testing.T, conn net.Conn, body []byte, fatalf func(string, ...any)) {
	t.Helper()
	frame := binary.LittleEndian.AppendUint16(nil, uint16(len(body)+2))
	frame = append(frame, body...)
	if _, err := conn.Write(frame); err != nil {
		fatalf("write: %v", err)
	}
}

// readTestFrame reads one length-prefixed packet body.
func readTestFrame(conn net.Conn, fatalf func(string, ...any)) []byte {
	var hdr [2]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		fatalf("read header: %v", err)
	}
	n := int(binary.LittleEndian.Uint16(hdr[:])) - 2
	if n <= 0 {
		fatalf("length prefix %d", n+2)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(conn, b); err != nil {
		fatalf("read body: %v", err)
	}
	return b
}
