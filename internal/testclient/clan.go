package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// Client opcodes of the clan and alliance packets, from Mobius
// ClientPackets.
const (
	OpRequestJoinPledge              byte = 0x24
	OpRequestAnswerJoinPledge        byte = 0x25
	OpRequestWithdrawalPledge        byte = 0x26
	OpRequestOustPledgeMember        byte = 0x27
	OpRequestPledgeMemberList        byte = 0x3c
	OpRequestStartPledgeWar          byte = 0x4d
	OpRequestReplyStartPledgeWar     byte = 0x4e
	OpRequestStopPledgeWar           byte = 0x4f
	OpRequestReplyStopPledgeWar      byte = 0x50
	OpRequestSurrenderPledgeWar      byte = 0x51
	OpRequestReplySurrenderPledgeWar byte = 0x52
	OpRequestSetPledgeCrest          byte = 0x53
	OpRequestGiveNickName            byte = 0x55
	OpRequestPledgeInfo              byte = 0x66
	OpRequestPledgeExtendedInfo      byte = 0x67
	OpRequestPledgeCrest             byte = 0x68
	OpRequestJoinAlly                byte = 0x82
	OpRequestAnswerJoinAlly          byte = 0x83
	OpAllyLeave                      byte = 0x84
	OpAllyDismiss                    byte = 0x85
	OpRequestDismissAlly             byte = 0x86
	OpRequestSetAllyCrest            byte = 0x87
	OpRequestAllyCrest               byte = 0x88
	OpRequestAllyInfo                byte = 0x8e
	OpRequestPledgePower             byte = 0xc0
)

// Ex client sub-opcodes of the clan packets, from Mobius ExClientPackets.
const (
	ExOpRequestExPledgeCrestLarge        uint16 = 0x10
	ExOpRequestExSetPledgeCrestLarge     uint16 = 0x11
	ExOpRequestPledgeSetAcademyMaster    uint16 = 0x19
	ExOpRequestPledgePowerGradeList      uint16 = 0x1a
	ExOpRequestPledgeMemberPowerInfo     uint16 = 0x1b
	ExOpRequestPledgeSetMemberPowerGrade uint16 = 0x1c
	ExOpRequestPledgeMemberInfo          uint16 = 0x1d
	ExOpRequestPledgeWarList             uint16 = 0x1e
	ExOpRequestPledgeReorganizeMember    uint16 = 0x24
)

// The actions of RequestPledgePower.
const (
	PledgePowerShow int32 = 1
	PledgePowerSet  int32 = 2
)

