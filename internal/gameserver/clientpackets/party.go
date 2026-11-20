package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the party packets (Mobius ClientPackets).
const (
	OpRequestJoinParty        = 0x29
	OpRequestAnswerJoinParty  = 0x2a
	OpRequestWithDrawalParty  = 0x2b
	OpRequestOustPartyMember  = 0x2c
	OpRequestPartyMatchConfig = 0x6f
)

// ExOpRequestChangePartyLeader is the sub-opcode of the extended packet
// RequestChangePartyLeader (Mobius ExClientPackets).
const ExOpRequestChangePartyLeader = 0x04

// The answers of RequestAnswerJoinParty.Response.
const (
	PartyAnswerRefusalMode = -1 // the client refuses all invitations
	PartyAnswerDeclined    = 0
	PartyAnswerAccepted    = 1
)

// RequestJoinParty invites a player into the party. Mobius answers with
// AskJoinParty to the invited player.
type RequestJoinParty struct {
	Name string
	// Loot is the loot mode the inviter chose, a Mobius
	// PartyDistributionType ID (serverpackets.PartyLoot), not checked here.
	// It applies when the invitation creates the party.
	Loot int32
}

// ParseRequestJoinParty parses a RequestJoinParty body (Mobius
// RequestJoinParty.readImpl): S name, d loot mode.
func ParseRequestJoinParty(data []byte) (RequestJoinParty, error) {
	r := packets.NewReader(data)
	req := RequestJoinParty{
		Name: r.ReadString(),
		Loot: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestJoinParty{}, fmt.Errorf("RequestJoinParty: %w", err)
	}
	return req, nil
}

// RequestAnswerJoinParty is the invited player's answer to AskJoinParty.
type RequestAnswerJoinParty struct {
	// Response is one of the PartyAnswer constants, not checked here.
	// Mobius passes any value back to the inviter in JoinParty.
	Response int32
}

// ParseRequestAnswerJoinParty parses a RequestAnswerJoinParty body (Mobius
// RequestAnswerJoinParty.readImpl): d response.
func ParseRequestAnswerJoinParty(data []byte) (RequestAnswerJoinParty, error) {
	r := packets.NewReader(data)
	req := RequestAnswerJoinParty{Response: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestAnswerJoinParty{}, fmt.Errorf("RequestAnswerJoinParty: %w", err)
	}
	return req, nil
}

// RequestWithDrawalParty leaves the party. It has no fields.
type RequestWithDrawalParty struct{}

// ParseRequestWithDrawalParty parses a RequestWithDrawalParty body. Like
// Mobius it reads nothing, so it never fails.
func ParseRequestWithDrawalParty([]byte) (RequestWithDrawalParty, error) {
	return RequestWithDrawalParty{}, nil
}

// RequestOustPartyMember dismisses a member from the party (leader only).
type RequestOustPartyMember struct {
	Name string
}

// ParseRequestOustPartyMember parses a RequestOustPartyMember body (Mobius
// RequestOustPartyMember.readImpl): S name.
func ParseRequestOustPartyMember(data []byte) (RequestOustPartyMember, error) {
	name, err := partyFriendName(data)
	if err != nil {
		return RequestOustPartyMember{}, fmt.Errorf("RequestOustPartyMember: %w", err)
	}
	return RequestOustPartyMember{Name: name}, nil
}

// RequestChangePartyLeader hands the party lead to a member. It is the
// extended packet ExOpRequestChangePartyLeader.
type RequestChangePartyLeader struct {
	Name string
}

// ParseRequestChangePartyLeader parses a RequestChangePartyLeader body, the
// ExPacket.Body after the sub-opcode (Mobius
// RequestChangePartyLeader.readImpl): S name.
func ParseRequestChangePartyLeader(data []byte) (RequestChangePartyLeader, error) {
	name, err := partyFriendName(data)
	if err != nil {
		return RequestChangePartyLeader{}, fmt.Errorf("RequestChangePartyLeader: %w", err)
	}
	return RequestChangePartyLeader{Name: name}, nil
}

// RequestPartyMatchConfig opens the party matching window. In M9 the
// server answers with an empty ListPartyWating (Plan 12 T10).
type RequestPartyMatchConfig struct {
	Auto     int32
	Location int32 // the region filter; 0 or less for all
	// Level is what Mobius names "my level" in readImpl but passes to
	// ListPartyWating as its limit: 0 shows only the rooms that fit the
	// player's level.
	Level int32
}

// ParseRequestPartyMatchConfig parses a RequestPartyMatchConfig body
// (Mobius RequestPartyMatchConfig.readImpl): d auto, d location, d level.
func ParseRequestPartyMatchConfig(data []byte) (RequestPartyMatchConfig, error) {
	r := packets.NewReader(data)
	req := RequestPartyMatchConfig{
		Auto:     r.ReadInt32(),
		Location: r.ReadInt32(),
		Level:    r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestPartyMatchConfig{}, fmt.Errorf("RequestPartyMatchConfig: %w", err)
	}
	return req, nil
}

// partyFriendName reads a body that is one string, a character name.
func partyFriendName(data []byte) (string, error) {
	r := packets.NewReader(data)
	name := r.ReadString()
	if err := r.Err(); err != nil {
		return "", err
	}
	return name, nil
}
