package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSystemMessage is the opcode of SystemMessage.
const OpSystemMessage = 0x64

// SystemMessageID is the ID of a message in the client's systemmsg-e.dat
// (Mobius SystemMessageId).
type SystemMessageID int32

// System messages L2Go sends. The comment is the client text; $s1, $s2 and
// so on are the parameters.
const (
	// SysMsgWelcome: "Welcome to the World of Lineage II." Sent at the end
	// of EnterWorld.
	SysMsgWelcome SystemMessageID = 34
	// SysMsgCannotExitInCombat: "You cannot exit while in combat."
	SysMsgCannotExitInCombat SystemMessageID = 101
	// SysMsgCannotRestartInCombat: "You cannot restart while in combat."
	SysMsgCannotRestartInCombat SystemMessageID = 102
	// SysMsgPlayerNotOnline: "That player is not online." A whisper to a
	// player who is not in the world.
	SysMsgPlayerNotOnline SystemMessageID = 145
	// SysMsgChattingProhibited: "Chatting is currently prohibited."
	SysMsgChattingProhibited SystemMessageID = 147
	// SysMsgMessageRefusal: "That person is in message refusal mode." A
	// whisper to a player who blocks the sender or is chat banned.
	SysMsgMessageRefusal SystemMessageID = 176
	// SysMsgFriendLoggedIn: "$s1 (Friend) has logged in."
	SysMsgFriendLoggedIn SystemMessageID = 503
	// SysMsgCurrentLocationTalkingIsland: "Current location : $s1, $s2, $s3
	// (Near Talking Island Village)". The /loc answers are one message per
	// map region (910-926 and others); the map region data gives the ID.
	SysMsgCurrentLocationTalkingIsland SystemMessageID = 910
	// SysMsgChatTooLong: "When a user's keyboard input exceeds a certain
	// cumulative score a chat ban will be applied. ..." Mobius sends it for
	// chat longer than 105 characters.
	SysMsgChatTooLong SystemMessageID = 1078
	// SysMsgText: "$s1". Mobius Player.sendMessage sends free text with it,
	// for example the answers to GM commands.
	SysMsgText SystemMessageID = 1983
)

// SystemMessageParamType is the type code of a SystemMessage parameter
// (Mobius SystemMessage.TYPE_*).
type SystemMessageParamType int32

// Parameter types. Type 8 is unused in Interlude and 14 is unknown.
const (
	ParamText         SystemMessageParamType = 0  // a string
	ParamNumber       SystemMessageParamType = 1  // an int
	ParamNPCName      SystemMessageParamType = 2  // NPC ID + 1,000,000; the client shows the name
	ParamItemName     SystemMessageParamType = 3  // item ID
	ParamSkillName    SystemMessageParamType = 4  // skill ID and level
	ParamCastleName   SystemMessageParamType = 5  // castle or clan hall ID
	ParamLongNumber   SystemMessageParamType = 6  // a long
	ParamZoneName     SystemMessageParamType = 7  // x, y, z; the client shows the zone name
	ParamElementName  SystemMessageParamType = 9  // element ID
	ParamInstanceName SystemMessageParamType = 10 // instance zone ID
	ParamDoorName     SystemMessageParamType = 11 // door ID
	ParamPlayerName   SystemMessageParamType = 12 // a string
	ParamSystemString SystemMessageParamType = 13 // sysstring-e.dat ID
)

// clientNpcIDBase is added to NPC template IDs in packets: the client
// numbers NPCs from 1,000,000.
const clientNpcIDBase = 1_000_000

// SystemMessageParam is one parameter of a SystemMessage. Use the
// constructors below; which fields are sent depends on Type.
type SystemMessageParam struct {
	Type    SystemMessageParamType
	Text    string // ParamText, ParamPlayerName
	Value   int64  // the number or ID; for ParamSkillName the skill ID
	Level   int32  // ParamSkillName: the skill level
	X, Y, Z int32  // ParamZoneName
}

// TextParam is a string parameter (Mobius addString).
func TextParam(s string) SystemMessageParam {
	return SystemMessageParam{Type: ParamText, Text: s}
}

// PlayerNameParam is a player name parameter. Mobius sends player names
// with TextParam (addPcName); this type exists for completeness.
func PlayerNameParam(name string) SystemMessageParam {
	return SystemMessageParam{Type: ParamPlayerName, Text: name}
}

// NumberParam is an int parameter (Mobius addInt).
func NumberParam(n int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamNumber, Value: int64(n)}
}

// LongNumberParam is a long parameter (Mobius addLong).
func LongNumberParam(n int64) SystemMessageParam {
	return SystemMessageParam{Type: ParamLongNumber, Value: n}
}

// NPCNameParam is the name of an NPC template (Mobius addNpcName(int)). It
// adds the 1,000,000 offset the client expects.
func NPCNameParam(npcID int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamNPCName, Value: int64(npcID) + clientNpcIDBase}
}

// ItemNameParam is the name of an item template (Mobius addItemName).
func ItemNameParam(itemID int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamItemName, Value: int64(itemID)}
}

// SkillNameParam is the name of a skill at a level (Mobius addSkillName).
func SkillNameParam(skillID, level int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamSkillName, Value: int64(skillID), Level: level}
}

// ZoneNameParam is the name of the zone at a location (Mobius addZoneName).
func ZoneNameParam(x, y, z int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamZoneName, X: x, Y: y, Z: z}
}

// CastleNameParam is the name of a castle or clan hall (Mobius
// addCastleId).
func CastleNameParam(id int32) SystemMessageParam {
	return SystemMessageParam{Type: ParamCastleName, Value: int64(id)}
}

// SystemMessage is a message from the client's message table, with its
// parameters in order.
type SystemMessage struct {
	ID     SystemMessageID
	Params []SystemMessageParam
}

// NewSystemMessagePacket builds SystemMessage (Mobius
// SystemMessage.writeImpl, without the multilingual mode).
//
// Mobius writes the number of parameters its message table declares and
// skips missing ones; this builder writes the parameters it is given, so
// give each message all of its parameters.
func NewSystemMessagePacket(m SystemMessage) []byte {
	w := packets.NewWriter(9 + len(m.Params)*16)
	w.WriteUint8(OpSystemMessage)
	w.WriteInt32(int32(m.ID))
	putCount32(w, len(m.Params))
	for _, p := range m.Params {
		w.WriteInt32(int32(p.Type))
		switch p.Type {
		case ParamText, ParamPlayerName:
			w.WriteString(p.Text)
		case ParamLongNumber:
			putInt64(w, p.Value)
		case ParamItemName, ParamCastleName, ParamNumber, ParamNPCName,
			ParamElementName, ParamSystemString, ParamInstanceName, ParamDoorName:
			w.WriteInt32(int32(p.Value)) //nolint:gosec // an int parameter: the low 32 bits, as Java writes it
		case ParamSkillName:
			w.WriteInt32(int32(p.Value)) //nolint:gosec // a skill ID
			w.WriteInt32(p.Level)
		case ParamZoneName:
			w.WriteInt32(p.X)
			w.WriteInt32(p.Y)
			w.WriteInt32(p.Z)
		}
		// Other types carry no value, as in Mobius.
	}
	return w.Bytes()
}

// NewSystemMessageTextPacket builds the SystemMessage Mobius sends for
// free text (Player.sendMessage): SysMsgText with one text parameter.
func NewSystemMessageTextPacket(text string) []byte {
	return NewSystemMessagePacket(SystemMessage{ID: SysMsgText, Params: []SystemMessageParam{TextParam(text)}})
}
