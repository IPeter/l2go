package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Sub-opcodes of the extended clan packets (Mobius ExClientPackets, after
// 0xD0). The parser of RequestExSetPledgeCrestLarge is in crest.go.
const (
	ExOpRequestExPledgeCrestLarge        = 0x10
	ExOpRequestExSetPledgeCrestLarge     = 0x11
	ExOpRequestPledgeSetAcademyMaster    = 0x19
	ExOpRequestPledgePowerGradeList      = 0x1a
	ExOpRequestPledgeMemberPowerInfo     = 0x1b
	ExOpRequestPledgeSetMemberPowerGrade = 0x1c
	ExOpRequestPledgeMemberInfo          = 0x1d
	ExOpRequestPledgeWarList             = 0x1e
	ExOpRequestPledgeReorganizeMember    = 0x24
)

// The parsers below take the body after the sub-opcode (ExPacket.Body).

// RequestExPledgeCrestLarge asks for the image of a large clan crest
// (ExPledgeEmblem).
type RequestExPledgeCrestLarge struct {
	CrestID int32
}

// ParseRequestExPledgeCrestLarge parses a RequestExPledgeCrestLarge body
// (Mobius RequestExPledgeCrestLarge.readImpl): d crest ID.
func ParseRequestExPledgeCrestLarge(data []byte) (RequestExPledgeCrestLarge, error) {
	v, err := readClanInt32("RequestExPledgeCrestLarge", data)
	return RequestExPledgeCrestLarge{CrestID: v}, err
}

// RequestPledgeSetAcademyMaster makes a clan member the sponsor of an
// academy member, or removes the sponsor.
type RequestPledgeSetAcademyMaster struct {
	Set        int32 // 0 removes the sponsor, other values set it
	MemberName string
	// SponsorName is the sponsor's name (Mobius _targetPlayerName).
	SponsorName string
}

// ParseRequestPledgeSetAcademyMaster parses a RequestPledgeSetAcademyMaster
// body (Mobius RequestPledgeSetAcademyMaster.readImpl): d set, S member,
// S sponsor.
func ParseRequestPledgeSetAcademyMaster(data []byte) (RequestPledgeSetAcademyMaster, error) {
	r := packets.NewReader(data)
	p := RequestPledgeSetAcademyMaster{Set: r.ReadInt32(), MemberName: r.ReadString(), SponsorName: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestPledgeSetAcademyMaster{}, fmt.Errorf("RequestPledgeSetAcademyMaster: %w", err)
	}
	return p, nil
}

// RequestPledgePowerGradeList asks for the clan's ranks
// (PledgePowerGradeList). It has no fields.
type RequestPledgePowerGradeList struct{}

// ParseRequestPledgePowerGradeList parses a RequestPledgePowerGradeList
// body. Like Mobius it reads nothing, so it never fails.
func ParseRequestPledgePowerGradeList([]byte) (RequestPledgePowerGradeList, error) {
	return RequestPledgePowerGradeList{}, nil
}

// RequestPledgeMemberPowerInfo asks for a member's rank and privileges
// (PledgeReceivePowerInfo).
type RequestPledgeMemberPowerInfo struct {
	Unknown int32 // Mobius reads it and ignores it
	Name    string
}

// ParseRequestPledgeMemberPowerInfo parses a RequestPledgeMemberPowerInfo
// body (Mobius RequestPledgeMemberPowerInfo.readImpl): d unknown, S name.
func ParseRequestPledgeMemberPowerInfo(data []byte) (RequestPledgeMemberPowerInfo, error) {
	r := packets.NewReader(data)
	p := RequestPledgeMemberPowerInfo{Unknown: r.ReadInt32(), Name: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestPledgeMemberPowerInfo{}, fmt.Errorf("RequestPledgeMemberPowerInfo: %w", err)
	}
	return p, nil
}

// RequestPledgeSetMemberPowerGrade gives a member another rank.
type RequestPledgeSetMemberPowerGrade struct {
	Name       string
	PowerGrade int32
}

// ParseRequestPledgeSetMemberPowerGrade parses a
// RequestPledgeSetMemberPowerGrade body (Mobius
// RequestPledgeSetMemberPowerGrade.readImpl): S member, d power grade.
func ParseRequestPledgeSetMemberPowerGrade(data []byte) (RequestPledgeSetMemberPowerGrade, error) {
	name, grade, err := readClanNamedAnswer("RequestPledgeSetMemberPowerGrade", data)
	return RequestPledgeSetMemberPowerGrade{Name: name, PowerGrade: grade}, err
}

// RequestPledgeMemberInfo asks for the detail of a member
// (PledgeReceiveMemberInfo).
type RequestPledgeMemberInfo struct {
	Unknown int32 // Mobius reads it and ignores it
	Name    string
}

// ParseRequestPledgeMemberInfo parses a RequestPledgeMemberInfo body (Mobius
// RequestPledgeMemberInfo.readImpl): d unknown, S name.
func ParseRequestPledgeMemberInfo(data []byte) (RequestPledgeMemberInfo, error) {
	r := packets.NewReader(data)
	p := RequestPledgeMemberInfo{Unknown: r.ReadInt32(), Name: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestPledgeMemberInfo{}, fmt.Errorf("RequestPledgeMemberInfo: %w", err)
	}
	return p, nil
}

// RequestPledgeWarList asks for a tab of the clan war list
// (PledgeReceiveWarList).
type RequestPledgeWarList struct {
	Unknown int32 // Mobius reads it and ignores it (the page)
	// Tab is serverpackets.WarListDeclared or WarListUnderAttack; Mobius
	// treats any value but 0 as the second.
	Tab int32
}

// ParseRequestPledgeWarList parses a RequestPledgeWarList body (Mobius
// RequestPledgeWarList.readImpl): d unknown, d tab.
func ParseRequestPledgeWarList(data []byte) (RequestPledgeWarList, error) {
	r := packets.NewReader(data)
	p := RequestPledgeWarList{Unknown: r.ReadInt32(), Tab: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestPledgeWarList{}, fmt.Errorf("RequestPledgeWarList: %w", err)
	}
	return p, nil
}

// RequestPledgeReorganizeMember moves a member to another unit, swapping
// places with a member of that unit when one is selected.
type RequestPledgeReorganizeMember struct {
	// MemberSelected is 0 when no member is selected; Mobius then does
	// nothing.
	MemberSelected int32
	MemberName     string
	NewPledgeType  int32
	// SelectedMember is the member to swap with.
	SelectedMember string
}

// ParseRequestPledgeReorganizeMember parses a RequestPledgeReorganizeMember
// body (Mobius RequestPledgeReorganizeMember.readImpl): d selected, S
// member, d new pledge type, S selected member.
func ParseRequestPledgeReorganizeMember(data []byte) (RequestPledgeReorganizeMember, error) {
	r := packets.NewReader(data)
	p := RequestPledgeReorganizeMember{
		MemberSelected: r.ReadInt32(),
		MemberName:     r.ReadString(),
		NewPledgeType:  r.ReadInt32(),
		SelectedMember: r.ReadString(),
	}
	if err := r.Err(); err != nil {
		return RequestPledgeReorganizeMember{}, fmt.Errorf("RequestPledgeReorganizeMember: %w", err)
	}
	return p, nil
}
