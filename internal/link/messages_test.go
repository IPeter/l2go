package link

import (
	"bytes"
	"crypto/hmac"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

func sampleMessages() []message {
	ch := challengeMsg{version: protocolVersion}
	for i := range ch.challenge {
		ch.challenge[i] = byte(i)
	}
	reg := registerMsg{version: protocolVersion, serverID: 7, maxPlayers: 1000}
	for i := range reg.mac {
		reg.mac[i] = byte(255 - i)
	}
	return []message{
		ch,
		reg,
		registerResultMsg{},
		registerResultMsg{reason: ReasonWrongSecret},
		registerResultMsg{reason: ReasonAlreadyRegistered},
		statusMsg{players: 12, maxPlayers: 1000},
		statusMsg{players: math.MaxUint16, maxPlayers: math.MaxUint16},
		verifySessionMsg{
			requestID: 42,
			account:   "alice",
			keys:      session.Keys{LoginOK1: 1, LoginOK2: -2, PlayOK1: math.MaxInt32, PlayOK2: math.MinInt32},
		},
		verifySessionMsg{
			requestID: math.MaxUint32,
			account:   "Ünïcødé𝄞",
			keys:      session.Keys{LoginOK1: -1, LoginOK2: -1, PlayOK1: -1, PlayOK2: -1},
		},
		verifySessionResultMsg{requestID: 42, ok: true},
		verifySessionResultMsg{requestID: 43},
		pingMsg{},
		pongMsg{},
	}
}

func TestRoundTrip(t *testing.T) {
	for _, m := range sampleMessages() {
		got, err := decode(m.encode())
		if err != nil {
			t.Errorf("decode(%T) = %v", m, err)
			continue
		}
		if !reflect.DeepEqual(got, m) {
			t.Errorf("round trip of %T = %+v, want %+v", m, got, m)
		}
	}
}

// The wire format is fixed: both sides may be built from different commits.
func TestEncodeLayout(t *testing.T) {
	tests := []struct {
		m    message
		want []byte
	}{
		{registerResultMsg{reason: ReasonUnknownServer}, []byte{0x03, 0x02}},
		{statusMsg{players: 0x0102, maxPlayers: 0x0304}, []byte{0x04, 0x02, 0x01, 0x04, 0x03}},
		{
			verifySessionMsg{requestID: 1, account: "ab", keys: session.Keys{LoginOK1: 1, LoginOK2: 2, PlayOK1: 3, PlayOK2: -1}},
			[]byte{
				0x05,
				1, 0, 0, 0,
				'a', 0, 'b', 0, 0, 0,
				1, 0, 0, 0,
				2, 0, 0, 0,
				3, 0, 0, 0,
				0xff, 0xff, 0xff, 0xff,
			},
		},
		{verifySessionResultMsg{requestID: 0x01020304, ok: true}, []byte{0x06, 4, 3, 2, 1, 1}},
		{verifySessionResultMsg{requestID: 5}, []byte{0x06, 5, 0, 0, 0, 0}},
		{pingMsg{}, []byte{0x07}},
		{pongMsg{}, []byte{0x08}},
	}
	for _, tt := range tests {
		if got := tt.m.encode(); !bytes.Equal(got, tt.want) {
			t.Errorf("%T.encode() = % x, want % x", tt.m, got, tt.want)
		}
	}

	reg := registerMsg{version: 1, serverID: 9, maxPlayers: 0x0102}
	reg.mac[0], reg.mac[31] = 0xaa, 0xbb
	got := reg.encode()
	if len(got) != 1+2+1+32+2 {
		t.Fatalf("register is %d bytes, want 38", len(got))
	}
	if got[0] != 0x02 || got[1] != 1 || got[2] != 0 || got[3] != 9 || got[4] != 0xaa || got[35] != 0xbb || got[36] != 0x02 || got[37] != 0x01 {
		t.Errorf("register = % x", got)
	}

	ch := challengeMsg{version: 1}
	ch.challenge[0], ch.challenge[31] = 0xcc, 0xdd
	got = ch.encode()
	if len(got) != 1+2+32 || got[0] != 0x01 || got[1] != 1 || got[2] != 0 || got[3] != 0xcc || got[34] != 0xdd {
		t.Errorf("challenge = % x", got)
	}
}

func TestDecodeRejects(t *testing.T) {
	challenge := sampleMessages()[0].encode()
	register := sampleMessages()[1].encode()
	tests := []struct {
		name string
		body []byte
	}{
		{"empty", nil},
		{"zero opcode", []byte{0x00}},
		{"unknown opcode", []byte{0xff}},
		{"short challenge", challenge[:10]},
		{"long challenge", append(append([]byte(nil), challenge...), 0)},
		{"short register", register[:len(register)-1]},
		{"empty register result", []byte{opRegisterResult}},
		{"short status", []byte{opStatus, 1, 0, 2}},
		{"long status", []byte{opStatus, 1, 0, 2, 0, 0}},
		{"ping with data", []byte{opPing, 0}},
		{"pong with data", []byte{opPong, 0}},
		{"bad ok flag", []byte{opVerifySessionResult, 1, 0, 0, 0, 2}},
		{"short session result", []byte{opVerifySessionResult, 1, 0, 0, 0}},
		{"unterminated account", []byte{opVerifySession, 1, 0, 0, 0, 'a', 0}},
		{"missing keys", []byte{opVerifySession, 1, 0, 0, 0, 'a', 0, 0, 0}},
		{"empty account", verifySessionMsg{requestID: 1}.encode()},
		{"long account", verifySessionMsg{requestID: 1, account: strings.Repeat("a", maxAccountLen+1)}.encode()},
		{"trailing bytes", append(verifySessionMsg{requestID: 1, account: "a"}.encode(), 0)},
	}
	for _, tt := range tests {
		m, err := decode(tt.body)
		if !errors.Is(err, errMalformed) {
			t.Errorf("%s: decode = %+v, %v; want errMalformed", tt.name, m, err)
		}
		if m != nil {
			t.Errorf("%s: decode returned a message with an error", tt.name)
		}
	}
}

func TestValidAccount(t *testing.T) {
	tests := []struct {
		account string
		want    bool
	}{
		{"", false},
		{"a", true},
		{strings.Repeat("a", maxAccountLen), true},
		{strings.Repeat("a", maxAccountLen+1), false},
		{strings.Repeat("𝄞", maxAccountLen), true},
		{"a\x00b", false},
	}
	for _, tt := range tests {
		if got := validAccount(tt.account); got != tt.want {
			t.Errorf("validAccount(%q) = %v, want %v", tt.account, got, tt.want)
		}
	}
}

// The largest valid message must fit in maxMessageSize, or a long account
// name would break the link.
func TestLargestMessageFits(t *testing.T) {
	m := verifySessionMsg{requestID: math.MaxUint32, account: strings.Repeat("𝄞", maxAccountLen)}
	body := m.encode()
	if len(body) > maxMessageSize {
		t.Fatalf("largest VerifySession is %d bytes, max %d", len(body), maxMessageSize)
	}
	var buf bytes.Buffer
	if err := packets.WritePacket(&buf, body); err != nil {
		t.Fatal(err)
	}
	got, err := readMsg(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("readMsg = %+v, want %+v", got, m)
	}
}

func TestReadMsgRejectsOversizedPacket(t *testing.T) {
	var buf bytes.Buffer
	body := make([]byte, maxMessageSize+1)
	body[0] = opPing
	if err := packets.WritePacket(&buf, body); err != nil {
		t.Fatal(err)
	}
	if _, err := readMsg(&buf); !errors.Is(err, packets.ErrPacketSize) {
		t.Errorf("readMsg = %v, want ErrPacketSize", err)
	}
}

func TestComputeMAC(t *testing.T) {
	var c1, c2 [challengeSize]byte
	c2[0] = 1
	a := computeMAC([]byte("secret"), c1)
	if len(a) != macSize {
		t.Fatalf("MAC is %d bytes, want %d", len(a), macSize)
	}
	if !hmac.Equal(a, computeMAC([]byte("secret"), c1)) {
		t.Error("MAC is not deterministic")
	}
	if hmac.Equal(a, computeMAC([]byte("other"), c1)) {
		t.Error("MAC does not depend on the secret")
	}
	if hmac.Equal(a, computeMAC([]byte("secret"), c2)) {
		t.Error("MAC does not depend on the challenge")
	}
}

func TestReasonString(t *testing.T) {
	for _, r := range []Reason{reasonNone, ReasonWrongSecret, ReasonUnknownServer, ReasonAlreadyRegistered, ReasonUnsupportedVersion, 200} {
		if r.String() == "" {
			t.Errorf("Reason(%d).String() is empty", r)
		}
	}
	err := &RejectedError{Reason: ReasonWrongSecret}
	if !strings.Contains(err.Error(), "wrong secret") {
		t.Errorf("Error() = %q", err.Error())
	}
}

// FuzzDecode checks that decode never panics, and that whatever it accepts
// encodes back to a message that decodes to the same value.
func FuzzDecode(f *testing.F) {
	for _, m := range sampleMessages() {
		f.Add(m.encode())
	}
	f.Add([]byte{})
	f.Add([]byte{opVerifySession, 0xff, 0xff})
	f.Add([]byte{opVerifySession, 1, 0, 0, 0, 0x00, 0xd8, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 4, 0, 0, 0})
	f.Fuzz(func(t *testing.T, body []byte) {
		m, err := decode(body)
		if err != nil {
			if !errors.Is(err, errMalformed) {
				t.Fatalf("decode error %v does not wrap errMalformed", err)
			}
			if m != nil {
				t.Fatalf("decode returned %+v with an error", m)
			}
			return
		}
		again, err := decode(m.encode())
		if err != nil {
			t.Fatalf("decoding the re-encoded %T: %v", m, err)
		}
		if !reflect.DeepEqual(again, m) {
			t.Fatalf("re-encoded %T decodes to %+v, want %+v", m, again, m)
		}
	})
}
