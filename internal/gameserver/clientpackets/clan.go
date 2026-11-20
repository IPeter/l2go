package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the clan packets (Mobius ClientPackets). The crest uploads are
// in crest.go.
const (
	OpRequestJoinPledge              = 0x24
	OpRequestAnswerJoinPledge        = 0x25
	OpRequestWithdrawalPledge        = 0x26
	OpRequestOustPledgeMember        = 0x27
	OpRequestPledgeMemberList        = 0x3c
	OpRequestStartPledgeWar          = 0x4d
	OpRequestReplyStartPledgeWar     = 0x4e
	OpRequestStopPledgeWar           = 0x4f
	OpRequestReplyStopPledgeWar      = 0x50
	OpRequestSurrenderPledgeWar      = 0x51
	OpRequestReplySurrenderPledgeWar = 0x52
	OpRequestGiveNickName            = 0x55
	OpRequestPledgeInfo              = 0x66
	OpRequestPledgeExtendedInfo      = 0x67
	OpRequestPledgeCrest             = 0x68
	OpRequestPledgePower             = 0xc0
)

// The actions of RequestPledgePower.
const (
	PledgePowerShow = 1 // show a rank's privileges (ManagePledgePower)
	PledgePowerSet  = 2 // set a rank's privileges; only then Privileges is sent
)

// The clan and alliance packets share a few layouts. These helpers read
// them and wrap a short body with the packet name.

// readClanInt32 reads a body of one d.
func readClanInt32(packet string, data []byte) (int32, error) {
	r := packets.NewReader(data)
	v := r.ReadInt32()
	if err := r.Err(); err != nil {
		return 0, fmt.Errorf("%s: %w", packet, err)
	}
	return v, nil
}

// readClanString reads a body of one S.
func readClanString(packet string, data []byte) (string, error) {
	r := packets.NewReader(data)
	s := r.ReadString()
	if err := r.Err(); err != nil {
		return "", fmt.Errorf("%s: %w", packet, err)
	}
	return s, nil
}

// readClanNamedAnswer reads a body of S name, d answer.
func readClanNamedAnswer(packet string, data []byte) (string, int32, error) {
	r := packets.NewReader(data)
	name := r.ReadString()
	answer := r.ReadInt32()
	if err := r.Err(); err != nil {
		return "", 0, fmt.Errorf("%s: %w", packet, err)
	}
	return name, answer, nil
}

// RequestJoinPledge invites a player into the clan or one of its units.
type RequestJoinPledge struct {
	TargetID   int32 // the invited player's object ID
	PledgeType int32 // the unit, a serverpackets.Pledge value
}

// ParseRequestJoinPledge parses a RequestJoinPledge body (Mobius
// RequestJoinPledge.readImpl): d target, d pledge type.
func ParseRequestJoinPledge(data []byte) (RequestJoinPledge, error) {
	r := packets.NewReader(data)
	p := RequestJoinPledge{TargetID: r.ReadInt32(), PledgeType: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestJoinPledge{}, fmt.Errorf("RequestJoinPledge: %w", err)
	}
	return p, nil
}

// RequestAnswerJoinPledge answers a clan invitation.
type RequestAnswerJoinPledge struct {
	Answer int32 // 0 refuses, other values accept
}

// ParseRequestAnswerJoinPledge parses a RequestAnswerJoinPledge body (Mobius
// RequestAnswerJoinPledge.readImpl): d answer.
func ParseRequestAnswerJoinPledge(data []byte) (RequestAnswerJoinPledge, error) {
	v, err := readClanInt32("RequestAnswerJoinPledge", data)
	return RequestAnswerJoinPledge{Answer: v}, err
}

// RequestWithdrawalPledge leaves the clan. It has no fields.
type RequestWithdrawalPledge struct{}

// ParseRequestWithdrawalPledge parses a RequestWithdrawalPledge body. Like
// Mobius it reads nothing, so it never fails.
func ParseRequestWithdrawalPledge([]byte) (RequestWithdrawalPledge, error) {
	return RequestWithdrawalPledge{}, nil
}

