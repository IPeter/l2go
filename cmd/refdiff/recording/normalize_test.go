package recording

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/cmd/refdiff/scenario"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/testclient"
)

// alice and bob are the identities of the hand-made captures.
var (
	alice = scenario.Identity{Role: "alice", Account: "rdkx7", Char: "Rdqz9"}
	bob   = scenario.Identity{Role: "bob", Account: "rdbbb", Char: "Rdzzz"}
)

func srvPacket(name string, v any) scenario.Packet {
	return scenario.Packet{Dir: testclient.FromServer, Name: name, Value: v}
}

func cliPacket(name string) scenario.Packet {
	return scenario.Packet{Dir: testclient.FromClient, Name: name}
}

// newCapture returns a capture of server 1 with the given streams.
func newCapture(players []scenario.Identity, streams ...scenario.Stream) *scenario.Capture {
	return &scenario.Capture{
		Scenario: "test",
		Target:   scenario.Target{Label: "l2go", LoginAddr: "127.0.0.1:2106", ServerID: 1},
		Players:  players,
		Streams:  streams,
	}
}

// field returns the value of path in e.
func field(t *testing.T, e Entry, path string) string {
	t.Helper()
	for _, f := range e.Fields {
		if f.Path == path {
			return f.Value
		}
	}
	t.Fatalf("%s has no field %q; fields: %v", e.Name, path, e.Fields)
	return ""
}

// checkFields compares fields of e with want (path to value).
func checkFields(t *testing.T, e Entry, want map[string]string) {
	t.Helper()
	for path, v := range want {
		if got := field(t, e, path); got != v {
			t.Errorf("%s.%s = %s, want %s", e.Name, path, got, v)
		}
	}
}

// le returns v as four little-endian bytes.
func le(v int32) []byte {
	return binary.LittleEndian.AppendUint32(nil, uint32(v)) //nolint:gosec // test bits
}

func TestNormalizeObjectIDs(t *testing.T) {
	const (
		player = int32(0x10000001)
		sword  = int32(0x10000002)
		shirt  = int32(0x10000003)
		dagger = int32(0x10000004)
		other  = int32(0x10000005)
		npc1   = int32(0x60000001)
		npc2   = int32(0x60000002)
	)
	var equip [testclient.NumEquipSlots]int32
	equip[testclient.SlotRHand] = sword
	equip[testclient.SlotChest] = shirt
	c := newCapture([]scenario.Identity{alice, bob},
		scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			srvPacket("CharSelectionInfo", testclient.CharacterList{Characters: []testclient.CharInfo{
				{Name: alice.Char, ObjectID: player, EquipObjectIDs: equip},
			}}),
			srvPacket("UserInfo", testclient.UserInfo{ObjectID: player, PaperdollObjectIDs: equip}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npc1, NpcTypeID: 1030001}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npc2, NpcTypeID: 1030002}),
			srvPacket("MyTargetSelected", testclient.MyTargetSelected{ObjectID: npc2}),
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{
				{ObjectID: sword, ItemID: 2369}, {ObjectID: dagger, ItemID: 10},
			}}),
		}},
		scenario.Stream{Name: "game bob", Packets: []scenario.Packet{
			srvPacket("CharInfo", testclient.CharInfoPacket{ObjectID: player, Name: alice.Char}),
			srvPacket("CharSelected", testclient.CharSelected{ObjectID: other, Name: bob.Char}),
		}},
	)
	rec := Normalize(c)
	a, b := rec.Streams[0].Entries, rec.Streams[1].Entries
	checkFields(t, a[0], map[string]string{
		"Characters[0].ObjectID":           "#player1",
		"Characters[0].EquipObjectIDs[7]":  "#item1",
		"Characters[0].EquipObjectIDs[10]": "#item2",
		"Characters[0].EquipObjectIDs[0]":  "0",
	})
	checkFields(t, a[1], map[string]string{"ObjectID": "#player1", "PaperdollObjectIDs[7]": "#item1"})
	checkFields(t, a[2], map[string]string{"ObjectID": "#npc1", "NpcTypeID": "1030001"})
	checkFields(t, a[3], map[string]string{"ObjectID": "#npc2", "NpcTypeID": "1030002"})
	checkFields(t, a[4], map[string]string{"ObjectID": "#npc2"})
	checkFields(t, a[5], map[string]string{"Items[0].ObjectID": "#item1", "Items[1].ObjectID": "#item3", "Items.len": "2"})
	// The same object keeps its name in another stream.
	checkFields(t, b[0], map[string]string{"ObjectID": "#player1"})
	checkFields(t, b[1], map[string]string{"ObjectID": "#player2", "X": "start"})
}