// encodeClanInts builds a packet of d fields.
func encodeClanInts(op byte, vs ...int32) []byte {
	w := packets.NewWriter(1 + 4*len(vs))
	w.WriteUint8(op)
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// encodeClanNamedInt builds a packet of S name, d value: the war replies.
func encodeClanNamedInt(op byte, name string, v int32) []byte {
	w := packets.NewWriter(7 + 2*len(name))
	w.WriteUint8(op)
	w.WriteString(name)
	w.WriteInt32(v)
	return w.Bytes()
}

// encodeRequestGiveNickName builds RequestGiveNickName: S target, S title.
func encodeRequestGiveNickName(target, title string) []byte {
	w := packets.NewWriter(5 + 2*len(target) + 2*len(title))
	w.WriteUint8(OpRequestGiveNickName)
	w.WriteString(target)
	w.WriteString(title)
	return w.Bytes()
}

// encodeRequestPledgePower builds RequestPledgePower: d rank, d action,
// and d privileges for PledgePowerSet only.
func encodeRequestPledgePower(rank, action, privileges int32) []byte {
	if action == PledgePowerSet {
		return encodeClanInts(OpRequestPledgePower, rank, action, privileges)
	}
	return encodeClanInts(OpRequestPledgePower, rank, action)
}

// encodeCrestUpload builds a crest upload: d length, then the image. With
// OpExClient, sub is the sub-opcode; else it is ignored.
func encodeCrestUpload(op byte, sub uint16, length int32, data []byte) []byte {
	w := packets.NewWriter(7 + len(data))
	w.WriteUint8(op)
	if op == OpExClient {
		w.WriteUint16(sub)
	}
	w.WriteInt32(length)
	w.WriteBytes(data)
	return w.Bytes()
}

// clanExWriter starts an Ex client packet.
func clanExWriter(sub uint16, size int) *packets.Writer {
	w := packets.NewWriter(3 + size)
	w.WriteUint8(OpExClient)
	w.WriteUint16(sub)
	return w
}

// encodeExIntString builds an Ex packet of d value, S name:
// RequestPledgeMemberPowerInfo and RequestPledgeMemberInfo.
func encodeExIntString(sub uint16, v int32, name string) []byte {
	w := clanExWriter(sub, 6+2*len(name))
	w.WriteInt32(v)
	w.WriteString(name)
	return w.Bytes()
}

// encodeRequestPledgeSetAcademyMaster builds RequestPledgeSetAcademyMaster:
// d set, S member, S sponsor.
func encodeRequestPledgeSetAcademyMaster(set int32, member, sponsor string) []byte {
	w := clanExWriter(ExOpRequestPledgeSetAcademyMaster, 8+2*len(member)+2*len(sponsor))
	w.WriteInt32(set)
	w.WriteString(member)
	w.WriteString(sponsor)
	return w.Bytes()
}

// encodeRequestPledgeSetMemberPowerGrade builds
// RequestPledgeSetMemberPowerGrade: S member, d grade.
func encodeRequestPledgeSetMemberPowerGrade(member string, grade int32) []byte {
	w := clanExWriter(ExOpRequestPledgeSetMemberPowerGrade, 6+2*len(member))
	w.WriteString(member)
	w.WriteInt32(grade)
	return w.Bytes()
}

// encodeRequestPledgeWarList builds RequestPledgeWarList: d page (0), d
// tab.
func encodeRequestPledgeWarList(tab int32) []byte {
	w := clanExWriter(ExOpRequestPledgeWarList, 8)
	w.WriteInt32(0)
	w.WriteInt32(tab)
	return w.Bytes()
}

// encodeRequestExPledgeCrestLarge builds RequestExPledgeCrestLarge: d crest.
func encodeRequestExPledgeCrestLarge(crestID int32) []byte {
	w := clanExWriter(ExOpRequestExPledgeCrestLarge, 4)
	w.WriteInt32(crestID)
	return w.Bytes()
}

// Reorganize is a RequestPledgeReorganizeMember: move Member to the unit
// NewPledgeType, swapping with Selected.
type Reorganize struct {
	// MemberSelected is 0 when no member is selected; Mobius then does
	// nothing.
	MemberSelected int32
	Member         string
	NewPledgeType  int32
	Selected       string
}

// encodeRequestPledgeReorganizeMember builds RequestPledgeReorganizeMember:
// d selected, S member, d new pledge type, S selected member.
func encodeRequestPledgeReorganizeMember(r Reorganize) []byte {
	w := clanExWriter(ExOpRequestPledgeReorganizeMember, 12+2*len(r.Member)+2*len(r.Selected))
	w.WriteInt32(r.MemberSelected)
	w.WriteString(r.Member)
	w.WriteInt32(r.NewPledgeType)
	w.WriteString(r.Selected)
	return w.Bytes()
}

// InviteToClan invites the player objectID into the clan or the unit
// pledgeType (RequestJoinPledge) without waiting. The invited player gets
// AskJoinPledge.
func (c *GameConn) InviteToClan(ctx context.Context, objectID, pledgeType int32) error {
	return c.send(ctx, encodeClanInts(OpRequestJoinPledge, objectID, pledgeType))
}

// AwaitClanInvite waits for an AskJoinPledge.
func (c *GameConn) AwaitClanInvite(ctx context.Context) (AskJoinPledge, error) {
	return Await[AskJoinPledge](ctx, c, nil)
}

// AnswerClanInvite answers AskJoinPledge (RequestAnswerJoinPledge) without
// waiting: 0 refuses, other values accept.
func (c *GameConn) AnswerClanInvite(ctx context.Context, answer int32) error {
	return c.send(ctx, encodeClanInts(OpRequestAnswerJoinPledge, answer))
}

// AcceptClanInvite accepts AskJoinPledge and returns the JoinPledge that
// answers it.
func (c *GameConn) AcceptClanInvite(ctx context.Context) (JoinPledge, error) {
	if err := c.AnswerClanInvite(ctx, 1); err != nil {
		return JoinPledge{}, err
	}
	return Await[JoinPledge](ctx, c, nil)
}

// LeaveClan leaves the clan (RequestWithdrawalPledge) without waiting.
func (c *GameConn) LeaveClan(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestWithdrawalPledge))
}