// RequestOustPledgeMember dismisses a clan member.
type RequestOustPledgeMember struct {
	Name string // the member's name
}

// ParseRequestOustPledgeMember parses a RequestOustPledgeMember body (Mobius
// RequestOustPledgeMember.readImpl): S name.
func ParseRequestOustPledgeMember(data []byte) (RequestOustPledgeMember, error) {
	s, err := readClanString("RequestOustPledgeMember", data)
	return RequestOustPledgeMember{Name: s}, err
}

// RequestPledgeMemberList asks for the clan window
// (PledgeShowMemberListAll). It has no fields.
type RequestPledgeMemberList struct{}

// ParseRequestPledgeMemberList parses a RequestPledgeMemberList body. Like
// Mobius it reads nothing, so it never fails.
func ParseRequestPledgeMemberList([]byte) (RequestPledgeMemberList, error) {
	return RequestPledgeMemberList{}, nil
}

// RequestStartPledgeWar declares war on a clan.
type RequestStartPledgeWar struct {
	ClanName string
}

// ParseRequestStartPledgeWar parses a RequestStartPledgeWar body (Mobius
// RequestStartPledgeWar.readImpl): S clan name.
func ParseRequestStartPledgeWar(data []byte) (RequestStartPledgeWar, error) {
	s, err := readClanString("RequestStartPledgeWar", data)
	return RequestStartPledgeWar{ClanName: s}, err
}

// RequestReplyStartPledgeWar answers StartPledgeWar.
type RequestReplyStartPledgeWar struct {
	Name   string // the requestor's name; Mobius reads it and ignores it
	Answer int32  // 1 accepts
}

// ParseRequestReplyStartPledgeWar parses a RequestReplyStartPledgeWar body
// (Mobius RequestReplyStartPledgeWar.readImpl): S name, d answer.
func ParseRequestReplyStartPledgeWar(data []byte) (RequestReplyStartPledgeWar, error) {
	name, answer, err := readClanNamedAnswer("RequestReplyStartPledgeWar", data)
	return RequestReplyStartPledgeWar{Name: name, Answer: answer}, err
}

// RequestStopPledgeWar ends a war on a clan.
type RequestStopPledgeWar struct {
	ClanName string
}

// ParseRequestStopPledgeWar parses a RequestStopPledgeWar body (Mobius
// RequestStopPledgeWar.readImpl): S clan name.
func ParseRequestStopPledgeWar(data []byte) (RequestStopPledgeWar, error) {
	s, err := readClanString("RequestStopPledgeWar", data)
	return RequestStopPledgeWar{ClanName: s}, err
}

// RequestReplyStopPledgeWar answers StopPledgeWar.
type RequestReplyStopPledgeWar struct {
	Name   string // the requestor's name; Mobius reads it and ignores it
	Answer int32  // 1 accepts
}

// ParseRequestReplyStopPledgeWar parses a RequestReplyStopPledgeWar body
// (Mobius RequestReplyStopPledgeWar.readImpl): S name, d answer.
func ParseRequestReplyStopPledgeWar(data []byte) (RequestReplyStopPledgeWar, error) {
	name, answer, err := readClanNamedAnswer("RequestReplyStopPledgeWar", data)
	return RequestReplyStopPledgeWar{Name: name, Answer: answer}, err
}

// RequestSurrenderPledgeWar surrenders a war to a clan.
type RequestSurrenderPledgeWar struct {
	ClanName string
}

// ParseRequestSurrenderPledgeWar parses a RequestSurrenderPledgeWar body
// (Mobius RequestSurrenderPledgeWar.readImpl): S clan name.
func ParseRequestSurrenderPledgeWar(data []byte) (RequestSurrenderPledgeWar, error) {
	s, err := readClanString("RequestSurrenderPledgeWar", data)
	return RequestSurrenderPledgeWar{ClanName: s}, err
}