// twoNpcCapture builds the same session with the given raw IDs: the
// player, two NPCs and an item.
func twoNpcCapture(player, npcA, npcB, item int32) *scenario.Capture {
	return newCapture([]scenario.Identity{alice},
		scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			cliPacket("EnterWorld"),
			srvPacket("UserInfo", testclient.UserInfo{ObjectID: player, Name: alice.Char}),
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{{ObjectID: item, ItemID: 57, Count: 100}}}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npcA, NpcTypeID: 1030001}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npcB, NpcTypeID: 1030002}),
			cliPacket("Action"),
			srvPacket("MyTargetSelected", testclient.MyTargetSelected{ObjectID: npcB}),
			srvPacket("NpcHtmlMessage", testclient.NpcHtmlMessage{
				ObjectID: npcB,
				HTML:     `<a action="bypass -h npc_` + strconv.Itoa(int(npcB)) + `_Chat 1">Chat</a>`,
			}),
		}})
}

func TestNormalizeDoesNotDependOnRawIDs(t *testing.T) {
	a := Normalize(twoNpcCapture(0x10000001, 0x10000100, 0x10000101, 0x10000002))
	b := Normalize(twoNpcCapture(0x10004000, 0x60000007, 0x60000003, 0x10004001))
	if !reflect.DeepEqual(a.Streams, b.Streams) {
		t.Errorf("normalized streams differ:\n%+v\n%+v", a.Streams, b.Streams)
	}
	html := field(t, a.Streams[0].Entries[7], "HTML")
	if want := strconv.Quote(`<a action="bypass -h npc_#npc2_Chat 1">Chat</a>`); html != want {
		t.Errorf("HTML = %s, want %s", html, want)
	}
}

// visibilityCapture shows two NPCs and another player in the given order.
func visibilityCapture(npcFirst bool, npcA, npcB, other int32) *scenario.Capture {
	npcs := []scenario.Packet{
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npcA, NpcTypeID: 1030001, X: 100}),
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npcB, NpcTypeID: 1030002, X: 200}),
	}
	player := []scenario.Packet{
		srvPacket("CharInfo", testclient.CharInfoPacket{ObjectID: other, Name: bob.Char}),
		srvPacket("RelationChanged", testclient.RelationChanged{ObjectID: other, Relation: 1}),
	}
	var burst []scenario.Packet
	if npcFirst {
		burst = append(append(burst, npcs...), player...)
	} else {
		burst = append(append(burst, player...), npcs[1], npcs[0])
	}
	packets := append([]scenario.Packet{cliPacket("EnterWorld")}, burst...)
	packets = append(packets, srvPacket("MyTargetSelected", testclient.MyTargetSelected{ObjectID: npcB}))
	return newCapture([]scenario.Identity{alice, bob}, scenario.Stream{Name: "game alice", Packets: packets})
}

func TestNormalizeSortsVisibilityBursts(t *testing.T) {
	a := Normalize(visibilityCapture(true, 0x60000001, 0x60000002, 0x10000009))
	b := Normalize(visibilityCapture(false, 0x10000302, 0x10000301, 0x10000077))
	if !reflect.DeepEqual(a.Streams, b.Streams) {
		t.Fatalf("normalized streams differ:\n%+v\n%+v", a.Streams, b.Streams)
	}
	var names []string
	for _, e := range a.Streams[0].Entries {
		names = append(names, e.Name)
	}
	want := []string{"EnterWorld", "CharInfo", "RelationChanged", "NpcInfo", "NpcInfo", "MyTargetSelected"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("order %v, want %v", names, want)
	}
	es := a.Streams[0].Entries
	checkFields(t, es[1], map[string]string{"ObjectID": "#player1", "Name": strconv.Quote("<char:bob>")})
	checkFields(t, es[2], map[string]string{"ObjectID": "#player1"})
	checkFields(t, es[3], map[string]string{"ObjectID": "#npc1", "NpcTypeID": "1030001"})
	checkFields(t, es[5], map[string]string{"ObjectID": "#npc2"})
}

func TestNormalizeSortsUnparsedBurstsWithoutRawIDs(t *testing.T) {
	spawn := func(id, itemID int32) scenario.Packet {
		return srvPacket("SpawnItem", testclient.Unknown{Opcode: 0x0b, Body: append(le(id), le(itemID)...)})
	}
	a := Normalize(newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		spawn(0x10000050, 57), spawn(0x10000051, 1060),
	}}))
	b := Normalize(newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		spawn(0x10000999, 1060), spawn(0x10000111, 57),
	}}))
	for _, rec := range []Recording{a, b} {
		es := rec.Streams[0].Entries
		// Item 1060 (24 04 00 00) sorts before item 57 (39 00 00 00).
		if !strings.HasSuffix(es[0].Raw, "24040000") || !strings.HasSuffix(es[1].Raw, "39000000") {
			t.Errorf("order: %s, %s; want item 1060 first", es[0].Raw, es[1].Raw)
		}
	}
}

