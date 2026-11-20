package testclient

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/session"
)

// mobiusCharSelectionInfo writes chars in CharSelectionInfo.writeImpl
// order.
func mobiusCharSelectionInfo(chars []CharInfo) []byte {
	w := msg(0x13).i32(int32(len(chars)))
	for _, c := range chars {
		w.str(c.Name).i32(c.ObjectID).str(c.Account).i32(c.SessionID).i32(c.ClanID)
		w.i32(c.BuilderLevel).i32(c.Sex).i32(c.Race).i32(c.BaseClassID).i32(c.GameServer)
		w.i32(c.X).i32(c.Y).i32(c.Z)
		w.f64(c.CurrentHP).f64(c.CurrentMP).i32(c.SP).i64(c.Exp).i32(c.Level).i32(c.Karma)
		for _, v := range c.Reserved {
			w.i32(v)
		}
		for _, v := range c.EquipObjectIDs {
			w.i32(v)
		}
		for _, v := range c.EquipItemIDs {
			w.i32(v)
		}
		w.i32(c.HairStyle).i32(c.HairColor).i32(c.Face).f64(c.MaxHP).f64(c.MaxMP)
		w.i32(c.DeleteTimer).i32(c.ClassID).i32(b2i(c.Active)).u8(c.Enchant).i32(c.AugmentationID)
	}
	return w.b
}

// mobiusNewCharacterSuccess writes templates in
// NewCharacterSuccess.writeImpl order, with Mobius's fixed 0x46 and 0x0a
// around each base stat.
func mobiusNewCharacterSuccess(templates []CharTemplate) []byte {
	w := msg(0x17).i32(int32(len(templates)))
	for _, t := range templates {
		w.i32(t.Race).i32(t.ClassID)
		w.i32(0x46).i32(t.STR.Base).i32(0x0a)
		w.i32(0x46).i32(t.DEX.Base).i32(0x0a)
		w.i32(0x46).i32(t.CON.Base).i32(0x0a)
		w.i32(0x46).i32(t.INT.Base).i32(0x0a)
		w.i32(0x46).i32(t.WIT.Base).i32(0x0a)
		w.i32(0x46).i32(t.MEN.Base).i32(0x0a)
	}
	return w.b
}

// mobiusCharacterCreate writes a request in CharacterCreate.readImpl
// order.
func mobiusCharacterCreate(r CreateRequest) []byte {
	return msg(0x0b).str(r.Name).
		i32(r.Race).i32(r.Sex).i32(r.ClassID).
		i32(r.INT).i32(r.STR).i32(r.CON).i32(r.MEN).i32(r.DEX).i32(r.WIT).
		i32(r.HairStyle).i32(r.HairColor).i32(r.Face).b
}

// stat returns a NewCharacterSuccess stat with Mobius's fixed bounds.
func stat(base int32) StatRange { return StatRange{Max: 0x46, Base: base, Min: 0x0a} }

// testTemplates returns the nine starting classes in Mobius NewCharacter
// order. Human Fighter has its Mobius stats; the others have made-up ones.
func testTemplates() []CharTemplate {
	ts := []CharTemplate{
		{Race: 0, ClassID: 0, STR: stat(40), DEX: stat(30), CON: stat(43), INT: stat(21), WIT: stat(11), MEN: stat(25)},
	}
	for i, rc := range [][2]int32{{0, 10}, {1, 18}, {1, 25}, {2, 31}, {2, 38}, {3, 44}, {3, 49}, {4, 53}} {
		b := int32(i * 6)
		ts = append(ts, CharTemplate{
			Race: rc[0], ClassID: rc[1],
			STR: stat(b + 1), DEX: stat(b + 2), CON: stat(b + 3), INT: stat(b + 4), WIT: stat(b + 5), MEN: stat(b + 6),
		})
	}
	return ts
}

// testChar returns a character with every field set, so that a field read
// from the wrong place shows up.
func testChar(name string, objectID int32) CharInfo {
	c := CharInfo{
		Name: name, ObjectID: objectID, Account: "player1", SessionID: 0x0c, ClanID: 0x2a,
		Sex: 1, Race: 2, BaseClassID: 31, GameServer: 1,
		CurrentHP: 94.5, CurrentMP: 32.25, SP: 17, Exp: 1<<33 + 9, Level: 3, Karma: 5,
		HairStyle: 6, HairColor: 3, Face: 2, MaxHP: 101.75, MaxMP: 40.5,
		DeleteTimer: 0, ClassID: 32, Active: true, Enchant: 7, AugmentationID: 0x0102,
	}
	for i := range c.EquipObjectIDs {
		c.EquipObjectIDs[i] = objectID + 1 + int32(i)
		c.EquipItemIDs[i] = 1000 + int32(i)
	}
	return c
}

