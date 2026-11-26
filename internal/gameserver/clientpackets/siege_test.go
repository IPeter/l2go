package clientpackets

import "testing"

// Mobius RequestSiegeInfo.readImpl reads nothing.
func TestParseRequestSiegeInfo(t *testing.T) {
	for _, body := range m3NoFieldBodies {
		if got, err := ParseRequestSiegeInfo(body); err != nil || got != (RequestSiegeInfo{}) {
			t.Errorf("ParseRequestSiegeInfo(% x) = %+v, %v", body, got, err)
		}
	}
}

// Mobius RequestSiegeAttackerList.readImpl and
// RequestSiegeDefenderList.readImpl: d castle ID.
func TestParseRequestSiegeLists(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want int32
	}{
		{"Gludio", new(m67Body).d(1).b, 1},
		{"Rune", new(m67Body).d(8).b, 8},
		{"negative", []byte{0xff, 0xff, 0xff, 0xff}, -1},
		{"trailing data", new(m67Body).d(5).c(9).b, 5},
	}
	for _, tt := range tests {
		a, err := ParseRequestSiegeAttackerList(tt.body)
		m67Parsed(t, "attackers "+tt.name, a, err, RequestSiegeAttackerList{CastleID: tt.want})
		d, err := ParseRequestSiegeDefenderList(tt.body)
		m67Parsed(t, "defenders "+tt.name, d, err, RequestSiegeDefenderList{CastleID: tt.want})
	}
	m67Truncations(t, "RequestSiegeAttackerList", new(m67Body).d(3).b, ParseRequestSiegeAttackerList)
	m67Truncations(t, "RequestSiegeDefenderList", new(m67Body).d(3).b, ParseRequestSiegeDefenderList)
}

// Mobius RequestJoinSiege.readImpl: d castle ID, d isAttacker, d isJoining.
func TestParseRequestJoinSiege(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestJoinSiege
	}{
		{"join as attacker", new(m67Body).d(2).d(1).d(1).b, RequestJoinSiege{CastleID: 2, IsAttacker: 1, IsJoining: 1}},
		{"join as defender", new(m67Body).d(2).d(0).d(1).b, RequestJoinSiege{CastleID: 2, IsAttacker: 0, IsJoining: 1}},
		{"withdraw", new(m67Body).d(7).d(1).d(0).b, RequestJoinSiege{CastleID: 7, IsAttacker: 1, IsJoining: 0}},
		// Not checked here: Mobius takes anything but 1 as false.
		{"odd values", new(m67Body).d(3).d(5).d(-1).b, RequestJoinSiege{CastleID: 3, IsAttacker: 5, IsJoining: -1}},
		{"trailing data", new(m67Body).d(1).d(1).d(1).c(0).b, RequestJoinSiege{CastleID: 1, IsAttacker: 1, IsJoining: 1}},
	}
	for _, tt := range tests {
		got, err := ParseRequestJoinSiege(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "RequestJoinSiege", new(m67Body).d(2).d(1).d(1).b, ParseRequestJoinSiege)
}

// Mobius RequestConfirmSiegeWaitingList.readImpl: d castle ID, d clan ID,
// d approved.
func TestParseRequestConfirmSiegeWaitingList(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestConfirmSiegeWaitingList
	}{
		{"approve", new(m67Body).d(4).d(0x10000003).d(1).b, RequestConfirmSiegeWaitingList{CastleID: 4, ClanID: 0x10000003, Approved: 1}},
		{"refuse", new(m67Body).d(4).d(0x10000003).d(0).b, RequestConfirmSiegeWaitingList{CastleID: 4, ClanID: 0x10000003, Approved: 0}},
		{"trailing data", new(m67Body).d(1).d(2).d(1).c(7).b, RequestConfirmSiegeWaitingList{CastleID: 1, ClanID: 2, Approved: 1}},
	}
	for _, tt := range tests {
		got, err := ParseRequestConfirmSiegeWaitingList(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "RequestConfirmSiegeWaitingList", new(m67Body).d(4).d(5).d(1).b, ParseRequestConfirmSiegeWaitingList)
}

// The opcodes are those of Mobius ClientPackets (Interlude).
func TestSiegeClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestSiegeInfo", OpRequestSiegeInfo, 0x47},
		{"RequestSiegeAttackerList", OpRequestSiegeAttackerList, 0xa2},
		{"RequestSiegeDefenderList", OpRequestSiegeDefenderList, 0xa3},
		{"RequestJoinSiege", OpRequestJoinSiege, 0xa4},
		{"RequestConfirmSiegeWaitingList", OpRequestConfirmSiegeWaitingList, 0xa5},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

func FuzzParseRequestJoinSiege(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(2).d(1).d(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestJoinSiege(data)
		if m67Fuzzed(t, data, got, err, 12, false, nil) &&
			(got.CastleID != m67Int(data, 0) || got.IsAttacker != m67Int(data, 4) || got.IsJoining != m67Int(data, 8)) {
			t.Errorf("% x: %+v", data, got)
		}
	})
}

func FuzzParseRequestConfirmSiegeWaitingList(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(4).d(5).d(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestConfirmSiegeWaitingList(data)
		if m67Fuzzed(t, data, got, err, 12, false, nil) &&
			(got.CastleID != m67Int(data, 0) || got.ClanID != m67Int(data, 4) || got.Approved != m67Int(data, 8)) {
			t.Errorf("% x: %+v", data, got)
		}
	})
}