func TestNormalizePlaceholders(t *testing.T) {
	const (
		sid    = int32(0x1234)
		player = int32(0x10000001)
		npc    = int32(0x60000010)
	)
	c := newCapture([]scenario.Identity{alice},
		scenario.Stream{Name: "login alice", Packets: []scenario.Packet{
			srvPacket("Init", testclient.Init{SessionID: sid, Protocol: testclient.LoginProtocol, GameGuard: testclient.MobiusGameGuard}),
			cliPacket("AuthGameGuard"),
			srvPacket("GGAuth", testclient.GGAuth{Response: sid}),
			cliPacket("RequestAuthLogin"),
			srvPacket("LoginOk", testclient.LoginOk{LoginOK1: 111, LoginOK2: 222}),
			cliPacket("RequestServerList"),
			srvPacket("ServerList", testclient.ServerListResult{LastServer: 1, Servers: []testclient.ServerInfo{
				{ID: 1, IP: netip.MustParseAddr("192.168.1.10"), Port: 7777, CurrentPlayers: 3, MaxPlayers: 100, Up: true},
				{ID: 9, IP: netip.MustParseAddr("192.168.1.11"), Port: 7778, MaxPlayers: 100},
			}}),
			cliPacket("RequestServerLogin"),
			srvPacket("PlayOk", testclient.PlayOk{PlayOK1: 333, PlayOK2: 444}),
		}},
		scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			cliPacket("ProtocolVersion"),
			srvPacket("KeyPacket", testclient.KeyPacket{Result: 1, KeyHalf: [8]byte{1, 2, 3, 4, 5, 6, 7, 8}, Encryption: true, ServerID: 1, Unknown: 1}),
			cliPacket("AuthLogin"),
			srvPacket("CharSelectionInfo", testclient.CharacterList{Characters: []testclient.CharInfo{
				{Name: alice.Char, ObjectID: player, Account: alice.Account, SessionID: 333},
			}}),
			srvPacket("CharSelected", testclient.CharSelected{Name: alice.Char, ObjectID: player, SessionID: 333, GameTime: 754}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npc, NpcTypeID: 1030001, Name: "Guard"}),
			srvPacket("NpcHtmlMessage", testclient.NpcHtmlMessage{
				ObjectID: npc,
				HTML:     `Hello Rdqz9! <a action="bypass -h npc_1610612752_Chat 1">Talk</a> (1000 adena)`,
			}),
			srvPacket("CreatureSay", testclient.CreatureSay{ObjectID: player, Type: testclient.ChatWhisper, Name: "->" + alice.Char, Text: "hi"}),
			srvPacket("ClientSetTime", testclient.Unknown{Opcode: 0xec, Body: []byte{0x58, 0x02, 0, 0, 6, 0, 0, 0}}),
		}},
	)
	c.Keys = []session.Keys{{LoginOK1: 111, LoginOK2: 222, PlayOK1: 333, PlayOK2: 444}}
	c.SessionIDs = []int32{sid}
	rec := Normalize(c)
	login, game := rec.Streams[0].Entries, rec.Streams[1].Entries

	checkFields(t, login[0], map[string]string{
		"SessionID":        "<session-id>",
		"ScrambledModulus": "<rsa-modulus>",
		"BlowfishKey":      "<blowfish-key>",
		"Protocol":         strconv.Itoa(int(testclient.LoginProtocol)),
	})
	if login[1].Dir != DirClient || login[1].Name != "AuthGameGuard" || login[1].Fields != nil {
		t.Errorf("client entry = %+v, want AuthGameGuard without fields", login[1])
	}
	checkFields(t, login[2], map[string]string{"Response": "<session-id>"})
	checkFields(t, login[4], map[string]string{"LoginOK1": "<login-ok1>", "LoginOK2": "<login-ok2>"})
	checkFields(t, login[6], map[string]string{
		"LastServer":                "1",
		"Servers[0].ID":             "<server-id>",
		"Servers[0].IP":             "<ip>",
		"Servers[0].Port":           "<port>",
		"Servers[0].CurrentPlayers": "<players>",
		"Servers[0].MaxPlayers":     "100",
		"Servers[1].ID":             "9",
	})
	checkFields(t, login[8], map[string]string{"PlayOK1": "<play-ok1>", "PlayOK2": "<play-ok2>"})

	checkFields(t, game[1], map[string]string{"KeyHalf": "<xor-key>", "ServerID": "<server-id>", "Encryption": "true"})
	checkFields(t, game[3], map[string]string{
		"Characters[0].Name":      strconv.Quote("<char:alice>"),
		"Characters[0].Account":   strconv.Quote("<account:alice>"),
		"Characters[0].SessionID": "<play-ok1>",
	})
	checkFields(t, game[4], map[string]string{
		"Name":      strconv.Quote("<char:alice>"),
		"SessionID": "<play-ok1>",
		"GameTime":  "<game-time>",
	})
	checkFields(t, game[6], map[string]string{
		"ObjectID": "#npc1",
		"HTML":     strconv.Quote(`Hello <char:alice>! <a action="bypass -h npc_#npc1_Chat 1">Talk</a> (1000 adena)`),
	})
	checkFields(t, game[7], map[string]string{"Name": strconv.Quote("-><char:alice>"), "Text": strconv.Quote("hi")})
	if got, want := game[8].Raw, "{game-time}06000000"; got != want {
		t.Errorf("ClientSetTime raw = %s, want %s", got, want)
	}
}