// RequestReplySurrenderPledgeWar answers SurrenderPledgeWar.
type RequestReplySurrenderPledgeWar struct {
	Name   string // the requestor's name
	Answer int32  // 1 accepts
}

// ParseRequestReplySurrenderPledgeWar parses a
// RequestReplySurrenderPledgeWar body (Mobius
// RequestReplySurrenderPledgeWar.readImpl): S name, d answer.
func ParseRequestReplySurrenderPledgeWar(data []byte) (RequestReplySurrenderPledgeWar, error) {
	name, answer, err := readClanNamedAnswer("RequestReplySurrenderPledgeWar", data)
	return RequestReplySurrenderPledgeWar{Name: name, Answer: answer}, err
}

// RequestGiveNickName sets a clan member's title, or the player's own.
type RequestGiveNickName struct {
	Target string // the member's name
	Title  string
}

// ParseRequestGiveNickName parses a RequestGiveNickName body (Mobius
// RequestGiveNickName.readImpl): S target, S title. The title length is
// checked by the game layer.
func ParseRequestGiveNickName(data []byte) (RequestGiveNickName, error) {
	r := packets.NewReader(data)
	p := RequestGiveNickName{Target: r.ReadString(), Title: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestGiveNickName{}, fmt.Errorf("RequestGiveNickName: %w", err)
	}
	return p, nil
}

// RequestPledgeInfo asks for a clan's name and alliance (PledgeInfo).
type RequestPledgeInfo struct {
	ClanID int32
}

// ParseRequestPledgeInfo parses a RequestPledgeInfo body (Mobius
// RequestPledgeInfo.readImpl): d clan ID.
func ParseRequestPledgeInfo(data []byte) (RequestPledgeInfo, error) {
	v, err := readClanInt32("RequestPledgeInfo", data)
	return RequestPledgeInfo{ClanID: v}, err
}

// RequestPledgeExtendedInfo is a request Mobius reads and ignores.
type RequestPledgeExtendedInfo struct {
	Name string // a name, maybe the clan's
}

// ParseRequestPledgeExtendedInfo parses a RequestPledgeExtendedInfo body
// (Mobius RequestPledgeExtendedInfo.readImpl): S name.
func ParseRequestPledgeExtendedInfo(data []byte) (RequestPledgeExtendedInfo, error) {
	s, err := readClanString("RequestPledgeExtendedInfo", data)
	return RequestPledgeExtendedInfo{Name: s}, err
}

// RequestPledgeCrest asks for the image of a clan crest (PledgeCrest).
type RequestPledgeCrest struct {
	CrestID int32
}

// ParseRequestPledgeCrest parses a RequestPledgeCrest body (Mobius
// RequestPledgeCrest.readImpl): d crest ID.
func ParseRequestPledgeCrest(data []byte) (RequestPledgeCrest, error) {
	v, err := readClanInt32("RequestPledgeCrest", data)
	return RequestPledgeCrest{CrestID: v}, err
}

// RequestPledgePower shows or sets the privileges of a clan rank.
type RequestPledgePower struct {
	Rank int32
	// Action is PledgePowerShow or PledgePowerSet; Mobius shows the
	// privileges for any action but PledgePowerSet.
	Action int32
	// Privileges is the new privilege mask; it is only sent, and only read,
	// with PledgePowerSet.
	Privileges int32
}

// ParseRequestPledgePower parses a RequestPledgePower body (Mobius
// RequestPledgePower.readImpl): d rank, d action, and d privileges when
// the action is PledgePowerSet.
func ParseRequestPledgePower(data []byte) (RequestPledgePower, error) {
	r := packets.NewReader(data)
	p := RequestPledgePower{Rank: r.ReadInt32(), Action: r.ReadInt32()}
	if p.Action == PledgePowerSet {
		p.Privileges = r.ReadInt32()
	}
	if err := r.Err(); err != nil {
		return RequestPledgePower{}, fmt.Errorf("RequestPledgePower: %w", err)
	}
	return p, nil
}
