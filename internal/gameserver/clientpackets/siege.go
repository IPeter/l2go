package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the siege window packets (Mobius ClientPackets).
const (
	OpRequestSiegeInfo               = 0x47
	OpRequestSiegeAttackerList       = 0xa2
	OpRequestSiegeDefenderList       = 0xa3
	OpRequestJoinSiege               = 0xa4
	OpRequestConfirmSiegeWaitingList = 0xa5
)

// RequestSiegeInfo is a trigger without fields. Mobius reads nothing and
// does nothing with it; the siege window is opened from the castle NPCs.
type RequestSiegeInfo struct{}

// ParseRequestSiegeInfo parses a RequestSiegeInfo body. Like Mobius it
// reads nothing, so it never fails.
func ParseRequestSiegeInfo([]byte) (RequestSiegeInfo, error) {
	return RequestSiegeInfo{}, nil
}

// RequestSiegeAttackerList asks for the attacker tab of a castle's siege
// window (answered with serverpackets.SiegeAttackerList).
type RequestSiegeAttackerList struct {
	CastleID int32
}

// ParseRequestSiegeAttackerList parses a RequestSiegeAttackerList body
// (Mobius RequestSiegeAttackerList.readImpl): d castle.
func ParseRequestSiegeAttackerList(data []byte) (RequestSiegeAttackerList, error) {
	id, err := readClanInt32("RequestSiegeAttackerList", data)
	if err != nil {
		return RequestSiegeAttackerList{}, err
	}
	return RequestSiegeAttackerList{CastleID: id}, nil
}

// RequestSiegeDefenderList asks for the defender tab of a castle's siege
// window (answered with serverpackets.SiegeDefenderList).
type RequestSiegeDefenderList struct {
	CastleID int32
}

// ParseRequestSiegeDefenderList parses a RequestSiegeDefenderList body
// (Mobius RequestSiegeDefenderList.readImpl): d castle.
func ParseRequestSiegeDefenderList(data []byte) (RequestSiegeDefenderList, error) {
	id, err := readClanInt32("RequestSiegeDefenderList", data)
	if err != nil {
		return RequestSiegeDefenderList{}, err
	}
	return RequestSiegeDefenderList{CastleID: id}, nil
}

// RequestJoinSiege registers the player's clan for a siege, or withdraws
// the registration.
type RequestJoinSiege struct {
	CastleID int32
	// IsAttacker is 1 for the attacker side; Mobius takes any other value
	// as the defender side.
	IsAttacker int32
	// IsJoining is 1 to register; Mobius takes any other value as a
	// withdrawal.
	IsJoining int32
}

// ParseRequestJoinSiege parses a RequestJoinSiege body (Mobius
// RequestJoinSiege.readImpl): d castle, d attacker, d joining.
func ParseRequestJoinSiege(data []byte) (RequestJoinSiege, error) {
	r := packets.NewReader(data)
	p := RequestJoinSiege{CastleID: r.ReadInt32(), IsAttacker: r.ReadInt32(), IsJoining: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestJoinSiege{}, fmt.Errorf("RequestJoinSiege: %w", err)
	}
	return p, nil
}

// RequestConfirmSiegeWaitingList is the castle owner's leader approving a
// waiting defender clan or refusing a defender clan.
type RequestConfirmSiegeWaitingList struct {
	CastleID int32
	ClanID   int32
	// Approved is 1 to approve a waiting clan; Mobius takes any other value
	// as a refusal that removes the clan (waiting or approved) from the
	// siege.
	Approved int32
}

// ParseRequestConfirmSiegeWaitingList parses a
// RequestConfirmSiegeWaitingList body (Mobius
// RequestConfirmSiegeWaitingList.readImpl): d castle, d clan, d approved.
func ParseRequestConfirmSiegeWaitingList(data []byte) (RequestConfirmSiegeWaitingList, error) {
	r := packets.NewReader(data)
	p := RequestConfirmSiegeWaitingList{CastleID: r.ReadInt32(), ClanID: r.ReadInt32(), Approved: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestConfirmSiegeWaitingList{}, fmt.Errorf("RequestConfirmSiegeWaitingList: %w", err)
	}
	return p, nil
}
