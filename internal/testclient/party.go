package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// The party, friend and block list client packets of M9 (Mobius
// ClientPackets and ExClientPackets).
const (
	OpRequestJoinParty          byte = 0x29
	OpRequestAnswerJoinParty    byte = 0x2a
	OpRequestWithDrawalParty    byte = 0x2b
	OpRequestOustPartyMember    byte = 0x2c
	OpRequestFriendInvite       byte = 0x5e
	OpRequestAnswerFriendInvite byte = 0x5f
	OpRequestFriendList         byte = 0x60
	OpRequestFriendDel          byte = 0x61
	OpRequestPartyMatchConfig   byte = 0x6f
	OpRequestBlock              byte = 0xa0
	OpRequestSendFriendMsg      byte = 0xcc

	ExOpRequestChangePartyLeader uint16 = 0x04
)

// Types of RequestBlock (Mobius RequestBlock constants).
const (
	BlockAdd       int32 = 0 // block a player
	BlockRemove    int32 = 1 // unblock a player
	BlockShowList  int32 = 2 // show the block list
	BlockRefuseAll int32 = 3 // message refusal on
	BlockAcceptAll int32 = 4 // message refusal off
)

// encodeRequestJoinParty builds RequestJoinParty: S name, d loot mode.
func encodeRequestJoinParty(name string, loot int32) []byte {
	w := packets.NewWriter(7 + 2*len(name))
	w.WriteUint8(OpRequestJoinParty)
	w.WriteString(name)
	w.WriteInt32(loot)
	return w.Bytes()
}

// encodeRequestChangePartyLeader builds the Ex packet
// RequestChangePartyLeader: S name.
func encodeRequestChangePartyLeader(name string) []byte {
	w := packets.NewWriter(5 + 2*len(name))
	w.WriteUint8(OpExClient)
	w.WriteUint16(ExOpRequestChangePartyLeader)
	w.WriteString(name)
	return w.Bytes()
}

// encodeRequestBlock builds RequestBlock: d type, and S name only for
// BlockAdd and BlockRemove.
func encodeRequestBlock(typ int32, name string) []byte {
	w := packets.NewWriter(7 + 2*len(name))
	w.WriteUint8(OpRequestBlock)
	w.WriteInt32(typ)
	if typ == BlockAdd || typ == BlockRemove {
		w.WriteString(name)
	}
	return w.Bytes()
}

// encodeRequestSendFriendMsg builds RequestSendFriendMsg: S message,
// S receiver.
func encodeRequestSendFriendMsg(receiver, message string) []byte {
	w := packets.NewWriter(5 + 2*(len(message)+len(receiver)))
	w.WriteUint8(OpRequestSendFriendMsg)
	w.WriteString(message)
	w.WriteString(receiver)
	return w.Bytes()
}

// RequestJoinParty invites the player name into the party with the loot
// mode loot, a Loot constant (RequestJoinParty), without waiting. The
// invited player gets AskJoinParty.
func (c *GameConn) RequestJoinParty(ctx context.Context, name string, loot int32) error {
	return c.send(ctx, encodeRequestJoinParty(name, loot))
}

// InviteToParty invites the player name (RequestJoinParty) and returns the
// JoinParty with the answer. Mobius refuses an invitation with a system
// message, so a refused one ends with the context.
func (c *GameConn) InviteToParty(ctx context.Context, name string, loot int32) (JoinParty, error) {
	if err := c.RequestJoinParty(ctx, name, loot); err != nil {
		return JoinParty{}, err
	}
	return Await[JoinParty](ctx, c, nil)
}

// AwaitPartyInvite waits for the next party invitation.
func (c *GameConn) AwaitPartyInvite(ctx context.Context) (AskJoinParty, error) {
	return Await[AskJoinParty](ctx, c, nil)
}

// AnswerPartyInvite answers the open party invitation with a PartyAnswer
// constant (RequestAnswerJoinParty) without waiting.
func (c *GameConn) AnswerPartyInvite(ctx context.Context, answer int32) error {
	return c.send(ctx, encodeInt(OpRequestAnswerJoinParty, answer))
}

