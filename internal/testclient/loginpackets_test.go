package testclient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
)

func TestEncodeLoginRequests(t *testing.T) {
	block := bytes.Repeat([]byte{0xab}, 128)
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{
			// Mobius AuthGameGuard.readImpl: d session ID, then four d.
			name: "AuthGameGuard",
			got:  encodeAuthGameGuard(0x01020304),
			want: []byte{0x07, 0x04, 0x03, 0x02, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			// Mobius RequestAuthLogin.readImpl: the RSA block(s) right after
			// the opcode.
			name: "RequestAuthLogin",
			got:  encodeRequestAuthLogin(block),
			want: append([]byte{0x00}, block...),
		},
		{
			// Mobius RequestServerList.readImpl: d loginOk1, d loginOk2.
			name: "RequestServerList",
			got:  encodeRequestServerList(0x0a0b0c0d, -2),
			want: []byte{0x05, 0x0d, 0x0c, 0x0b, 0x0a, 0xfe, 0xff, 0xff, 0xff},
		},
		{
			// Mobius RequestServerLogin.readImpl: d skey1, d skey2, c server.
			name: "RequestServerLogin",
			got:  encodeRequestServerLogin(1, 2, 3),
			want: []byte{0x02, 1, 0, 0, 0, 2, 0, 0, 0, 3},
		},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}
}

// initPlain returns Init as the client holds it after decryption and the
// XOR pass: the body, zero padding, the XOR key at len-8 and a zero word.
func initPlain(body []byte) []byte {
	buf := make([]byte, logincrypt.StaticPacketSize(len(body)))
	copy(buf, body)
	binary.LittleEndian.PutUint32(buf[len(buf)-8:], 0xcafebabe)
	return buf
}

func TestParseInit(t *testing.T) {
	scrambled := make([]byte, 128)
	for i := range scrambled {
		scrambled[i] = byte(i * 7)
	}
	body := mobiusInit(-5, scrambled, testBlowfishKey[:])
	if len(body) != InitBodySize || InitBodySize != 170 {
		t.Fatalf("Mobius Init body is %d bytes, InitBodySize %d, want 170", len(body), InitBodySize)
	}
	plain := initPlain(body)
	if len(plain) != 192 {
		t.Fatalf("padded Init is %d bytes, want 192", len(plain))
	}

	got, err := parseInit(plain)
	if err != nil {
		t.Fatal(err)
	}
	want := Init{
		SessionID:   -5,
		Protocol:    0xc621,
		GameGuard:   MobiusGameGuard,
		BlowfishKey: testBlowfishKey,
	}
	copy(want.ScrambledModulus[:], scrambled)
	if got != want {
		t.Errorf("parseInit\n got %+v\nwant %+v", got, want)
	}
}

func TestParseInitErrors(t *testing.T) {
	good := mobiusInit(1, make([]byte, 128), testBlowfishKey[:])
	tests := []struct {
		name   string
		plain  func() []byte
		substr string
	}{
		{
			name: "wrong protocol",
			plain: func() []byte {
				b := initPlain(good)
				binary.LittleEndian.PutUint32(b[5:], 0x785a) // C1
				return b
			},
			substr: "protocol revision",
		},
		{
			name: "missing terminator",
			plain: func() []byte {
				b := initPlain(good)
				b[InitBodySize-1] = 1
				return b
			},
			substr: "last body byte",
		},
		{
			name: "padding not zero",
			plain: func() []byte {
				b := initPlain(good)
				b[InitBodySize] = 1
				return b
			},
			substr: "padding byte",
		},
		{
			name: "last word not zero",
			plain: func() []byte {
				b := initPlain(good)
				b[len(b)-1] = 1
				return b
			},
			substr: "after the XOR key",
		},
		{
			name: "extra 16 bytes after the modulus",
			plain: func() []byte {
				// The comment in Mobius Init mentions 0x10 zero bytes after
				// the key, but writeImpl does not write them.
				long := append(append(bytes.Clone(good[:137]), make([]byte, 16)...), good[137:]...)
				return initPlain(long)
			},
			substr: "Mobius sends 192",
		},
	}
	for _, tt := range tests {
		_, err := parseInit(tt.plain())
		if !errors.Is(err, ErrMalformed) || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want ErrMalformed containing %q", tt.name, err, tt.substr)
		}
	}

	// A first packet that is not Init.
	_, err := parseInit(initPlain(mobiusLoginFail(0x06)))
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Opcode != OpLoginFail || upe.Name != "LoginFail" {
		t.Errorf("LoginFail instead of Init: got %v", err)
	}
}

