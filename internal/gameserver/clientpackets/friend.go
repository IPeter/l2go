package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the friend and block list packets (Mobius ClientPackets).
const (
	OpRequestFriendInvite       = 0x5e
	OpRequestAnswerFriendInvite = 0x5f
	OpRequestFriendList         = 0x60
	OpRequestFriendDel          = 0x61
	OpRequestBlock              = 0xa0
	OpRequestSendFriendMsg      = 0xcc
)

// The values of RequestBlock.Type (Mobius RequestBlock constants).
const (
	BlockAdd       = 0 // block a player (BLOCK)
	BlockRemove    = 1 // unblock a player (UNBLOCK)
	BlockShowList  = 2 // show the block list (BLOCKLIST)
	BlockRefuseAll = 3 // message refusal on (ALLBLOCK)
	BlockAcceptAll = 4 // message refusal off (ALLUNBLOCK)
)

// friendAnswerAccept is the RequestAnswerFriendInvite response that
// accepts.
const friendAnswerAccept = 1

// RequestFriendInvite invites a player into the friend list. Mobius answers
// with FriendAddRequest to the invited player.
type RequestFriendInvite struct {
	Name string
}

// ParseRequestFriendInvite parses a RequestFriendInvite body (Mobius
// RequestFriendInvite.readImpl): S name.
func ParseRequestFriendInvite(data []byte) (RequestFriendInvite, error) {
	name, err := partyFriendName(data)
	if err != nil {
		return RequestFriendInvite{}, fmt.Errorf("RequestFriendInvite: %w", err)
	}
	return RequestFriendInvite{Name: name}, nil
}

// RequestAnswerFriendInvite is the invited player's answer to
// FriendAddRequest.
type RequestAnswerFriendInvite struct {
	// Response 1 accepts; Mobius takes every other value as a refusal.
	Response int32
}

// Accepted reports whether the answer accepts the invitation.
func (r RequestAnswerFriendInvite) Accepted() bool { return r.Response == friendAnswerAccept }

// ParseRequestAnswerFriendInvite parses a RequestAnswerFriendInvite body
// (Mobius RequestAnswerFriendInvite.readImpl): d response.
func ParseRequestAnswerFriendInvite(data []byte) (RequestAnswerFriendInvite, error) {
	r := packets.NewReader(data)
	req := RequestAnswerFriendInvite{Response: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestAnswerFriendInvite{}, fmt.Errorf("RequestAnswerFriendInvite: %w", err)
	}
	return req, nil
}

// RequestFriendList is /friendlist; Mobius answers with system messages.
// It has no fields.
type RequestFriendList struct{}

// ParseRequestFriendList parses a RequestFriendList body. Like Mobius it
// reads nothing, so it never fails.
func ParseRequestFriendList([]byte) (RequestFriendList, error) { return RequestFriendList{}, nil }

// RequestFriendDel removes a friend, from both lists.
type RequestFriendDel struct {
	Name string
}

// ParseRequestFriendDel parses a RequestFriendDel body (Mobius
// RequestFriendDel.readImpl): S name.
func ParseRequestFriendDel(data []byte) (RequestFriendDel, error) {
	name, err := partyFriendName(data)
	if err != nil {
		return RequestFriendDel{}, fmt.Errorf("RequestFriendDel: %w", err)
	}
	return RequestFriendDel{Name: name}, nil
}

// RequestBlock changes or shows the block list.
type RequestBlock struct {
	// Type is one of the Block constants, not checked here: Mobius ignores
	// other values.
	Type int32
	// Name is the player to block or unblock; only BlockAdd and BlockRemove
	// carry it.
	Name string
}

// ParseRequestBlock parses a RequestBlock body (Mobius
// RequestBlock.readImpl): d type, then S name for BlockAdd and BlockRemove
// only.
func ParseRequestBlock(data []byte) (RequestBlock, error) {
	r := packets.NewReader(data)
	req := RequestBlock{Type: r.ReadInt32()}
	if req.Type == BlockAdd || req.Type == BlockRemove {
		req.Name = r.ReadString()
	}
	if err := r.Err(); err != nil {
		return RequestBlock{}, fmt.Errorf("RequestBlock: %w", err)
	}
	return req, nil
}

// RequestSendFriendMsg sends a message to a friend through the friend
// window. Mobius checks the length (1 to 300 characters) and the friend
// lists in runImpl, so they are not checked here.
type RequestSendFriendMsg struct {
	Message  string
	Receiver string
}

// ParseRequestSendFriendMsg parses a RequestSendFriendMsg body (Mobius
// RequestSendFriendMsg.readImpl): S message, S receiver.
func ParseRequestSendFriendMsg(data []byte) (RequestSendFriendMsg, error) {
	r := packets.NewReader(data)
	req := RequestSendFriendMsg{
		Message:  r.ReadString(),
		Receiver: r.ReadString(),
	}
	if err := r.Err(); err != nil {
		return RequestSendFriendMsg{}, fmt.Errorf("RequestSendFriendMsg: %w", err)
	}
	return req, nil
}
