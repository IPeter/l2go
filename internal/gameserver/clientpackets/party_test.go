package clientpackets

import "testing"

// The party, friend and block list packets of M9, built by hand from the
// Mobius readImpl.

// The opcodes are those of Mobius ClientPackets and ExClientPackets, and
// the constants those of the Mobius classes named.
func TestM9PartyFriendClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestJoinParty", OpRequestJoinParty, 0x29},
		{"RequestAnswerJoinParty", OpRequestAnswerJoinParty, 0x2a},
		{"RequestWithDrawalParty", OpRequestWithDrawalParty, 0x2b},
		{"RequestOustPartyMember", OpRequestOustPartyMember, 0x2c},
		{"RequestFriendInvite", OpRequestFriendInvite, 0x5e},
		{"RequestAnswerFriendInvite", OpRequestAnswerFriendInvite, 0x5f},
		{"RequestFriendList", OpRequestFriendList, 0x60},
		{"RequestFriendDel", OpRequestFriendDel, 0x61},
		{"RequestPartyMatchConfig", OpRequestPartyMatchConfig, 0x6f},
		{"RequestBlock", OpRequestBlock, 0xa0},
		{"RequestSendFriendMsg", OpRequestSendFriendMsg, 0xcc},
		{"Ex RequestChangePartyLeader", ExOpRequestChangePartyLeader, 0x04},
		{"RequestAnswerJoinParty -1", PartyAnswerRefusalMode, -1},
		{"RequestAnswerJoinParty 0", PartyAnswerDeclined, 0},
		{"RequestAnswerJoinParty 1", PartyAnswerAccepted, 1},
		{"RequestBlock.BLOCK", BlockAdd, 0},
		{"RequestBlock.UNBLOCK", BlockRemove, 1},
		{"RequestBlock.BLOCKLIST", BlockShowList, 2},
		{"RequestBlock.ALLBLOCK", BlockRefuseAll, 3},
		{"RequestBlock.ALLUNBLOCK", BlockAcceptAll, 4},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius RequestJoinParty.readImpl: S name, d partyDistributionTypeId.
func TestParseRequestJoinParty(t *testing.T) {
	got, err := ParseRequestJoinParty(new(m3Body).s("Pal").d(1).b)
	m3Parsed(t, "random", got, err, RequestJoinParty{Name: "Pal", Loot: 1})
	got, err = ParseRequestJoinParty([]byte{'A', 0, 'l', 0, 0, 0, 4, 0, 0, 0})
	m3Parsed(t, "bytes", got, err, RequestJoinParty{Name: "Al", Loot: 4})
	// Unknown loot modes parse; Mobius maps them in runImpl.
	got, err = ParseRequestJoinParty(new(m3Body).s("Pal").d(-7).b)
	m3Parsed(t, "unknown loot", got, err, RequestJoinParty{Name: "Pal", Loot: -7})
	// Bytes after the fields are ignored, as in Mobius.
	got, err = ParseRequestJoinParty(new(m3Body).s("Pal").d(0).c(9).b)
	m3Parsed(t, "trailing", got, err, RequestJoinParty{Name: "Pal"})

	m3Truncations(t, "RequestJoinParty", new(m3Body).s("Pal").d(1).b, ParseRequestJoinParty)
}

// Mobius RequestAnswerJoinParty.readImpl: d response.
func TestParseRequestAnswerJoinParty(t *testing.T) {
	for _, v := range []int32{PartyAnswerRefusalMode, PartyAnswerDeclined, PartyAnswerAccepted, 5} {
		got, err := ParseRequestAnswerJoinParty(new(m3Body).d(v).b)
		m3Parsed(t, "answer", got, err, RequestAnswerJoinParty{Response: v})
	}
	got, err := ParseRequestAnswerJoinParty([]byte{0xff, 0xff, 0xff, 0xff})
	m3Parsed(t, "bytes", got, err, RequestAnswerJoinParty{Response: -1})
	m3Truncations(t, "RequestAnswerJoinParty", new(m3Body).d(1).b, ParseRequestAnswerJoinParty)
}