func TestEncodeGameRequests(t *testing.T) {
	keys := session.Keys{LoginOK1: 0x0a, LoginOK2: 0x0b, PlayOK1: 0x0c, PlayOK2: 0x0d}
	req := CreateRequest{
		Name: "Bob", Race: 1, Sex: 1, ClassID: 0x12,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 2, HairColor: 3, Face: 1,
	}
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{
			name: "ProtocolVersion 746",
			got:  encodeProtocolVersion(InterludeProtocol),
			want: []byte{0x00, 0xea, 0x02, 0x00, 0x00},
		},
		{
			name: "ProtocolVersion ping",
			got:  encodeProtocolVersion(PingProtocol),
			want: []byte{0x00, 0xfe, 0xff, 0xff, 0xff},
		},
		{
			// Mobius AuthLogin.readImpl: S name, d playKey2, d playKey1,
			// d loginKey1, d loginKey2.
			name: "AuthLogin",
			got:  encodeAuthLogin("Alice", keys),
			want: []byte{
				0x08,
				'a', 0, 'l', 0, 'i', 0, 'c', 0, 'e', 0, 0, 0,
				0x0d, 0, 0, 0, // PlayOK2
				0x0c, 0, 0, 0, // PlayOK1
				0x0a, 0, 0, 0, // LoginOK1
				0x0b, 0, 0, 0, // LoginOK2
			},
		},
		{
			name: "NewCharacter",
			got:  encodeNewCharacter(),
			want: []byte{0x0e},
		},
		{
			// Mobius CharacterCreate.readImpl: S name, d race, d sex,
			// d class, d INT, d STR, d CON, d MEN, d DEX, d WIT,
			// d hair style, d hair color, d face.
			name: "CharacterCreate",
			got:  encodeCharacterCreate(req),
			want: []byte{
				0x0b,
				'B', 0, 'o', 0, 'b', 0, 0, 0,
				1, 0, 0, 0, // race
				1, 0, 0, 0, // sex
				0x12, 0, 0, 0, // class
				21, 0, 0, 0, // INT
				40, 0, 0, 0, // STR
				43, 0, 0, 0, // CON
				25, 0, 0, 0, // MEN
				30, 0, 0, 0, // DEX
				11, 0, 0, 0, // WIT
				2, 0, 0, 0, // hair style
				3, 0, 0, 0, // hair color
				1, 0, 0, 0, // face
			},
		},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}
	if got, want := encodeCharacterCreate(req), mobiusCharacterCreate(req); !bytes.Equal(got, want) {
		t.Errorf("CharacterCreate differs from the test builder:\n got % x\nwant % x", got, want)
	}
}

func TestParseKeyPacket(t *testing.T) {
	// KeyPacket.writeImpl: c 0x00, c result, 8 key bytes, d encryption,
	// d server ID, c 1, d obfuscation key.
	body := []byte{
		0x00,
		0x01,
		0x9a, 0x3c, 0x51, 0xe7, 0x02, 0xb4, 0x6d, 0xf8,
		0x01, 0, 0, 0,
		0x02, 0, 0, 0,
		0x01,
		0, 0, 0, 0,
	}
	got, err := parseKeyPacket(body)
	if err != nil {
		t.Fatal(err)
	}
	want := KeyPacket{Result: 1, KeyHalf: testKeyHalf, Encryption: true, ServerID: 2, Unknown: 1}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if b := mobiusKeyPacket(1, testKeyHalf, true, 2); !bytes.Equal(b, body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", b, body)
	}

	bad := func(i int, v byte) []byte {
		b := bytes.Clone(body)
		b[i] = v
		return b
	}
	errTests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"extra byte", append(bytes.Clone(body), 0), "1 bytes after the last field"},
		{"short", body[:len(body)-1], "short packet"},
		{"result 2", bad(1, 2), "result is 2"},
		{"encryption flag 2", bad(10, 2), "encryption flag is 2"},
		{"wrong opcode", bad(0, 0x13), "opcode 0x13"},
	}
	for _, tt := range errTests {
		_, err := parseKeyPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
		}
	}
}