func TestDecodeLoginPackets(t *testing.T) {
	servers := []ServerInfo{
		{
			ID: 1, IP: netip.MustParseAddr("127.0.0.1"), Port: 7777, AgeLimit: 18, PvP: true,
			CurrentPlayers: 12, MaxPlayers: 1000, Up: true, ServerType: 1, Brackets: false,
		},
		{
			ID: 2, IP: netip.MustParseAddr("10.20.30.40"), Port: 7778, AgeLimit: 0, PvP: false,
			CurrentPlayers: 0, MaxPlayers: 500, Up: false, ServerType: 4 | 8, Brackets: true,
		},
	}
	chars := []ServerCharacters{
		{ServerID: 1, Characters: 3, DeleteTimers: []int32{100, 86400}},
		{ServerID: 2, Characters: 1},
	}
	tests := []struct {
		name string
		body []byte
		want any
	}{
		{"GGAuth", mobiusGGAuth(testSessionID), GGAuth{Response: testSessionID}},
		{
			"LoginOk", mobiusLoginOk(testLoginOK1, testLoginOK2),
			LoginOk{LoginOK1: testLoginOK1, LoginOK2: testLoginOK2, Fields: [6]int32{0, 0, 0x3ea, 0, 0, 0}},
		},
		{"LoginFail", mobiusLoginFail(0x02), LoginFail{Reason: LoginFailUserOrPassWrong}},
		{"AccountKicked", mobiusAccountKicked(0x20), AccountKicked{Reason: AccountKickedPermanentlyBanned}},
		{"PlayOk", mobiusPlayOk(testPlayOK1, testPlayOK2), PlayOk{PlayOK1: testPlayOK1, PlayOK2: testPlayOK2}},
		{"PlayFail", mobiusPlayFail(0x0f), PlayFail{Reason: PlayFailServerOverloaded}},
		{
			// Mobius with no character data: h 0, then a zero count.
			"ServerList without character data", mobiusServerList(1, servers, nil),
			ServerListResult{LastServer: 1, Servers: servers},
		},
		{
			"ServerList with character data", mobiusServerList(2, servers, chars),
			ServerListResult{LastServer: 2, Servers: servers, Characters: chars},
		},
		{"empty ServerList", mobiusServerList(0, nil, nil), ServerListResult{}},
	}
	for _, tt := range tests {
		got, err := decodeLoginPacket(sealedLogin(t, tt.body))
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s:\n got %#v\nwant %#v", tt.name, got, tt.want)
		}
	}
}

