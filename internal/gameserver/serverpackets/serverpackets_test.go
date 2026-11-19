package serverpackets

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// unhex decodes hex with spaces and line breaks.
func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// golden builds an expected packet field by field, like the Mobius
// WriteBuffer, without the production packets.Writer.
type golden struct{ b []byte }

func (g *golden) c(v byte) *golden { g.b = append(g.b, v); return g }

func (g *golden) d(v int32) *golden {
	g.b = binary.LittleEndian.AppendUint32(g.b, uint32(v))
	return g
}

func (g *golden) q(v int64) *golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, uint64(v))
	return g
}

func (g *golden) f(v float64) *golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, math.Float64bits(v))
	return g
}

func (g *golden) s(v string) *golden {
	for _, u := range utf16.Encode([]rune(v)) {
		g.b = binary.LittleEndian.AppendUint16(g.b, u)
	}
	g.b = append(g.b, 0, 0)
	return g
}

func checkPacket(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("%s:\n got % x\nwant % x", name, got, want)
	}
}

func TestKeyPacket(t *testing.T) {
	key, err := gamecrypt.KeyFromHalf([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})
	if err != nil {
		t.Fatal(err)
	}
	// Mobius KeyPacket.writeImpl: id, c result, 8 key bytes, d encryption,
	// d server ID, c 1, d obfuscation key.
	checkPacket(t, "accepted", NewKeyPacket(key, true, 5), unhex(t, `
		00
		01
		11 22 33 44 55 66 77 88
		01 00 00 00
		05 00 00 00
		01
		00 00 00 00`))
	checkPacket(t, "refused", NewKeyPacket(key, false, 1), unhex(t, `
		00
		00
		11 22 33 44 55 66 77 88
		01 00 00 00
		01 00 00 00
		01
		00 00 00 00`))
}

func TestCharCreateFailPacket(t *testing.T) {
	for code := CharCreateFailCreationFailed; code <= CharCreateFailChooseAnotherServer; code++ {
		want := []byte{0x1a, byte(code), 0x00, 0x00, 0x00}
		checkPacket(t, "CharCreateFail", NewCharCreateFailPacket(code), want)
	}
}

// The script reason names map to the Interlude codes of Mobius
// CharCreateFail.
func TestCharCreateFailReasonByName(t *testing.T) {
	tests := map[string]CharCreateFailReason{
		"creation_failed":      0x00,
		"too_many_characters":  0x01,
		"name_exists":          0x02,
		"name_too_long":        0x03,
		"incorrect_name":       0x04,
		"creation_not_allowed": 0x05,
	}
	for name, want := range tests {
		if got, ok := CharCreateFailReasonByName(name); !ok || got != want {
			t.Errorf("CharCreateFailReasonByName(%q) = %#x, %v; want %#x", name, got, ok, want)
		}
	}
	for _, name := range []string{"", "unknown", "Incorrect_Name", "choose_another_server"} {
		if _, ok := CharCreateFailReasonByName(name); ok {
			t.Errorf("CharCreateFailReasonByName(%q) found a code", name)
		}
	}
}

func TestCharCreateOkPacket(t *testing.T) {
	checkPacket(t, "CharCreateOk", NewCharCreateOkPacket(), unhex(t, "19 01000000"))
}

func TestLoginFailPacket(t *testing.T) {
	checkPacket(t, "LoginFail", NewLoginFailPacket(LoginFailSystemErrorLoginLater), unhex(t, "14 01000000"))
}

func TestNewCharacterSuccessPacket(t *testing.T) {
	humanFighter := data.Template{
		ClassID: 0, Race: 0,
		Stats: data.BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25},
	}
	orcMystic := data.Template{
		ClassID: 49, Race: 3,
		Stats: data.BaseStats{STR: 27, CON: 31, DEX: 24, INT: 31, WIT: 15, MEN: 42},
	}
	// Mobius NewCharacterSuccess.writeImpl: d count, then per template d
	// race, d class, and (0x46, base, 0x0a) for STR, DEX, CON, INT, WIT, MEN.
	want := unhex(t, `
		17
		02000000
		00000000 00000000
		46000000 28000000 0a000000
		46000000 1e000000 0a000000
		46000000 2b000000 0a000000
		46000000 15000000 0a000000
		46000000 0b000000 0a000000
		46000000 19000000 0a000000
		03000000 31000000
		46000000 1b000000 0a000000
		46000000 18000000 0a000000
		46000000 1f000000 0a000000
		46000000 1f000000 0a000000
		46000000 0f000000 0a000000
		46000000 2a000000 0a000000`)
	checkPacket(t, "NewCharacterSuccess", NewCharacterSuccessPacket([]data.Template{humanFighter, orcMystic}), want)
	checkPacket(t, "empty NewCharacterSuccess", NewCharacterSuccessPacket(nil), unhex(t, "17 00000000"))
}