// TestCharSelectionInfoMobiusLayout writes one character exactly as the
// lines of Mobius CharSelectionInfo.writeImpl do, with a different value
// in every field, and checks each field by name.
func TestCharSelectionInfoMobiusLayout(t *testing.T) {
	paperdoll := []string{
		"UNDER", "REAR", "LEAR", "NECK", "RFINGER", "LFINGER", "HEAD", "RHAND", "LHAND",
		"GLOVES", "CHEST", "LEGS", "FEET", "CLOAK", "RHAND", "HAIR", "HAIR2",
	}
	objID := map[string]int32{}
	itemID := map[string]int32{}
	for i, s := range paperdoll {
		if _, ok := objID[s]; !ok {
			objID[s] = 0x7000 + int32(i)
			itemID[s] = 0x8000 + int32(i)
		}
	}

	w := msg(0x13)
	w.i32(1)               // Created character count
	w.str("Hero")          // Character name
	w.i32(0x10000001)      // Character ID
	w.str("acc")           // Account name
	w.i32(0x0badf00d)      // Account ID
	w.i32(77)              // Clan ID
	w.i32(0)               // Builder level
	w.i32(1)               // Sex
	w.i32(4)               // Race
	w.i32(0x35)            // Base class ID
	w.i32(1)               // GameServerName
	w.i32(0).i32(0).i32(0) // three zeros
	w.f64(123.5)           // Current HP
	w.f64(45.25)           // Current MP
	w.i32(999)             // SP
	w.i64(1<<40 + 5)       // Exp
	w.i32(20)              // Level
	w.i32(3)               // Karma
	for range 9 {          // nine zeros
		w.i32(0)
	}
	for _, s := range paperdoll { // paperdoll object IDs
		w.i32(objID[s])
	}
	for _, s := range paperdoll { // paperdoll item IDs
		w.i32(itemID[s])
	}
	w.i32(4)          // Hair style
	w.i32(2)          // Hair color
	w.i32(1)          // Face
	w.f64(200.75)     // Max HP
	w.f64(100.5)      // Max MP
	w.i32(3600)       // Delete timer, in seconds
	w.i32(0x36)       // Class ID
	w.i32(1)          // Active
	w.u8(7)           // Enchant effect
	w.i32(0x12345678) // Augmentation ID

	p, err := decodeGamePacket(w.b)
	if err != nil {
		t.Fatal(err)
	}
	list, ok := p.(CharacterList)
	if !ok || len(list.Characters) != 1 {
		t.Fatalf("got %#v, want a CharacterList with one character", p)
	}
	c := list.Characters[0]
	checks := []struct {
		field     string
		got, want any
	}{
		{"Name", c.Name, "Hero"},
		{"ObjectID", c.ObjectID, int32(0x10000001)},
		{"Account", c.Account, "acc"},
		{"SessionID", c.SessionID, int32(0x0badf00d)},
		{"ClanID", c.ClanID, int32(77)},
		{"Sex", c.Sex, int32(1)},
		{"Race", c.Race, int32(4)},
		{"BaseClassID", c.BaseClassID, int32(0x35)},
		{"GameServer", c.GameServer, int32(1)},
		{"CurrentHP", c.CurrentHP, 123.5},
		{"CurrentMP", c.CurrentMP, 45.25},
		{"SP", c.SP, int32(999)},
		{"Exp", c.Exp, int64(1<<40 + 5)},
		{"Level", c.Level, int32(20)},
		{"Karma", c.Karma, int32(3)},
		{"HairStyle", c.HairStyle, int32(4)},
		{"HairColor", c.HairColor, int32(2)},
		{"Face", c.Face, int32(1)},
		{"MaxHP", c.MaxHP, 200.75},
		{"MaxMP", c.MaxMP, 100.5},
		{"DeleteTimer", c.DeleteTimer, int32(3600)},
		{"ClassID", c.ClassID, int32(0x36)},
		{"Active", c.Active, true},
		{"Enchant", c.Enchant, uint8(7)},
		{"AugmentationID", c.AugmentationID, int32(0x12345678)},
	}
	for _, ch := range checks {
		if ch.got != ch.want {
			t.Errorf("%s = %v, want %v", ch.field, ch.got, ch.want)
		}
	}
	slots := map[int]string{
		SlotUnder: "UNDER", SlotREar: "REAR", SlotLEar: "LEAR", SlotNeck: "NECK",
		SlotRFinger: "RFINGER", SlotLFinger: "LFINGER", SlotHead: "HEAD", SlotRHand: "RHAND",
		SlotLHand: "LHAND", SlotGloves: "GLOVES", SlotChest: "CHEST", SlotLegs: "LEGS",
		SlotFeet: "FEET", SlotCloak: "CLOAK", SlotRHand2: "RHAND", SlotHair: "HAIR", SlotHair2: "HAIR2",
	}
	if len(slots) != NumEquipSlots || NumEquipSlots != 17 {
		t.Fatalf("%d slot names, NumEquipSlots %d, want 17", len(slots), NumEquipSlots)
	}
	for slot, s := range slots {
		if c.EquipObjectIDs[slot] != objID[s] || c.EquipItemIDs[slot] != itemID[s] {
			t.Errorf("slot %d (%s): object %#x item %#x, want %#x %#x",
				slot, s, c.EquipObjectIDs[slot], c.EquipItemIDs[slot], objID[s], itemID[s])
		}
	}
}

