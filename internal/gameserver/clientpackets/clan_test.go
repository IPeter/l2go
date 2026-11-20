package clientpackets

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The parser tests of the clan, clan war, crest and alliance packets. Each
// body is built by hand from the Mobius readImpl quoted above it.

// The opcodes are those of Mobius ClientPackets and ExClientPackets.
func TestClanClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestJoinPledge", OpRequestJoinPledge, 0x24},
		{"RequestAnswerJoinPledge", OpRequestAnswerJoinPledge, 0x25},
		{"RequestWithdrawalPledge", OpRequestWithdrawalPledge, 0x26},
		{"RequestOustPledgeMember", OpRequestOustPledgeMember, 0x27},
		{"RequestPledgeMemberList", OpRequestPledgeMemberList, 0x3c},
		{"RequestStartPledgeWar", OpRequestStartPledgeWar, 0x4d},
		{"RequestReplyStartPledgeWar", OpRequestReplyStartPledgeWar, 0x4e},
		{"RequestStopPledgeWar", OpRequestStopPledgeWar, 0x4f},
		{"RequestReplyStopPledgeWar", OpRequestReplyStopPledgeWar, 0x50},
		{"RequestSurrenderPledgeWar", OpRequestSurrenderPledgeWar, 0x51},
		{"RequestReplySurrenderPledgeWar", OpRequestReplySurrenderPledgeWar, 0x52},
		{"RequestSetPledgeCrest", OpRequestSetPledgeCrest, 0x53},
		{"RequestGiveNickName", OpRequestGiveNickName, 0x55},
		{"RequestPledgeInfo", OpRequestPledgeInfo, 0x66},
		{"RequestPledgeExtendedInfo", OpRequestPledgeExtendedInfo, 0x67},
		{"RequestPledgeCrest", OpRequestPledgeCrest, 0x68},
		{"RequestJoinAlly", OpRequestJoinAlly, 0x82},
		{"RequestAnswerJoinAlly", OpRequestAnswerJoinAlly, 0x83},
		{"AllyLeave", OpAllyLeave, 0x84},
		{"AllyDismiss", OpAllyDismiss, 0x85},
		{"RequestDismissAlly", OpRequestDismissAlly, 0x86},
		{"RequestSetAllyCrest", OpRequestSetAllyCrest, 0x87},
		{"RequestAllyCrest", OpRequestAllyCrest, 0x88},
		{"RequestAllyInfo", OpRequestAllyInfo, 0x8e},
		{"RequestPledgePower", OpRequestPledgePower, 0xc0},
		{"Ex RequestExPledgeCrestLarge", ExOpRequestExPledgeCrestLarge, 0x10},
		{"Ex RequestExSetPledgeCrestLarge", ExOpRequestExSetPledgeCrestLarge, 0x11},
		{"Ex RequestPledgeSetAcademyMaster", ExOpRequestPledgeSetAcademyMaster, 0x19},
		{"Ex RequestPledgePowerGradeList", ExOpRequestPledgePowerGradeList, 0x1a},
		{"Ex RequestPledgeMemberPowerInfo", ExOpRequestPledgeMemberPowerInfo, 0x1b},
		{"Ex RequestPledgeSetMemberPowerGrade", ExOpRequestPledgeSetMemberPowerGrade, 0x1c},
		{"Ex RequestPledgeMemberInfo", ExOpRequestPledgeMemberInfo, 0x1d},
		{"Ex RequestPledgeWarList", ExOpRequestPledgeWarList, 0x1e},
		{"Ex RequestPledgeReorganizeMember", ExOpRequestPledgeReorganizeMember, 0x24},
		{"RequestPledgePower show", PledgePowerShow, 1},
		{"RequestPledgePower set", PledgePowerSet, 2},
		{"MaxPledgeCrestSize", MaxPledgeCrestSize, 256},
		{"MaxPledgeCrestLargeSize", MaxPledgeCrestLargeSize, 2176},
		{"MaxAllyCrestSize", MaxAllyCrestSize, 192},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// The packets without fields read nothing, so any body parses.