// DismissClanMember dismisses the member name (RequestOustPledgeMember)
// without waiting.
func (c *GameConn) DismissClanMember(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestOustPledgeMember, name))
}

// RequestPledgeMemberList asks for the clan window without waiting.
func (c *GameConn) RequestPledgeMemberList(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestPledgeMemberList))
}

// ClanMembers asks for the clan window (RequestPledgeMemberList) and
// returns the PledgeShowMemberListAll that answers it. A player without a
// clan gets no answer, so the call ends with the context.
func (c *GameConn) ClanMembers(ctx context.Context) (PledgeShowMemberListAll, error) {
	if err := c.RequestPledgeMemberList(ctx); err != nil {
		return PledgeShowMemberListAll{}, err
	}
	return Await[PledgeShowMemberListAll](ctx, c, nil)
}

// DeclareClanWar declares war on the clan clanName
// (RequestStartPledgeWar) without waiting.
func (c *GameConn) DeclareClanWar(ctx context.Context, clanName string) error {
	return c.send(ctx, encodeString(OpRequestStartPledgeWar, clanName))
}

// ReplyStartPledgeWar answers StartPledgeWar (RequestReplyStartPledgeWar):
// 1 accepts.
func (c *GameConn) ReplyStartPledgeWar(ctx context.Context, requestor string, answer int32) error {
	return c.send(ctx, encodeClanNamedInt(OpRequestReplyStartPledgeWar, requestor, answer))
}

// StopClanWar ends the war on clanName (RequestStopPledgeWar) without
// waiting.
func (c *GameConn) StopClanWar(ctx context.Context, clanName string) error {
	return c.send(ctx, encodeString(OpRequestStopPledgeWar, clanName))
}

// ReplyStopPledgeWar answers StopPledgeWar (RequestReplyStopPledgeWar): 1
// accepts.
func (c *GameConn) ReplyStopPledgeWar(ctx context.Context, requestor string, answer int32) error {
	return c.send(ctx, encodeClanNamedInt(OpRequestReplyStopPledgeWar, requestor, answer))
}

// SurrenderClanWar surrenders the war with clanName
// (RequestSurrenderPledgeWar) without waiting.
func (c *GameConn) SurrenderClanWar(ctx context.Context, clanName string) error {
	return c.send(ctx, encodeString(OpRequestSurrenderPledgeWar, clanName))
}

// ReplySurrenderPledgeWar answers SurrenderPledgeWar
// (RequestReplySurrenderPledgeWar): 1 accepts.
func (c *GameConn) ReplySurrenderPledgeWar(ctx context.Context, requestor string, answer int32) error {
	return c.send(ctx, encodeClanNamedInt(OpRequestReplySurrenderPledgeWar, requestor, answer))
}

// SetPledgeCrest uploads the clan crest (RequestSetPledgeCrest) without
// waiting; empty data deletes it.
func (c *GameConn) SetPledgeCrest(ctx context.Context, data []byte) error {
	return c.send(ctx, encodeCrestUpload(OpRequestSetPledgeCrest, 0, int32(len(data)), data)) //nolint:gosec // test images are small
}

// SetPledgeCrestLarge uploads the large clan crest
// (RequestExSetPledgeCrestLarge) without waiting; empty data deletes it.
func (c *GameConn) SetPledgeCrestLarge(ctx context.Context, data []byte) error {
	return c.send(ctx, encodeCrestUpload(OpExClient, ExOpRequestExSetPledgeCrestLarge, int32(len(data)), data)) //nolint:gosec // test images are small
}

// SetAllyCrest uploads the alliance crest (RequestSetAllyCrest) without
// waiting; empty data deletes it.
func (c *GameConn) SetAllyCrest(ctx context.Context, data []byte) error {
	return c.send(ctx, encodeCrestUpload(OpRequestSetAllyCrest, 0, int32(len(data)), data)) //nolint:gosec // test images are small
}