func TestCharSelectionInfoEmpty(t *testing.T) {
	got := NewCharSelectionInfoPacket("player1", 1234, nil, time.Now())
	checkPacket(t, "empty CharSelectionInfo", got, unhex(t, "13 00000000"))
}

// wantChar is one character entry as Mobius CharSelectionInfo.writeImpl
// (lines 100-188) writes it. The equipment arrays are in packet order:
// under, rear, lear, neck, rfinger, lfinger, head, rhand, lhand, gloves,
// chest, legs, feet, cloak, rhand (again), hair, hair2.
type wantChar struct {
	name                 string
	objectID             int32
	clanID               int32
	sex, race, baseClass int32
	curHP, curMP         float64
	sp                   int32
	exp                  int64
	level, karma         int32
	objectIDs, itemIDs   [17]int32
	hairStyle, hairColor int32
	face                 int32
	maxHP, maxMP         float64
	deleteTimer          int32
	classID              int32
	active               bool
	enchant              byte
}

func (g *golden) char(account string, sessionID int32, w wantChar) {
	g.s(w.name).d(w.objectID).s(account).d(sessionID).d(w.clanID)
	g.d(0) // builder level
	g.d(w.sex).d(w.race).d(w.baseClass)
	g.d(1)           // game server name
	g.d(0).d(0).d(0) // x, y, z
	g.f(w.curHP).f(w.curMP)
	g.d(w.sp).q(w.exp).d(w.level).d(w.karma)
	for range 9 {
		g.d(0)
	}
	for _, v := range w.objectIDs {
		g.d(v)
	}
	for _, v := range w.itemIDs {
		g.d(v)
	}
	g.d(w.hairStyle).d(w.hairColor).d(w.face)
	g.f(w.maxHP).f(w.maxMP)
	g.d(w.deleteTimer).d(w.classID)
	if w.active {
		g.d(1)
	} else {
		g.d(0)
	}
	g.c(w.enchant)
	g.d(0) // augmentation ID
}