func TestParseClanNoFieldPackets(t *testing.T) {
	for _, body := range m3NoFieldBodies {
		if got, err := ParseRequestWithdrawalPledge(body); err != nil || got != (RequestWithdrawalPledge{}) {
			t.Errorf("ParseRequestWithdrawalPledge(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestPledgeMemberList(body); err != nil || got != (RequestPledgeMemberList{}) {
			t.Errorf("ParseRequestPledgeMemberList(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestPledgePowerGradeList(body); err != nil || got != (RequestPledgePowerGradeList{}) {
			t.Errorf("ParseRequestPledgePowerGradeList(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseAllyLeave(body); err != nil || got != (AllyLeave{}) {
			t.Errorf("ParseAllyLeave(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestDismissAlly(body); err != nil || got != (RequestDismissAlly{}) {
			t.Errorf("ParseRequestDismissAlly(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestAllyInfo(body); err != nil || got != (RequestAllyInfo{}) {
			t.Errorf("ParseRequestAllyInfo(% x) = %+v, %v", body, got, err)
		}
	}
}

// The packets of one d: RequestAnswerJoinPledge (_answer),
// RequestPledgeInfo (_clanId), RequestPledgeCrest and RequestAllyCrest
// and RequestExPledgeCrestLarge (_crestId), RequestJoinAlly (_id),
// RequestAnswerJoinAlly (_response).
func TestParseClanIntPackets(t *testing.T) {
	var b m3Body
	b.d(0x10000100)
	full := b.b

	got1, err := ParseRequestAnswerJoinPledge(full)
	m3Parsed(t, "RequestAnswerJoinPledge", got1, err, RequestAnswerJoinPledge{Answer: 0x10000100})
	m3Truncations(t, "RequestAnswerJoinPledge", full, ParseRequestAnswerJoinPledge)

	got2, err := ParseRequestPledgeInfo(full)
	m3Parsed(t, "RequestPledgeInfo", got2, err, RequestPledgeInfo{ClanID: 0x10000100})
	m3Truncations(t, "RequestPledgeInfo", full, ParseRequestPledgeInfo)

	got3, err := ParseRequestPledgeCrest(full)
	m3Parsed(t, "RequestPledgeCrest", got3, err, RequestPledgeCrest{CrestID: 0x10000100})
	m3Truncations(t, "RequestPledgeCrest", full, ParseRequestPledgeCrest)

	got4, err := ParseRequestAllyCrest(full)
	m3Parsed(t, "RequestAllyCrest", got4, err, RequestAllyCrest{CrestID: 0x10000100})
	m3Truncations(t, "RequestAllyCrest", full, ParseRequestAllyCrest)

	got5, err := ParseRequestExPledgeCrestLarge(full)
	m3Parsed(t, "RequestExPledgeCrestLarge", got5, err, RequestExPledgeCrestLarge{CrestID: 0x10000100})
	m3Truncations(t, "RequestExPledgeCrestLarge", full, ParseRequestExPledgeCrestLarge)

	got6, err := ParseRequestJoinAlly(full)
	m3Parsed(t, "RequestJoinAlly", got6, err, RequestJoinAlly{TargetID: 0x10000100})
	m3Truncations(t, "RequestJoinAlly", full, ParseRequestJoinAlly)

	got7, err := ParseRequestAnswerJoinAlly(full)
	m3Parsed(t, "RequestAnswerJoinAlly", got7, err, RequestAnswerJoinAlly{Answer: 0x10000100})
	m3Truncations(t, "RequestAnswerJoinAlly", full, ParseRequestAnswerJoinAlly)

	// Mobius ignores the bytes after the fields.
	extra := append(bytes.Clone(full), 0xff, 0xff)
	got8, err := ParseRequestPledgeInfo(extra)
	m3Parsed(t, "RequestPledgeInfo with extra bytes", got8, err, RequestPledgeInfo{ClanID: 0x10000100})
}

// The packets of one S: RequestOustPledgeMember (_target),
// RequestStartPledgeWar, RequestStopPledgeWar, RequestSurrenderPledgeWar
// (_pledgeName), RequestPledgeExtendedInfo (an ignored name), AllyDismiss
// (_clanName).
func TestParseClanStringPackets(t *testing.T) {
	var b m3Body
	b.s("Saxons")
	full := b.b

	got1, err := ParseRequestOustPledgeMember(full)
	m3Parsed(t, "RequestOustPledgeMember", got1, err, RequestOustPledgeMember{Name: "Saxons"})
	m3Truncations(t, "RequestOustPledgeMember", full, ParseRequestOustPledgeMember)

	got2, err := ParseRequestStartPledgeWar(full)
	m3Parsed(t, "RequestStartPledgeWar", got2, err, RequestStartPledgeWar{ClanName: "Saxons"})
	m3Truncations(t, "RequestStartPledgeWar", full, ParseRequestStartPledgeWar)

	got3, err := ParseRequestStopPledgeWar(full)
	m3Parsed(t, "RequestStopPledgeWar", got3, err, RequestStopPledgeWar{ClanName: "Saxons"})
	m3Truncations(t, "RequestStopPledgeWar", full, ParseRequestStopPledgeWar)

	got4, err := ParseRequestSurrenderPledgeWar(full)
	m3Parsed(t, "RequestSurrenderPledgeWar", got4, err, RequestSurrenderPledgeWar{ClanName: "Saxons"})
	m3Truncations(t, "RequestSurrenderPledgeWar", full, ParseRequestSurrenderPledgeWar)

	got5, err := ParseRequestPledgeExtendedInfo(full)
	m3Parsed(t, "RequestPledgeExtendedInfo", got5, err, RequestPledgeExtendedInfo{Name: "Saxons"})
	m3Truncations(t, "RequestPledgeExtendedInfo", full, ParseRequestPledgeExtendedInfo)

	got6, err := ParseAllyDismiss(full)
	m3Parsed(t, "AllyDismiss", got6, err, AllyDismiss{ClanName: "Saxons"})
	m3Truncations(t, "AllyDismiss", full, ParseAllyDismiss)

	var empty m3Body
	empty.s("")
	got7, err := ParseAllyDismiss(empty.b)
	m3Parsed(t, "AllyDismiss empty name", got7, err, AllyDismiss{})
}

// The war replies: S name, d answer (RequestReplyStartPledgeWar,
// RequestReplyStopPledgeWar, RequestReplySurrenderPledgeWar).
func TestParsePledgeWarReplies(t *testing.T) {
	var b m3Body
	b.s("Arthur").d(1)
	full := b.b

	got1, err := ParseRequestReplyStartPledgeWar(full)
	m3Parsed(t, "RequestReplyStartPledgeWar", got1, err, RequestReplyStartPledgeWar{Name: "Arthur", Answer: 1})
	m3Truncations(t, "RequestReplyStartPledgeWar", full, ParseRequestReplyStartPledgeWar)

	got2, err := ParseRequestReplyStopPledgeWar(full)
	m3Parsed(t, "RequestReplyStopPledgeWar", got2, err, RequestReplyStopPledgeWar{Name: "Arthur", Answer: 1})
	m3Truncations(t, "RequestReplyStopPledgeWar", full, ParseRequestReplyStopPledgeWar)

	got3, err := ParseRequestReplySurrenderPledgeWar(full)
	m3Parsed(t, "RequestReplySurrenderPledgeWar", got3, err, RequestReplySurrenderPledgeWar{Name: "Arthur", Answer: 1})
	m3Truncations(t, "RequestReplySurrenderPledgeWar", full, ParseRequestReplySurrenderPledgeWar)
}

func TestParseRequestJoinPledge(t *testing.T) {
	// Mobius RequestJoinPledge.readImpl: d target, d pledge type.
	var b m3Body
	b.d(0x10000002).d(-1)
	got, err := ParseRequestJoinPledge(b.b)
	m3Parsed(t, "RequestJoinPledge", got, err, RequestJoinPledge{TargetID: 0x10000002, PledgeType: -1})
	m3Truncations(t, "RequestJoinPledge", b.b, ParseRequestJoinPledge)
}

func TestParseRequestGiveNickName(t *testing.T) {
	// Mobius RequestGiveNickName.readImpl: S target, S title.
	var b m3Body
	b.s("Lance").s("Sir")
	got, err := ParseRequestGiveNickName(b.b)
	m3Parsed(t, "RequestGiveNickName", got, err, RequestGiveNickName{Target: "Lance", Title: "Sir"})
	m3Truncations(t, "RequestGiveNickName", b.b, ParseRequestGiveNickName)
}

func TestParseRequestPledgePower(t *testing.T) {
	// Mobius RequestPledgePower.readImpl: d rank, d action, and d privs
	// only when the action is 2.
	var show m3Body
	show.d(6).d(1)
	got, err := ParseRequestPledgePower(show.b)
	m3Parsed(t, "RequestPledgePower show", got, err, RequestPledgePower{Rank: 6, Action: PledgePowerShow})
	m3Truncations(t, "RequestPledgePower show", show.b, ParseRequestPledgePower)

	// A show request with a trailing word does not read it.
	var showExtra m3Body
	showExtra.d(6).d(1).d(0x7fffffff)
	got, err = ParseRequestPledgePower(showExtra.b)
	m3Parsed(t, "RequestPledgePower show with extra", got, err, RequestPledgePower{Rank: 6, Action: PledgePowerShow})

	var set m3Body
	set.d(6).d(2).d(0x00800018)
	got, err = ParseRequestPledgePower(set.b)
	m3Parsed(t, "RequestPledgePower set", got, err, RequestPledgePower{Rank: 6, Action: PledgePowerSet, Privileges: 0x00800018})
	m3Truncations(t, "RequestPledgePower set", set.b, ParseRequestPledgePower)
}

func TestParsePledgeExPackets(t *testing.T) {
	// Mobius RequestPledgeSetAcademyMaster.readImpl: d set, S current
	// player, S target player.
	var academy m3Body
	academy.d(1).s("Morgan").s("Arthur")
	got1, err := ParseRequestPledgeSetAcademyMaster(academy.b)
	m3Parsed(t, "RequestPledgeSetAcademyMaster", got1, err,
		RequestPledgeSetAcademyMaster{Set: 1, MemberName: "Morgan", SponsorName: "Arthur"})
	m3Truncations(t, "RequestPledgeSetAcademyMaster", academy.b, ParseRequestPledgeSetAcademyMaster)

	// Mobius RequestPledgeMemberPowerInfo.readImpl and
	// RequestPledgeMemberInfo.readImpl: d unknown, S player.
	var member m3Body
	member.d(0).s("Lance")
	got2, err := ParseRequestPledgeMemberPowerInfo(member.b)
	m3Parsed(t, "RequestPledgeMemberPowerInfo", got2, err, RequestPledgeMemberPowerInfo{Name: "Lance"})
	m3Truncations(t, "RequestPledgeMemberPowerInfo", member.b, ParseRequestPledgeMemberPowerInfo)
	got3, err := ParseRequestPledgeMemberInfo(member.b)
	m3Parsed(t, "RequestPledgeMemberInfo", got3, err, RequestPledgeMemberInfo{Name: "Lance"})
	m3Truncations(t, "RequestPledgeMemberInfo", member.b, ParseRequestPledgeMemberInfo)

	// Mobius RequestPledgeSetMemberPowerGrade.readImpl: S member, d grade.
	var grade m3Body
	grade.s("Lance").d(7)
	got4, err := ParseRequestPledgeSetMemberPowerGrade(grade.b)
	m3Parsed(t, "RequestPledgeSetMemberPowerGrade", got4, err, RequestPledgeSetMemberPowerGrade{Name: "Lance", PowerGrade: 7})
	m3Truncations(t, "RequestPledgeSetMemberPowerGrade", grade.b, ParseRequestPledgeSetMemberPowerGrade)

	// Mobius RequestPledgeWarList.readImpl: d unknown, d tab.
	var war m3Body
	war.d(0).d(1)
	got5, err := ParseRequestPledgeWarList(war.b)
	m3Parsed(t, "RequestPledgeWarList", got5, err, RequestPledgeWarList{Tab: 1})
	m3Truncations(t, "RequestPledgeWarList", war.b, ParseRequestPledgeWarList)

	// Mobius RequestPledgeReorganizeMember.readImpl: d selected, S member,
	// d new pledge type, S selected member.
	var reorg m3Body
	reorg.d(1).s("Lance").d(1001).s("Gawain")
	got6, err := ParseRequestPledgeReorganizeMember(reorg.b)
	m3Parsed(t, "RequestPledgeReorganizeMember", got6, err, RequestPledgeReorganizeMember{
		MemberSelected: 1, MemberName: "Lance", NewPledgeType: 1001, SelectedMember: "Gawain",
	})
	m3Truncations(t, "RequestPledgeReorganizeMember", reorg.b, ParseRequestPledgeReorganizeMember)
}

// crestBody builds a crest upload: d length, then data.
func crestBody(length int32, data []byte) []byte {
	b := binary.LittleEndian.AppendUint32(nil, uint32(length))
	return append(b, data...)
}

// The crest uploads (Mobius RequestSetPledgeCrest,
// RequestExSetPledgeCrestLarge and RequestSetAllyCrest readImpl): d length;
// above the maximum Mobius returns without reading the data, else
// readBytes(length), which throws for a negative length or one the packet
// does not hold.
func TestParseCrestUploads(t *testing.T) {
	parsers := []struct {
		name  string
		parse func([]byte) (CrestUpload, error)
		max   int
	}{
		{"RequestSetPledgeCrest", ParseRequestSetPledgeCrest, MaxPledgeCrestSize},
		{"RequestExSetPledgeCrestLarge", ParseRequestExSetPledgeCrestLarge, MaxPledgeCrestLargeSize},
		{"RequestSetAllyCrest", ParseRequestSetAllyCrest, MaxAllyCrestSize},
	}
	for _, p := range parsers {
		image := bytes.Repeat([]byte{0xa5}, p.max)

		// The largest image.
		full := crestBody(int32(p.max), image)
		got, err := p.parse(full)
		m3Parsed(t, p.name+" at the maximum", got, err, CrestUpload{Length: int32(p.max), Data: image})
		m3Truncations(t, p.name+" at the maximum", full, p.parse)

		// A small image, with bytes after it that Mobius ignores.
		small := crestBody(3, []byte{1, 2, 3, 9, 9})
		got, err = p.parse(small)
		m3Parsed(t, p.name+" small", got, err, CrestUpload{Length: 3, Data: []byte{1, 2, 3}})

		// Length 0 deletes the crest.
		got, err = p.parse(crestBody(0, nil))
		m3Parsed(t, p.name+" delete", got, err, CrestUpload{Length: 0, Data: []byte{}})

		// One byte over the maximum: the length only, the data unread, even
		// when the packet holds it.
		over := crestBody(int32(p.max+1), append(bytes.Clone(image), 0xa5))
		got, err = p.parse(over)
		m3Parsed(t, p.name+" over the maximum", got, err, CrestUpload{Length: int32(p.max + 1)})
		got, err = p.parse(crestBody(0x7fffffff, nil))
		m3Parsed(t, p.name+" huge length", got, err, CrestUpload{Length: 0x7fffffff})

		// A negative length fails like readBytes.
		got, err = p.parse(crestBody(-1, image))
		m3Failed(t, p.name+" negative length", got, err)
	}
}

// fuzzCrest checks a crest parser on fuzzed input: no panic, failures are
// ErrShortPacket with the zero value, and a success holds the length and,
// up to the maximum, exactly the bytes after it.
func fuzzCrest(f *testing.F, parse func([]byte) (CrestUpload, error), maxSize int) {
	f.Add(crestBody(0, nil))
	f.Add(crestBody(4, []byte{1, 2, 3, 4}))
	f.Add(crestBody(int32(maxSize), make([]byte, maxSize)))
	f.Add(crestBody(int32(maxSize+1), nil))
	f.Add(crestBody(-1, []byte{1}))
	f.Add([]byte{1, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := parse(data)
		if err != nil {
			m3Failed(t, "fuzzed input", got, err)
			return
		}
		n := int32(binary.LittleEndian.Uint32(data))
		if got.Length != n {
			t.Fatalf("Length %d, the data says %d", got.Length, n)
		}
		if n > int32(maxSize) {
			if got.Data != nil {
				t.Fatalf("Data of %d bytes above the maximum", len(got.Data))
			}
			return
		}
		if !bytes.Equal(got.Data, data[4:4+n]) {
			t.Fatalf("Data % x, want % x", got.Data, data[4:4+n])
		}
	})
}

func FuzzParseRequestSetPledgeCrest(f *testing.F) {
	fuzzCrest(f, ParseRequestSetPledgeCrest, MaxPledgeCrestSize)
}

func FuzzParseRequestExSetPledgeCrestLarge(f *testing.F) {
	fuzzCrest(f, ParseRequestExSetPledgeCrestLarge, MaxPledgeCrestLargeSize)
}

func FuzzParseRequestSetAllyCrest(f *testing.F) {
	fuzzCrest(f, ParseRequestSetAllyCrest, MaxAllyCrestSize)
}

func FuzzParseRequestPledgePower(f *testing.F) {
	var set m3Body
	set.d(6).d(2).d(0x18)
	m3Fuzz(f, ParseRequestPledgePower, -1, set.b, []byte{6, 0, 0, 0, 1, 0, 0, 0})
}

func FuzzParseRequestPledgeReorganizeMember(f *testing.F) {
	var b m3Body
	b.d(1).s("Lance").d(1001).s("Gawain")
	m3Fuzz(f, ParseRequestPledgeReorganizeMember, -1, b.b)
}