// GiveTitle sets the title of the clan member target
// (RequestGiveNickName) without waiting.
func (c *GameConn) GiveTitle(ctx context.Context, target, title string) error {
	return c.send(ctx, encodeRequestGiveNickName(target, title))
}

// RequestPledgeInfo asks for the clan clanID without waiting.
func (c *GameConn) RequestPledgeInfo(ctx context.Context, clanID int32) error {
	return c.send(ctx, encodeClanInts(OpRequestPledgeInfo, clanID))
}

// ClanInfo asks for the clan clanID (RequestPledgeInfo) and returns the
// PledgeInfo that answers it.
func (c *GameConn) ClanInfo(ctx context.Context, clanID int32) (PledgeInfo, error) {
	if err := c.RequestPledgeInfo(ctx, clanID); err != nil {
		return PledgeInfo{}, err
	}
	return Await(ctx, c, func(p PledgeInfo) bool { return p.ClanID == clanID })
}

// RequestPledgeExtendedInfo sends RequestPledgeExtendedInfo, which Mobius
// ignores.
func (c *GameConn) RequestPledgeExtendedInfo(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestPledgeExtendedInfo, name))
}

// PledgeCrestImage asks for a clan crest (RequestPledgeCrest) and returns
// the PledgeCrest that answers it.
func (c *GameConn) PledgeCrestImage(ctx context.Context, crestID int32) (PledgeCrest, error) {
	if err := c.send(ctx, encodeClanInts(OpRequestPledgeCrest, crestID)); err != nil {
		return PledgeCrest{}, err
	}
	return Await(ctx, c, func(p PledgeCrest) bool { return p.CrestID == crestID })
}

// PledgeCrestLargeImage asks for a large clan crest
// (RequestExPledgeCrestLarge) and returns the ExPledgeEmblem that answers
// it.
func (c *GameConn) PledgeCrestLargeImage(ctx context.Context, crestID int32) (ExPledgeEmblem, error) {
	if err := c.send(ctx, encodeRequestExPledgeCrestLarge(crestID)); err != nil {
		return ExPledgeEmblem{}, err
	}
	return Await(ctx, c, func(p ExPledgeEmblem) bool { return p.CrestID == crestID })
}

// AllyCrestImage asks for an alliance crest (RequestAllyCrest) and returns
// the AllyCrest that answers it.
func (c *GameConn) AllyCrestImage(ctx context.Context, crestID int32) (AllyCrest, error) {
	if err := c.send(ctx, encodeClanInts(OpRequestAllyCrest, crestID)); err != nil {
		return AllyCrest{}, err
	}
	return Await(ctx, c, func(p AllyCrest) bool { return p.CrestID == crestID })
}

// RankPrivileges asks for the privileges of a clan rank (RequestPledgePower
// with PledgePowerShow) and returns the ManagePledgePower that answers it.
func (c *GameConn) RankPrivileges(ctx context.Context, rank int32) (ManagePledgePower, error) {
	if err := c.send(ctx, encodeRequestPledgePower(rank, PledgePowerShow, 0)); err != nil {
		return ManagePledgePower{}, err
	}
	return Await[ManagePledgePower](ctx, c, nil)
}

// SetRankPrivileges sets the privileges of a clan rank (RequestPledgePower
// with PledgePowerSet) without waiting; Mobius does not answer.
func (c *GameConn) SetRankPrivileges(ctx context.Context, rank, privileges int32) error {
	return c.send(ctx, encodeRequestPledgePower(rank, PledgePowerSet, privileges))
}

// SetAcademyMaster makes sponsor the sponsor of the academy member member
// (RequestPledgeSetAcademyMaster), or removes the sponsor when set is
// false, without waiting.
func (c *GameConn) SetAcademyMaster(ctx context.Context, set bool, member, sponsor string) error {
	return c.send(ctx, encodeRequestPledgeSetAcademyMaster(boolInt32(set), member, sponsor))
}

// PowerGrades asks for the clan's ranks (RequestPledgePowerGradeList) and
// returns the PledgePowerGradeList that answers it.
func (c *GameConn) PowerGrades(ctx context.Context) (PledgePowerGradeList, error) {
	if err := c.send(ctx, encodeEx(ExOpRequestPledgePowerGradeList)); err != nil {
		return PledgePowerGradeList{}, err
	}
	return Await[PledgePowerGradeList](ctx, c, nil)
}