func TestNormalizeServerIDOfAnotherServer(t *testing.T) {
	c := newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("KeyPacket", testclient.KeyPacket{Result: 1, ServerID: 2}),
	}})
	if got := field(t, Normalize(c).Streams[0].Entries[0], "ServerID"); got != "2" {
		t.Errorf("ServerID of another server = %s, want 2 (only the target's ID is a placeholder)", got)
	}
}

func TestNormalizeRelativeCoordinates(t *testing.T) {
	const (
		player = int32(0x10000001)
		npc    = int32(0x60000001)
	)
	c := newCapture([]scenario.Identity{alice}, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("CharSelected", testclient.CharSelected{ObjectID: player, X: -71338, Y: 258271, Z: -3104}),
		srvPacket("UserInfo", testclient.UserInfo{ObjectID: player, X: -71338, Y: 258271, Z: -3104}),
		srvPacket("MoveToLocation", testclient.MoveToLocation{
			ObjectID: player, DestX: -70838, DestY: 258271, DestZ: -3104, X: -71338, Y: 258271, Z: -3104,
		}),
		srvPacket("ValidateLocation", testclient.ValidateLocation{ObjectID: player, X: -71350, Y: 258280, Z: -3100}),
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npc, X: -71000, Y: 258000, Z: -3104}),
		srvPacket("MoveToPawn", testclient.MoveToPawn{
			ObjectID: player, TargetID: npc, X: -71338, Y: 258271, Z: -3104, TargetX: -71000, TargetY: 258000, TargetZ: -3104,
		}),
	}})
	rec := Normalize(c)
	es := rec.Streams[0].Entries
	checkFields(t, es[0], map[string]string{"X": "start", "Y": "start", "Z": "start"})
	checkFields(t, es[1], map[string]string{"X": "start", "Y": "start", "Z": "start"})
	checkFields(t, es[2], map[string]string{"DestX": "start+500", "DestY": "start", "X": "start"})
	checkFields(t, es[3], map[string]string{"X": "start-12", "Y": "start+9", "Z": "start+4"})
	checkFields(t, es[4], map[string]string{"X": "-71000", "Y": "258000"})
	checkFields(t, es[5], map[string]string{"X": "start", "TargetX": "-71000", "TargetID": "#npc1"})
	want := []Anchor{{Object: "#player1", X: -71338, Y: 258271, Z: -3104}}
	if !reflect.DeepEqual(rec.Anchors, want) {
		t.Errorf("anchors = %+v, want %+v", rec.Anchors, want)
	}
}

func TestNormalizeRawPackets(t *testing.T) {
	const player = int32(0x10000001)
	c := newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("UserInfo", testclient.UserInfo{ObjectID: player}),
		srvPacket("MagicSkillUse", testclient.Unknown{Opcode: 0x48, Body: append(le(player), 0x05)}),
		{
			Dir: testclient.FromServer, Name: "UserInfo", Opcode: testclient.OpUserInfo,
			Raw: []byte{testclient.OpUserInfo, 0x01, 0x02},
			Err: errors.New("testclient: malformed packet: UserInfo: too short"),
		},
	}})
	es := Normalize(c).Streams[0].Entries
	if got, want := es[1].Raw, "{#player1}05"; got != want {
		t.Errorf("unparsed packet raw = %s, want %s", got, want)
	}
	if !es[2].Malformed || es[2].Raw != "0102" || !strings.Contains(es[2].Error, "too short") || es[2].Fields != nil {
		t.Errorf("malformed entry = %+v, want malformed, raw 0102 and the error", es[2])
	}
}

