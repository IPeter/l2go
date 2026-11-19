package serverpackets

import (
	"bytes"
	"encoding/hex"
	"net/netip"
	"strings"
	"testing"
)

// The golden layouts below are written by hand from the writeImpl methods
// of the Mobius Interlude login server packets. Every value is
// little-endian; writeByte is one byte, writeShort two, writeInt four.

// unhex decodes a hex string that may contain spaces and newlines.
func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatalf("bad golden hex: %v", err)
	}
	return b
}

func seq(start byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = start + byte(i)
	}
	return b
}

func checkGolden(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("%s:\n got %x\nwant %x", name, got, want)
	}
}

// Init.writeImpl: c 0x00, d session ID, d 0xc621, b modulus (0x80), four GG
// ints, b Blowfish key (16), c 0.
func TestInitLayout(t *testing.T) {
	modulus := seq(0x00, 128)
	key := seq(0xa0, 16)
	got, err := NewInit(-2023406815, modulus, key) // 0x87654321
	if err != nil {
		t.Fatal(err)
	}
	var want []byte
	want = append(want, 0x00)
	want = append(want, unhex(t, "21 43 65 87")...) // session ID
	want = append(want, unhex(t, "21 c6 00 00")...) // protocol revision 0xc621
	want = append(want, modulus...)
	want = append(want, unhex(t, `
		4e 95 dd 29
		fc 9c c3 77
		20 b6 ad 97
		f7 e0 bd 07`)...)
	want = append(want, key...)
	want = append(want, 0x00)
	checkGolden(t, "Init", got, want)
	if len(got) != 170 {
		t.Errorf("Init is %d bytes, want 170", len(got))
	}
}

func TestInitRejectsBadSizes(t *testing.T) {
	if _, err := NewInit(1, make([]byte, 127), make([]byte, 16)); err == nil {
		t.Error("NewInit accepted a 127-byte modulus")
	}
	if _, err := NewInit(1, make([]byte, 128), make([]byte, 15)); err == nil {
		t.Error("NewInit accepted a 15-byte key")
	}
}

// GGAuth.writeImpl: c 0x0b, d response, d 0 x4.
func TestGGAuthLayout(t *testing.T) {
	checkGolden(t, "GGAuth", NewGGAuth(0x11223344), unhex(t, `
		0b
		44 33 22 11
		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00`))
}

// LoginOk.writeImpl: c 0x03, d loginOk1, d loginOk2, d 0, d 0, d 0x3ea,
// d 0, d 0, d 0, b 16 zero bytes.
func TestLoginOkLayout(t *testing.T) {
	checkGolden(t, "LoginOk", NewLoginOk(0x01020304, -1), unhex(t, `
		03
		04 03 02 01
		ff ff ff ff
		00 00 00 00
		00 00 00 00
		ea 03 00 00
		00 00 00 00
		00 00 00 00
		00 00 00 00
		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00`))
}

// LoginFail.writeImpl: c 0x01, c reason.
func TestLoginFailLayout(t *testing.T) {
	tests := []struct {
		reason LoginFailReason
		want   string
	}{
		{LoginFailSystemErrorLoginLater, "01 01"},
		{LoginFailUserOrPassWrong, "01 02"},
		{LoginFailAccessFailedTryLater, "01 04"},
		{LoginFailAccountInUse, "01 07"},
		{LoginFailServerOverloaded, "01 0f"},
		{LoginFailServerMaintenance, "01 10"},
		{LoginFailSystemError, "01 14"},
		{LoginFailAccessFailed, "01 15"},
	}
	for _, tt := range tests {
		checkGolden(t, "LoginFail", NewLoginFail(tt.reason), unhex(t, tt.want))
	}
}

// AccountKicked.writeImpl: c 0x02, d reason.
func TestAccountKickedLayout(t *testing.T) {
	tests := []struct {
		reason AccountKickedReason
		want   string
	}{
		{KickedDataStealer, "02 01 00 00 00"},
		{KickedGenericViolation, "02 08 00 00 00"},
		{Kicked7DaysSuspended, "02 10 00 00 00"},
		{KickedPermanentlyBanned, "02 20 00 00 00"},
	}
	for _, tt := range tests {
		checkGolden(t, "AccountKicked", NewAccountKicked(tt.reason), unhex(t, tt.want))
	}
}

// PlayFail.writeImpl: c 0x06, c reason.
func TestPlayFailLayout(t *testing.T) {
	tests := []struct {
		reason PlayFailReason
		want   string
	}{
		{PlayFailSystemErrorLoginLater, "06 01"},
		{PlayFailAccessFailedTryLater, "06 04"},
		{PlayFailAccountInUse, "06 07"},
		{PlayFailServerOverloaded, "06 0f"},
		{PlayFailServerMaintenance, "06 10"},
		{PlayFailSystemError, "06 14"},
		{PlayFailAccessFailed, "06 15"},
	}
	for _, tt := range tests {
		checkGolden(t, "PlayFail", NewPlayFail(tt.reason), unhex(t, tt.want))
	}
}

// PlayOk.writeImpl: c 0x07, d playOk1, d playOk2.
func TestPlayOkLayout(t *testing.T) {
	checkGolden(t, "PlayOk", NewPlayOk(0x0a0b0c0d, -2), unhex(t, `
		07
		0d 0c 0b 0a
		fe ff ff ff`))
}

// ServerList.writeImpl: c 0x04, c count, c last server, then per server
// c id, c c c c IP, d port, c age limit, c PvP, h players, h max players,
// c up, d server type, c brackets; then h 0 and c 0 (no character data).
func TestServerListLayout(t *testing.T) {
	servers := []ServerInfo{
		{
			ID: 1, IP: netip.MustParseAddr("192.168.1.2"), Port: 7777,
			AgeLimit: 18, PvP: true, Players: 3, MaxPlayers: 1000,
			Up: true, Type: ServerTypeNormal, Brackets: false,
		},
		{
			ID: 2, IP: netip.MustParseAddr("::ffff:10.0.0.5"), Port: 7778,
			AgeLimit: 0, PvP: false, Players: 0, MaxPlayers: 0xffff,
			Up: false, Type: ServerTypeFree | ServerTypeRelax, Brackets: true,
		},
		{
			ID: 255, IP: netip.MustParseAddr("2001:db8::1"), Port: 1,
		},
	}
	checkGolden(t, "ServerList", NewServerList(2, servers), unhex(t, `
		04 03 02

		01
		c0 a8 01 02
		61 1e 00 00
		12
		01
		03 00
		e8 03
		01
		01 00 00 00
		00

		02
		0a 00 00 05
		62 1e 00 00
		00
		00
		00 00
		ff ff
		00
		42 00 00 00
		01

		ff
		00 00 00 00
		01 00 00 00
		00
		00
		00 00
		00 00
		00
		00 00 00 00
		00

		00 00
		00`))
}

func TestServerListEmpty(t *testing.T) {
	checkGolden(t, "empty ServerList", NewServerList(0, nil), unhex(t, "04 00 00 00 00 00"))
}

func TestServerListCapsEntries(t *testing.T) {
	servers := make([]ServerInfo, 300)
	got := NewServerList(0, servers)
	if got[1] != MaxServers {
		t.Errorf("count = %d, want %d", got[1], MaxServers)
	}
	if want := 3 + 21*MaxServers + 3; len(got) != want {
		t.Errorf("length = %d, want %d", len(got), want)
	}
}