// AcceptPartyInvite accepts the open party invitation and returns the
// PartySmallWindowAll that shows the party, the other members without the
// character.
func (c *GameConn) AcceptPartyInvite(ctx context.Context) (PartySmallWindowAll, error) {
	if err := c.AnswerPartyInvite(ctx, PartyAnswerAccepted); err != nil {
		return PartySmallWindowAll{}, err
	}
	return Await[PartySmallWindowAll](ctx, c, nil)
}

// LeaveParty leaves the party (RequestWithDrawalParty) and returns the
// PartySmallWindowDeleteAll that closes the window. Mobius ignores the
// request outside a party, so such a call ends with the context.
func (c *GameConn) LeaveParty(ctx context.Context) (PartySmallWindowDeleteAll, error) {
	if err := c.send(ctx, encodeOpcode(OpRequestWithDrawalParty)); err != nil {
		return PartySmallWindowDeleteAll{}, err
	}
	return Await[PartySmallWindowDeleteAll](ctx, c, nil)
}

// OustPartyMember dismisses the member name (RequestOustPartyMember)
// without waiting: the other members get PartySmallWindowDelete, or
// PartySmallWindowDeleteAll when the party disperses.
func (c *GameConn) OustPartyMember(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestOustPartyMember, name))
}

// ChangePartyLeader hands the lead to the member name (the Ex packet
// RequestChangePartyLeader) without waiting: every member gets
// PartySmallWindowDeleteAll and a new PartySmallWindowAll.
func (c *GameConn) ChangePartyLeader(ctx context.Context, name string) error {
	return c.send(ctx, encodeRequestChangePartyLeader(name))
}

// PartyMatchConfig opens the party matching window (RequestPartyMatchConfig:
// auto, location, level limit) and returns the ListPartyWating that
// answers. A refusal (a party member who does not lead) comes as a system
// message and ActionFailed, so such a call ends with the context.
func (c *GameConn) PartyMatchConfig(ctx context.Context, auto, location, level int32) (ListPartyWating, error) {
	if err := c.send(ctx, encodeInts(OpRequestPartyMatchConfig, auto, location, level)); err != nil {
		return ListPartyWating{}, err
	}
	return Await[ListPartyWating](ctx, c, nil)
}

// InviteFriend invites the player name into the friend list
// (RequestFriendInvite) without waiting. The invited player gets
// FriendAddRequest.
func (c *GameConn) InviteFriend(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestFriendInvite, name))
}

// AwaitFriendInvite waits for the next friend invitation.
func (c *GameConn) AwaitFriendInvite(ctx context.Context) (FriendAddRequest, error) {
	return Await[FriendAddRequest](ctx, c, nil)
}

// AnswerFriendInvite answers the open friend invitation
// (RequestAnswerFriendInvite: 1 accepts, 0 refuses) without waiting.
func (c *GameConn) AnswerFriendInvite(ctx context.Context, accept bool) error {
	return c.send(ctx, encodeInt(OpRequestAnswerFriendInvite, boolInt32(accept)))
}

// AcceptFriendInvite accepts the open friend invitation and returns the
// L2Friend that adds the inviter to the friend window.
func (c *GameConn) AcceptFriendInvite(ctx context.Context) (L2Friend, error) {
	if err := c.AnswerFriendInvite(ctx, true); err != nil {
		return L2Friend{}, err
	}
	return Await(ctx, c, func(f L2Friend) bool { return f.Action == FriendAdded })
}

// RequestFriendList asks for the friend list (/friendlist) without
// waiting; Mobius answers with system messages.
func (c *GameConn) RequestFriendList(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestFriendList))
}

// DeleteFriend removes the friend name from both friend lists
// (RequestFriendDel) without waiting.
func (c *GameConn) DeleteFriend(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestFriendDel, name))
}

// SendFriendMessage sends message to the friend receiver
// (RequestSendFriendMsg) without waiting; the receiver gets L2FriendSay.
func (c *GameConn) SendFriendMessage(ctx context.Context, receiver, message string) error {
	return c.send(ctx, encodeRequestSendFriendMsg(receiver, message))
}

// RequestBlock changes or shows the block list (RequestBlock; typ is a
// Block constant, name is sent for BlockAdd and BlockRemove only) without
// waiting; Mobius answers with system messages.
func (c *GameConn) RequestBlock(ctx context.Context, typ int32, name string) error {
	return c.send(ctx, encodeRequestBlock(typ, name))
}