// RequestWithDrawalParty and RequestFriendList: Mobius readImpl reads
// nothing.
func TestParsePartyFriendNoFieldPackets(t *testing.T) {
	for _, body := range m3NoFieldBodies {
		if got, err := ParseRequestWithDrawalParty(body); err != nil || got != (RequestWithDrawalParty{}) {
			t.Errorf("ParseRequestWithDrawalParty(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestFriendList(body); err != nil || got != (RequestFriendList{}) {
			t.Errorf("ParseRequestFriendList(% x) = %+v, %v", body, got, err)
		}
	}
}

// Mobius RequestOustPartyMember, RequestChangePartyLeader (after the Ex
// sub-opcode), RequestFriendInvite and RequestFriendDel readImpl: S name.
func TestParseNamePackets(t *testing.T) {
	names := []string{"Pal", "", "Hérőke"}
	for _, name := range names {
		body := new(m3Body).s(name).b
		oust, err := ParseRequestOustPartyMember(body)
		m3Parsed(t, "RequestOustPartyMember "+name, oust, err, RequestOustPartyMember{Name: name})
		leader, err := ParseRequestChangePartyLeader(body)
		m3Parsed(t, "RequestChangePartyLeader "+name, leader, err, RequestChangePartyLeader{Name: name})
		invite, err := ParseRequestFriendInvite(body)
		m3Parsed(t, "RequestFriendInvite "+name, invite, err, RequestFriendInvite{Name: name})
		del, err := ParseRequestFriendDel(body)
		m3Parsed(t, "RequestFriendDel "+name, del, err, RequestFriendDel{Name: name})
	}

	full := new(m3Body).s("Pal").b
	m3Truncations(t, "RequestOustPartyMember", full, ParseRequestOustPartyMember)
	m3Truncations(t, "RequestChangePartyLeader", full, ParseRequestChangePartyLeader)
	m3Truncations(t, "RequestFriendInvite", full, ParseRequestFriendInvite)
	m3Truncations(t, "RequestFriendDel", full, ParseRequestFriendDel)

	// The whole Ex packet: D0 04 00, then the name.
	ex, err := ParseExPacket(append([]byte{0x04, 0x00}, full...))
	if err != nil || ex.SubOpcode != ExOpRequestChangePartyLeader {
		t.Fatalf("ParseExPacket = %+v, %v", ex, err)
	}
	leader, err := ParseRequestChangePartyLeader(ex.Body)
	m3Parsed(t, "RequestChangePartyLeader Ex", leader, err, RequestChangePartyLeader{Name: "Pal"})
}

// Mobius RequestPartyMatchConfig.readImpl: d auto, d loc, d level.
func TestParseRequestPartyMatchConfig(t *testing.T) {
	got, err := ParseRequestPartyMatchConfig(new(m3Body).d(0).d(-1).d(1).b)
	m3Parsed(t, "all", got, err, RequestPartyMatchConfig{Auto: 0, Location: -1, Level: 1})
	got, err = ParseRequestPartyMatchConfig([]byte{1, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0})
	m3Parsed(t, "bytes", got, err, RequestPartyMatchConfig{Auto: 1, Location: 5})
	m3Truncations(t, "RequestPartyMatchConfig", new(m3Body).d(1).d(2).d(3).b, ParseRequestPartyMatchConfig)
}

// Mobius RequestAnswerFriendInvite.readImpl: d response; only 1 accepts.
func TestParseRequestAnswerFriendInvite(t *testing.T) {
	tests := []struct {
		v      int32
		accept bool
	}{{1, true}, {0, false}, {-1, false}, {2, false}}
	for _, tt := range tests {
		got, err := ParseRequestAnswerFriendInvite(new(m3Body).d(tt.v).b)
		m3Parsed(t, "answer", got, err, RequestAnswerFriendInvite{Response: tt.v})
		if got.Accepted() != tt.accept {
			t.Errorf("response %d: Accepted() = %v", tt.v, got.Accepted())
		}
	}
	m3Truncations(t, "RequestAnswerFriendInvite", new(m3Body).d(1).b, ParseRequestAnswerFriendInvite)
}

// Mobius RequestBlock.readImpl: d type, then S name only for BLOCK (0) and
// UNBLOCK (1).
func TestParseRequestBlock(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestBlock
	}{
		{"block", new(m3Body).d(0).s("Pest").b, RequestBlock{Type: BlockAdd, Name: "Pest"}},
		{"unblock", new(m3Body).d(1).s("Pest").b, RequestBlock{Type: BlockRemove, Name: "Pest"}},
		{"list", new(m3Body).d(2).b, RequestBlock{Type: BlockShowList}},
		{"refuse all", new(m3Body).d(3).b, RequestBlock{Type: BlockRefuseAll}},
		{"accept all", new(m3Body).d(4).b, RequestBlock{Type: BlockAcceptAll}},
		// A name after the other types is not read.
		{"list with a name", new(m3Body).d(2).s("Pest").b, RequestBlock{Type: BlockShowList}},
		{"unknown type", new(m3Body).d(9).b, RequestBlock{Type: 9}},
		{"bytes", []byte{0, 0, 0, 0, 'A', 0, 0, 0}, RequestBlock{Type: BlockAdd, Name: "A"}},
	}
	for _, tt := range tests {
		got, err := ParseRequestBlock(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestBlock block", new(m3Body).d(0).s("Pest").b, ParseRequestBlock)
	m3Truncations(t, "RequestBlock list", new(m3Body).d(2).b, ParseRequestBlock)

	// A block without its name fails.
	got, err := ParseRequestBlock(new(m3Body).d(0).b)
	m3Failed(t, "block without a name", got, err)
}

// Mobius RequestSendFriendMsg.readImpl: S message, S receiver.
func TestParseRequestSendFriendMsg(t *testing.T) {
	got, err := ParseRequestSendFriendMsg(new(m3Body).s("hello there").s("Pal").b)
	m3Parsed(t, "message", got, err, RequestSendFriendMsg{Message: "hello there", Receiver: "Pal"})
	got, err = ParseRequestSendFriendMsg(new(m3Body).s("").s("").b)
	m3Parsed(t, "empty", got, err, RequestSendFriendMsg{})
	got, err = ParseRequestSendFriendMsg([]byte{'h', 0, 0, 0, 'P', 0, 0, 0})
	m3Parsed(t, "bytes", got, err, RequestSendFriendMsg{Message: "h", Receiver: "P"})
	m3Truncations(t, "RequestSendFriendMsg", new(m3Body).s("hi").s("Pal").b, ParseRequestSendFriendMsg)

	// A message without the receiver fails.
	got, err = ParseRequestSendFriendMsg(new(m3Body).s("hi").b)
	m3Failed(t, "no receiver", got, err)
}

func FuzzParseRequestJoinParty(f *testing.F) {
	m3Fuzz(f, ParseRequestJoinParty, -1, new(m3Body).s("Pal").d(1).b, new(m3Body).s("").d(0).b)
}

func FuzzParseRequestAnswerJoinParty(f *testing.F) {
	m3Fuzz(f, ParseRequestAnswerJoinParty, 4, new(m3Body).d(1).b)
}

func FuzzParseRequestWithDrawalParty(f *testing.F) {
	m3Fuzz(f, ParseRequestWithDrawalParty, 0, m3NoFieldBodies...)
}

func FuzzParseRequestOustPartyMember(f *testing.F) {
	m3Fuzz(f, ParseRequestOustPartyMember, -1, new(m3Body).s("Pal").b)
}

func FuzzParseRequestChangePartyLeader(f *testing.F) {
	m3Fuzz(f, ParseRequestChangePartyLeader, -1, new(m3Body).s("Pal").b)
}

func FuzzParseRequestPartyMatchConfig(f *testing.F) {
	m3Fuzz(f, ParseRequestPartyMatchConfig, 12, new(m3Body).d(0).d(-1).d(1).b)
}

func FuzzParseRequestFriendInvite(f *testing.F) {
	m3Fuzz(f, ParseRequestFriendInvite, -1, new(m3Body).s("Pal").b)
}

func FuzzParseRequestAnswerFriendInvite(f *testing.F) {
	m3Fuzz(f, ParseRequestAnswerFriendInvite, 4, new(m3Body).d(1).b)
}

func FuzzParseRequestFriendList(f *testing.F) {
	m3Fuzz(f, ParseRequestFriendList, 0, m3NoFieldBodies...)
}

func FuzzParseRequestFriendDel(f *testing.F) {
	m3Fuzz(f, ParseRequestFriendDel, -1, new(m3Body).s("Pal").b)
}

// FuzzParseRequestBlock also checks that only BLOCK and UNBLOCK read a
// name: the other types parse from their four bytes alone.
func FuzzParseRequestBlock(f *testing.F) {
	f.Add(new(m3Body).d(0).s("Pest").b)
	f.Add(new(m3Body).d(1).s("").b)
	f.Add(new(m3Body).d(3).b)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestBlock(data)
		if err != nil {
			m3Failed(t, "fuzzed input", got, err)
			if len(data) >= 4 {
				if typ := m67Int(data, 0); typ != BlockAdd && typ != BlockRemove {
					t.Errorf("% x: type %d failed: %v", data, typ, err)
				}
			}
			return
		}
		if got.Type != m67Int(data, 0) {
			t.Errorf("% x: type %d", data, got.Type)
		}
		if got.Type != BlockAdd && got.Type != BlockRemove && got.Name != "" {
			t.Errorf("% x: type %d read the name %q", data, got.Type, got.Name)
		}
	})
}

func FuzzParseRequestSendFriendMsg(f *testing.F) {
	m3Fuzz(f, ParseRequestSendFriendMsg, -1, new(m3Body).s("hello").s("Pal").b, new(m3Body).s("").s("").b)
}