func TestServerListBytes(t *testing.T) {
	// One server, written out byte by byte from ServerList.writeImpl.
	body := []byte{
		0x04,         // opcode
		0x01,         // server count
		0x01,         // last server
		0x01,         // server ID
		127, 0, 0, 1, // IP
		0x61, 0x1e, 0, 0, // port 7777
		0x00,       // age limit
		0x00,       // PvP
		0x05, 0x00, // current players
		0x64, 0x00, // max players
		0x01,          // up
		0x01, 0, 0, 0, // server type
		0x00,       // brackets
		0x00, 0x00, // unknown short
		0x00, // no character data
	}
	got, err := decodeLoginPacket(sealedLogin(t, body))
	if err != nil {
		t.Fatal(err)
	}
	want := ServerListResult{
		LastServer: 1,
		Servers: []ServerInfo{{
			ID: 1, IP: netip.MustParseAddr("127.0.0.1"), Port: 7777,
			CurrentPlayers: 5, MaxPlayers: 100, Up: true, ServerType: 1,
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwant %#v", got, want)
	}
}

func TestDecodeLoginPacketErrors(t *testing.T) {
	server := []ServerInfo{{ID: 1, IP: netip.MustParseAddr("127.0.0.1"), Port: 7777, Up: true}}
	list := mobiusServerList(1, server, nil)
	// An empty list is 6 bytes. Without its 3-byte trailing section it
	// pads to a smaller size, so the client can tell.
	emptyList := mobiusServerList(0, nil, nil)

	tests := []struct {
		name   string
		plain  []byte
		substr string
	}{
		{
			name:   "unknown opcode",
			plain:  sealedLogin(t, []byte{0x42, 1, 2, 3}),
			substr: "opcode 0x42",
		},
		{
			name:   "Init after the first packet",
			plain:  sealedLogin(t, []byte{0x00, 1, 2, 3}),
			substr: "Init (opcode 0x00",
		},
		{
			// One extra byte keeps Mobius's padded size of a LoginFail.
			name:   "garbage in the padding",
			plain:  sealedLogin(t, append(mobiusLoginFail(2), 7)),
			substr: "want zero padding",
		},
		{
			// 8 extra zero bytes move the body into a bigger padded size.
			name:   "extra fields",
			plain:  sealedLogin(t, append(mobiusPlayOk(1, 2), make([]byte, 8)...)),
			substr: "Mobius sends",
		},
		{
			name:   "ServerList without trailing section",
			plain:  sealedLogin(t, emptyList[:len(emptyList)-3]),
			substr: "Mobius sends",
		},
		{
			name: "status byte out of range",
			plain: func() []byte {
				b := bytes.Clone(list)
				b[3+1+4+4+1+1+2+2] = 2 // the "up" byte of server 0
				return sealedLogin(t, b)
			}(),
			substr: "status is 2",
		},
		{
			name: "truncated LoginOk",
			plain: func() []byte {
				// Only the keys: Mobius pads 9 bytes to 24, and a full
				// LoginOk does not fit in that.
				return sealedLogin(t, msg(0x03).i32(1).i32(2).b)
			}(),
			substr: "LoginOk",
		},
	}
	for _, tt := range tests {
		_, err := decodeLoginPacket(tt.plain)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
		}
	}
	// The cut ServerList case only works if the cut changes the padded
	// size; check that so the test stays meaningful.
	if n := len(emptyList); logincrypt.SessionPacketSize(n) == logincrypt.SessionPacketSize(n-3) {
		t.Errorf("test data: ServerList of %d bytes and %d bytes pad to the same size", n, n-3)
	}
}

func TestCredentialsReachServer(t *testing.T) {
	// The client side of RequestAuthLogin, checked with the server side of
	// rsakey: encrypt as Login does, decrypt and parse as the server does.
	kp, err := testKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := rsakey.PublicKeyFromScrambled(kp.ScrambledModulus())
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{128, 256} {
		block, err := rsakey.CredentialsBlock("someuser", "secret", size)
		if err != nil {
			t.Fatal(err)
		}
		enc, err := rsakey.EncryptRaw(pub, block)
		if err != nil {
			t.Fatal(err)
		}
		pkt := encodeRequestAuthLogin(enc)
		if len(pkt) != 1+size {
			t.Fatalf("RequestAuthLogin is %d bytes, want %d", len(pkt), 1+size)
		}
		dec, err := kp.DecryptRaw(pkt[1:])
		if err != nil {
			t.Fatal(err)
		}
		cred, err := rsakey.ParseCredentials(dec)
		if err != nil {
			t.Fatal(err)
		}
		if cred.User != "someuser" || cred.Password != "secret" {
			t.Errorf("%d-byte block: got %+v", size, cred)
		}
	}
}