func TestDecodeCharSelectionInfo(t *testing.T) {
	banned := testChar("Banned", 0x10000100)
	banned.DeleteTimer = -1
	banned.Active = false
	banned.Enchant = 127
	chars := []CharInfo{testChar("Hero", 0x10000001), banned}

	tests := []struct {
		name string
		body []byte
		want CharacterList
	}{
		{"empty", []byte{0x13, 0, 0, 0, 0}, CharacterList{}},
		{"two characters", mobiusCharSelectionInfo(chars), CharacterList{Characters: chars}},
	}
	for _, tt := range tests {
		got, err := decodeGamePacket(tt.body)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, tt.want)
		}
	}

	list := CharacterList{Characters: chars}
	if c, ok := list.Active(); !ok || c.Name != "Hero" {
		t.Errorf("Active() = %q, %v", c.Name, ok)
	}
	if c, ok := list.Find("Banned"); !ok || c.DeleteTimer != -1 {
		t.Errorf("Find(Banned) = %+v, %v", c, ok)
	}
}

func TestDecodeCharSelectionInfoErrors(t *testing.T) {
	one := mobiusCharSelectionInfo([]CharInfo{testChar("Hero", 1)})
	activeAt := len(one) - 4 - 1 - 4 // before augmentation and enchant
	badActive := bytes.Clone(one)
	badActive[activeAt] = 2
	countTwo := bytes.Clone(one)
	countTwo[1] = 2

	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"negative count", []byte{0x13, 0xff, 0xff, 0xff, 0xff}, "character count -1"},
		{"extra byte", append(bytes.Clone(one), 0), "1 bytes after the last field"},
		{"truncated", one[:len(one)-1], "character 0 of 1"},
		{"count larger than the list", countTwo, "character 1 of 2"},
		{"active flag 2", badActive, "active flag is 2"},
	}
	for _, tt := range tests {
		_, err := decodeGamePacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
		}
	}
}

func TestDecodeNewCharacterSuccess(t *testing.T) {
	templates := testTemplates()
	got, err := decodeGamePacket(mobiusNewCharacterSuccess(templates))
	if err != nil {
		t.Fatal(err)
	}
	want := NewCharacterSuccess{Templates: templates}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	// The first template, byte by byte: race, class, then (0x46, base,
	// 0x0a) for STR, DEX, CON, INT, WIT, MEN.
	body := []byte{0x17, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for _, base := range []byte{40, 30, 43, 21, 11, 25} {
		body = append(body, 0x46, 0, 0, 0, base, 0, 0, 0, 0x0a, 0, 0, 0)
	}
	got, err = decodeGamePacket(body)
	if err != nil {
		t.Fatal(err)
	}
	want = NewCharacterSuccess{Templates: templates[:1]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Human Fighter:\n got %+v\nwant %+v", got, want)
	}
}

func TestDecodeSmallGamePackets(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want any
	}{
		{"CharCreateOk", []byte{0x19, 1, 0, 0, 0}, CharCreateOk{Result: 1}},
		{"CharCreateFail", []byte{0x1a, 2, 0, 0, 0}, CharCreateFail{Reason: CreateFailNameExists}},
		{"LoginFail", []byte{0x14, 1, 0, 0, 0}, GameLoginFail{Reason: GameLoginFailSystemErrorLoginLater}},
	}
	for _, tt := range tests {
		got, err := decodeGamePacket(tt.body)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: got %#v, want %#v", tt.name, got, tt.want)
		}
	}
}

func TestDecodeGamePacketErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		// 0x08 is not an Interlude server opcode.
		{"unknown opcode", []byte{0x08, 1, 2}, "unknown packet (opcode 0x08"},
		{"KeyPacket after the handshake", mobiusKeyPacket(1, testKeyHalf, true, 1), "KeyPacket (opcode 0x00"},
		{"empty", nil, "empty body"},
		{"CharCreateFail with an extra byte", []byte{0x1a, 2, 0, 0, 0, 0}, "1 bytes after the last field"},
		{"short CharCreateOk", []byte{0x19, 1}, "short packet"},
	}
	for _, tt := range tests {
		_, err := decodeGamePacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
		}
	}
	_, err := decodeGamePacket([]byte{0x08})
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Opcode != 0x08 || upe.Server != "game" {
		t.Errorf("unknown opcode: got %#v", err)
	}
}