func TestCharSelectionInfo(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	chars := []storage.Character{
		{
			ObjectID: 0x10000000, Account: "player1", Name: "Frostwind",
			Race: 1, Sex: 1, ClassID: 18, BaseClassID: 18,
			HairStyle: 2, HairColor: 3, Face: 1, Level: 1,
			CurHP: 89, MaxHP: 89, CurMP: 30, MaxMP: 30, CurCP: 0, MaxCP: 35.6,
			LastAccess: now.Add(-time.Hour),
			Items: []storage.Item{
				{ObjectID: 0x10000001, ItemID: 10, Count: 1, Slot: storage.PaperdollNone},
				{ObjectID: 0x10000002, ItemID: 1146, Count: 1, Slot: storage.PaperdollChest},
				{ObjectID: 0x10000003, ItemID: 1147, Count: 1, Slot: storage.PaperdollLegs},
				{ObjectID: 0x10000004, ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand, Enchant: 3},
				{ObjectID: 0x10000005, ItemID: 5588, Count: 1, Slot: storage.PaperdollNone},
			},
		},
		{
			// Banned, played last, base class missing, a high enchant and
			// every equipment slot used.
			ObjectID: 0x10000010, Account: "player1", Name: "Banned1",
			Race: 1, Sex: 0, ClassID: 25, BaseClassID: 0,
			Level: 40, Exp: 1234567890123, SP: 77, Karma: 5, ClanID: 9,
			CurHP: 10.5, MaxHP: 1000.25, CurMP: 20.75, MaxMP: 500,
			AccessLevel: storage.AccessLevelBanned,
			DeleteTime:  now.Add(time.Hour),
			LastAccess:  now.Add(-time.Minute),
			Items: []storage.Item{
				{ObjectID: 100, ItemID: 1000, Slot: storage.PaperdollUnder},
				{ObjectID: 101, ItemID: 1001, Slot: storage.PaperdollLEar},
				{ObjectID: 102, ItemID: 1002, Slot: storage.PaperdollREar},
				{ObjectID: 103, ItemID: 1003, Slot: storage.PaperdollNeck},
				{ObjectID: 104, ItemID: 1004, Slot: storage.PaperdollLFinger},
				{ObjectID: 105, ItemID: 1005, Slot: storage.PaperdollRFinger},
				{ObjectID: 106, ItemID: 1006, Slot: storage.PaperdollHead},
				{ObjectID: 107, ItemID: 1007, Slot: storage.PaperdollRHand, Enchant: 200},
				{ObjectID: 108, ItemID: 1008, Slot: storage.PaperdollLHand},
				{ObjectID: 109, ItemID: 1009, Slot: storage.PaperdollGloves},
				{ObjectID: 110, ItemID: 1010, Slot: storage.PaperdollChest},
				{ObjectID: 111, ItemID: 1011, Slot: storage.PaperdollLegs},
				{ObjectID: 112, ItemID: 1012, Slot: storage.PaperdollFeet},
				{ObjectID: 113, ItemID: 1013, Slot: storage.PaperdollCloak},
				{ObjectID: 114, ItemID: 1014, Slot: storage.PaperdollFace},
				{ObjectID: 115, ItemID: 1015, Slot: storage.PaperdollHair},
				{ObjectID: 116, ItemID: 1016, Slot: storage.PaperdollHair2},
			},
		},
		{
			// Being deleted: 90.5 seconds left show as 90.
			ObjectID: 0x10000020, Account: "player1", Name: "Leaving",
			Race: 4, ClassID: 53, BaseClassID: 53, Level: 2,
			CurHP: 80, MaxHP: 80, CurMP: 30, MaxMP: 30,
			DeleteTime: now.Add(90*time.Second + 500*time.Millisecond),
		},
	}

	const account, sessionID = "player1", int32(-559038737)
	var g golden
	g.c(0x13).d(3)
	g.char(account, sessionID, wantChar{
		name: "Frostwind", objectID: 0x10000000,
		sex: 1, race: 1, baseClass: 18,
		curHP: 89, curMP: 30, level: 1,
		objectIDs: [17]int32{
			0, 0, 0, 0, 0, 0, 0, 0x10000004, 0, 0,
			0x10000002, 0x10000003, 0, 0, 0x10000004, 0, 0,
		},
		itemIDs: [17]int32{
			0, 0, 0, 0, 0, 0, 0, 2369, 0, 0,
			1146, 1147, 0, 0, 2369, 0, 0,
		},
		hairStyle: 2, hairColor: 3, face: 1,
		maxHP: 89, maxMP: 30, classID: 18, enchant: 3,
	})
	g.char(account, sessionID, wantChar{
		name: "Banned1", objectID: 0x10000010, clanID: 9,
		sex: 0, race: 1, baseClass: 25,
		curHP: 10.5, curMP: 20.75, sp: 77, exp: 1234567890123, level: 40, karma: 5,
		// The face slot (114) is not in the list; the right hand is twice.
		objectIDs: [17]int32{100, 102, 101, 103, 105, 104, 106, 107, 108, 109, 110, 111, 112, 113, 107, 115, 116},
		itemIDs:   [17]int32{1000, 1002, 1001, 1003, 1005, 1004, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1007, 1015, 1016},
		maxHP:     1000.25, maxMP: 500,
		deleteTimer: -1, classID: 25, active: true, enchant: 127,
	})
	g.char(account, sessionID, wantChar{
		name: "Leaving", objectID: 0x10000020,
		race: 4, baseClass: 53, curHP: 80, curMP: 30, level: 2,
		maxHP: 80, maxMP: 30, deleteTimer: 90, classID: 53,
	})

	checkPacket(t, "CharSelectionInfo", NewCharSelectionInfoPacket(account, sessionID, chars, now), g.b)
}

// Nobody has entered the world: no character is active.
func TestCharSelectionInfoNoActive(t *testing.T) {
	chars := []storage.Character{
		{ObjectID: 1, Name: "A", Level: 1},
		{ObjectID: 2, Name: "B", Level: 1},
	}
	var g golden
	g.c(0x13).d(2)
	g.char("acc", 7, wantChar{name: "A", objectID: 1, level: 1})
	g.char("acc", 7, wantChar{name: "B", objectID: 2, level: 1})
	checkPacket(t, "CharSelectionInfo", NewCharSelectionInfoPacket("acc", 7, chars, time.Now()), g.b)
}

func TestActiveIndex(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		last []time.Time
		want int
	}{
		{nil, -1},
		{[]time.Time{{}, {}}, -1},
		{[]time.Time{t0, {}}, 0},
		{[]time.Time{{}, t0}, 1},
		{[]time.Time{t0, t0.Add(time.Second), t0}, 1},
		{[]time.Time{t0, t0}, 0}, // ties go to the first
	}
	for _, tt := range tests {
		chars := make([]storage.Character, len(tt.last))
		for i, l := range tt.last {
			chars[i].LastAccess = l
		}
		if got := activeIndex(chars); got != tt.want {
			t.Errorf("activeIndex(%v) = %d, want %d", tt.last, got, tt.want)
		}
	}
}