// MemberPowerInfo asks for a member's rank and privileges
// (RequestPledgeMemberPowerInfo) and returns the PledgeReceivePowerInfo
// that answers it.
func (c *GameConn) MemberPowerInfo(ctx context.Context, name string) (PledgeReceivePowerInfo, error) {
	if err := c.send(ctx, encodeExIntString(ExOpRequestPledgeMemberPowerInfo, 0, name)); err != nil {
		return PledgeReceivePowerInfo{}, err
	}
	return Await(ctx, c, func(p PledgeReceivePowerInfo) bool { return p.Name == name })
}

// SetMemberPowerGrade gives the member name the rank grade
// (RequestPledgeSetMemberPowerGrade) without waiting.
func (c *GameConn) SetMemberPowerGrade(ctx context.Context, name string, grade int32) error {
	return c.send(ctx, encodeRequestPledgeSetMemberPowerGrade(name, grade))
}

// MemberInfo asks for the detail of a member (RequestPledgeMemberInfo) and
// returns the PledgeReceiveMemberInfo that answers it.
func (c *GameConn) MemberInfo(ctx context.Context, name string) (PledgeReceiveMemberInfo, error) {
	if err := c.send(ctx, encodeExIntString(ExOpRequestPledgeMemberInfo, 0, name)); err != nil {
		return PledgeReceiveMemberInfo{}, err
	}
	return Await(ctx, c, func(p PledgeReceiveMemberInfo) bool { return p.Name == name })
}

// WarList asks for a tab of the clan war list (RequestPledgeWarList) and
// returns the PledgeReceiveWarList that answers it.
func (c *GameConn) WarList(ctx context.Context, tab int32) (PledgeReceiveWarList, error) {
	if err := c.send(ctx, encodeRequestPledgeWarList(tab)); err != nil {
		return PledgeReceiveWarList{}, err
	}
	return Await(ctx, c, func(p PledgeReceiveWarList) bool { return p.Tab == tab })
}

// ReorganizeClanMember moves a member to another unit
// (RequestPledgeReorganizeMember) without waiting.
func (c *GameConn) ReorganizeClanMember(ctx context.Context, r Reorganize) error {
	return c.send(ctx, encodeRequestPledgeReorganizeMember(r))
}

// InviteToAlly invites the clan leader objectID's clan into the alliance
// (RequestJoinAlly) without waiting. The invited leader gets AskJoinAlly.
func (c *GameConn) InviteToAlly(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeClanInts(OpRequestJoinAlly, objectID))
}

// AwaitAllyInvite waits for an AskJoinAlly.
func (c *GameConn) AwaitAllyInvite(ctx context.Context) (AskJoinAlly, error) {
	return Await[AskJoinAlly](ctx, c, nil)
}

// AnswerAllyInvite answers AskJoinAlly (RequestAnswerJoinAlly) without
// waiting: 0 refuses, other values accept.
func (c *GameConn) AnswerAllyInvite(ctx context.Context, answer int32) error {
	return c.send(ctx, encodeClanInts(OpRequestAnswerJoinAlly, answer))
}

// LeaveAlly takes the clan out of its alliance (AllyLeave) without
// waiting.
func (c *GameConn) LeaveAlly(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpAllyLeave))
}

// DismissAllyClan expels the clan clanName from the alliance (AllyDismiss)
// without waiting.
func (c *GameConn) DismissAllyClan(ctx context.Context, clanName string) error {
	return c.send(ctx, encodeString(OpAllyDismiss, clanName))
}

// DissolveAlly dissolves the alliance (RequestDismissAlly) without
// waiting.
func (c *GameConn) DissolveAlly(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestDismissAlly))
}

// RequestAllyInfo asks for the alliance window without waiting.
func (c *GameConn) RequestAllyInfo(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestAllyInfo))
}

// AllyInfo asks for the alliance window (RequestAllyInfo) and returns the
// AllianceInfo that answers it. A player without an alliance gets a
// system message instead, so the call ends with the context.
func (c *GameConn) AllyInfo(ctx context.Context) (AllianceInfo, error) {
	if err := c.RequestAllyInfo(ctx); err != nil {
		return AllianceInfo{}, err
	}
	return Await[AllianceInfo](ctx, c, nil)
}