func TestNormalizeShortcutItems(t *testing.T) {
	const item = int32(0x10000002)
	c := newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{{ObjectID: item, ItemID: 10}}}),
		srvPacket("ShortcutInit", testclient.ShortcutInit{Shortcuts: []testclient.Shortcut{
			{Type: testclient.ShortcutItem, Slot: 0, ID: item, Level: 1},
			{Type: testclient.ShortcutAction, Slot: 1, ID: 2, Level: 1},
		}}),
	}})
	checkFields(t, Normalize(c).Streams[0].Entries[1], map[string]string{
		"Shortcuts[0].ID": "#item1",
		"Shortcuts[1].ID": "2",
	})
}

func TestNormalizeErrorsAndNotes(t *testing.T) {
	const npc = int32(0x60000010)
	c := newCapture([]scenario.Identity{alice}, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: npc}),
	}})
	c.Err = errors.New(`step "alice selects the character": character Rdqz9 is not in the list`)
	c.Notes = []string{"first link: bypass npc_1610612752_Chat 1"}
	rec := Normalize(c)
	if len(rec.Errors) != 1 || !strings.Contains(rec.Errors[0], "character <char:alice> is not") {
		t.Errorf("errors = %q, want the error with the role's name", rec.Errors)
	}
	if want := []string{"first link: bypass npc_#npc1_Chat 1"}; !reflect.DeepEqual(rec.Notes, want) {
		t.Errorf("notes = %q, want %q", rec.Notes, want)
	}
	if rec.Server != "l2go" || rec.Scenario != "test" || rec.Version != Version {
		t.Errorf("header = %q %q %d", rec.Server, rec.Scenario, rec.Version)
	}
}

func TestGenericPathAndFieldName(t *testing.T) {
	tests := []struct{ path, generic, field string }{
		{"X", "X", "X"},
		{"Servers[0].IP", "Servers[].IP", "IP"},
		{"Characters[12].EquipObjectIDs[7]", "Characters[].EquipObjectIDs[]", "EquipObjectIDs"},
		{"Items.len", "Items.len", "len"},
	}
	for _, tt := range tests {
		if got := genericPath(tt.path); got != tt.generic {
			t.Errorf("genericPath(%q) = %q, want %q", tt.path, got, tt.generic)
		}
		if got := fieldName(tt.path); got != tt.field {
			t.Errorf("fieldName(%q) = %q, want %q", tt.path, got, tt.field)
		}
	}
}

func TestFlatten(t *testing.T) {
	type inner struct{ A, B int32 }
	// Embedded is exported, as testclient.Item is in ItemChange: the
	// fields of an embedded struct join the parent's path.
	type Embedded struct{ E uint16 }
	type packet struct {
		Embedded
		Name  string
		Bytes [3]byte
		Slice []inner
		Ptr   *inner
		Flag  bool
		F     float64
		Addr  netip.Addr
		Enum  testclient.CreateFailReason
		ArrN  [2]int32
		hide  int //nolint:unused // unexported fields are skipped
	}
	got := flatten(packet{
		Embedded: Embedded{E: 7},
		Name:     "x",
		Bytes:    [3]byte{0xab, 0, 1},
		Slice:    []inner{{1, 2}},
		F:        0.5,
		Addr:     netip.MustParseAddr("10.0.0.1"),
		Enum:     testclient.CreateFailNameExists,
		ArrN:     [2]int32{3, 4},
	})
	var paths, vals []string
	for _, l := range got {
		paths = append(paths, l.path)
		vals = append(vals, l.val)
	}
	wantPaths := []string{"E", "Name", "Bytes", "Slice.len", "Slice[0].A", "Slice[0].B", "Ptr", "Flag", "F", "Addr", "Enum", "ArrN[0]", "ArrN[1]"}
	wantVals := []string{"7", "x", "ab0001", "1", "1", "2", "nil", "false", "0.5", "10.0.0.1", testclient.CreateFailNameExists.String(), "3", "4"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Errorf("paths = %q, want %q", paths, wantPaths)
	}
	if !reflect.DeepEqual(vals, wantVals) {
		t.Errorf("values = %q, want %q", vals, wantVals)
	}
	if !got[1].text || got[0].text {
		t.Error("only string fields are text")
	}
	if l := flatten(testclient.ActionFailed{}); len(l) != 0 {
		t.Errorf("a packet without fields flattens to %v", l)
	}
}
