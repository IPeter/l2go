package testclient

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Game client opcodes of the world, from Mobius ClientPackets. The client
// MoveToLocation is OpMoveToLocation; the server one is
// OpCharMoveToLocation.
const (
	OpMoveToLocation            byte = 0x01
	OpEnterWorld                byte = 0x03
	OpAction                    byte = 0x04
	OpLogout                    byte = 0x09
	OpAttackRequest             byte = 0x0a
	OpCharacterDelete           byte = 0x0c
	OpCharacterSelect           byte = 0x0d
	OpRequestItemList           byte = 0x0f
	OpRequestUnEquipItem        byte = 0x11
	OpRequestDropItem           byte = 0x12
	OpUseItem                   byte = 0x14
	OpRequestSocialAction       byte = 0x1b
	OpChangeMoveType2           byte = 0x1c
	OpChangeWaitType2           byte = 0x1d
	OpRequestSellItem           byte = 0x1e
	OpRequestBuyItem            byte = 0x1f
	OpRequestLinkHtml           byte = 0x20
	OpRequestBypassToServer     byte = 0x21
	OpRequestMagicSkillUse      byte = 0x2f
	OpAppearing                 byte = 0x30
	OpSendWareHouseDepositList  byte = 0x31
	OpSendWareHouseWithDrawList byte = 0x32
	OpRequestShortcutReg        byte = 0x33
	OpRequestShortcutDel        byte = 0x35
	OpCannotMoveAnymore         byte = 0x36
	OpRequestTargetCanceld      byte = 0x37
	OpSay2                      byte = 0x38
	OpRequestSkillList          byte = 0x3f
	OpRequestActionUse          byte = 0x45
	OpRequestRestart            byte = 0x46
	OpValidatePosition          byte = 0x48
	OpStartRotating             byte = 0x4a
	OpFinishRotating            byte = 0x4b
	OpRequestDestroyItem        byte = 0x59
	OpSendBypassBuildCmd        byte = 0x5b
	OpCharacterRestore          byte = 0x62
	OpRequestQuestList          byte = 0x63
	OpRequestAcquireSkillInfo   byte = 0x6b
	OpRequestAcquireSkill       byte = 0x6c
	OpRequestRestartPoint       byte = 0x6d
	OpRequestSkillCoolTime      byte = 0x9d
	OpMultiSellChoose           byte = 0xa7
	OpBypassUserCmd             byte = 0xaa
	OpRequestShowMiniMap        byte = 0xcd
	// OpExClient starts every Ex client packet. A little-endian uint16
	// sub-opcode follows it (Mobius GamePacketHandler).
	OpExClient byte = 0xd0
)

// Ex client packet sub-opcodes.
const (
	ExOpRequestAutoSoulShot uint16 = 0x05
	ExOpRequestManorList    uint16 = 0x08
)

// clientPacketNames maps client opcodes to Mobius packet names, for the
// recorder.
//
//nolint:gosec // packet names, not credentials
var clientPacketNames = map[byte]string{
	OpProtocolVersion:           "ProtocolVersion",
	OpMoveToLocation:            "MoveToLocation",
	OpEnterWorld:                "EnterWorld",
	OpAction:                    "Action",
	OpAuthLogin:                 "AuthLogin",
	OpLogout:                    "Logout",
	OpAttackRequest:             "AttackRequest",
	OpCharacterCreate:           "CharacterCreate",
	OpCharacterDelete:           "CharacterDelete",
	OpCharacterSelect:           "CharacterSelect",
	OpNewCharacter:              "NewCharacter",
	OpRequestItemList:           "RequestItemList",
	OpRequestUnEquipItem:        "RequestUnEquipItem",
	OpRequestDropItem:           "RequestDropItem",
	OpUseItem:                   "UseItem",
	OpRequestSocialAction:       "RequestSocialAction",
	OpChangeMoveType2:           "ChangeMoveType2",
	OpChangeWaitType2:           "ChangeWaitType2",
	OpRequestSellItem:           "RequestSellItem",
	OpRequestBuyItem:            "RequestBuyItem",
	OpRequestLinkHtml:           "RequestLinkHtml",
	OpRequestBypassToServer:     "RequestBypassToServer",
	OpRequestMagicSkillUse:      "RequestMagicSkillUse",
	OpAppearing:                 "Appearing",
	OpSendWareHouseDepositList:  "SendWareHouseDepositList",
	OpSendWareHouseWithDrawList: "SendWareHouseWithDrawList",
	OpRequestShortcutReg:        "RequestShortcutReg",
	OpRequestShortcutDel:        "RequestShortcutDel",
	OpCannotMoveAnymore:         "CannotMoveAnymore",
	OpRequestTargetCanceld:      "RequestTargetCanceld",
	OpSay2:                      "Say2",
	OpRequestSkillList:          "RequestSkillList",
	OpRequestActionUse:          "RequestActionUse",
	OpRequestRestart:            "RequestRestart",
	OpValidatePosition:          "ValidatePosition",
	OpStartRotating:             "StartRotating",
	OpFinishRotating:            "FinishRotating",
	OpRequestDestroyItem:        "RequestDestroyItem",
	OpSendBypassBuildCmd:        "SendBypassBuildCmd",
	OpCharacterRestore:          "CharacterRestore",
	OpRequestQuestList:          "RequestQuestList",
	OpRequestAcquireSkillInfo:   "RequestAcquireSkillInfo",
	OpRequestAcquireSkill:       "RequestAcquireSkill",
	OpRequestRestartPoint:       "RequestRestartPoint",
	OpRequestSkillCoolTime:      "RequestSkillCoolTime",
	OpMultiSellChoose:           "MultiSellChoose",
	OpBypassUserCmd:             "BypassUserCmd",
	OpRequestShowMiniMap:        "RequestShowMiniMap",
}

// exClientPacketNames maps Ex client sub-opcodes to Mobius packet names.
var exClientPacketNames = map[uint16]string{
	ExOpRequestAutoSoulShot: "RequestAutoSoulShot",
	ExOpRequestManorList:    "RequestManorList",
}

// clientPacketName returns the Mobius name of a raw client packet (opcode
// first), or "Unknown(0x..)".
func clientPacketName(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == OpExClient && len(raw) >= 3 {
		sub := uint16(raw[1]) | uint16(raw[2])<<8
		if name, ok := exClientPacketNames[sub]; ok {
			return name
		}
		return fmt.Sprintf("UnknownEx(0x%04x)", sub)
	}
	if name, ok := clientPacketNames[raw[0]]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(0x%02x)", raw[0])
}

// ChatType is the chat channel of Say2, CreatureSay and NpcSay (Mobius
// ChatType client IDs).
type ChatType int32

// Chat channels.
const (
	ChatGeneral            ChatType = 0
	ChatShout              ChatType = 1
	ChatWhisper            ChatType = 2
	ChatParty              ChatType = 3
	ChatClan               ChatType = 4
	ChatGM                 ChatType = 5
	ChatPetitionPlayer     ChatType = 6
	ChatPetitionGM         ChatType = 7
	ChatTrade              ChatType = 8
	ChatAlliance           ChatType = 9
	ChatAnnouncement       ChatType = 10
	ChatBoat               ChatType = 11
	ChatFriend             ChatType = 12
	ChatMSN                ChatType = 13
	ChatPartyMatchRoom     ChatType = 14
	ChatPartyRoomCommander ChatType = 15
	ChatPartyRoomAll       ChatType = 16
	ChatHeroVoice          ChatType = 17
	ChatCriticalAnnounce   ChatType = 18
	ChatScreenAnnounce     ChatType = 19
	ChatBattlefield        ChatType = 20
	ChatMPCCRoom           ChatType = 21
)

// Action IDs of RequestActionUse.
const (
	ActionSitStand int32 = 0
	ActionRunWalk  int32 = 1
)

// User command IDs of BypassUserCmd.
const (
	// UserCmdLoc is /loc.
	UserCmdLoc int32 = 0
)

// MoveRequest holds the fields of the client MoveToLocation.
type MoveRequest struct {
	Target Position
	// Origin is where the client thinks the character stands.
	Origin Position
	// Keyboard sends movement mode 0 (arrow keys) instead of 1 (mouse).
	Keyboard bool
}

// enterWorldHop is the address EnterWorld reports for each of the five
// trace route hops.
var enterWorldHop = [4]byte{127, 0, 0, 1}

// boolInt32 and boolByte write a Java boolean.
func boolInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func boolByte(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

// encodeOpcode builds a packet without fields.
func encodeOpcode(op byte) []byte { return []byte{op} }

// encodeCharacterSelect builds CharacterSelect in Mobius readImpl order:
// d slot, then h and three d that Mobius reads and ignores.
func encodeCharacterSelect(slot int32) []byte {
	w := packets.NewWriter(19)
	w.WriteUint8(OpCharacterSelect)
	w.WriteInt32(slot)
	w.WriteUint16(0)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteInt32(0)
	return w.Bytes()
}

// encodeCharacterDelete builds CharacterDelete: d slot.
func encodeCharacterDelete(slot int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpCharacterDelete)
	w.WriteInt32(slot)
	return w.Bytes()
}

// encodeCharacterRestore builds CharacterRestore: d slot.
func encodeCharacterRestore(slot int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpCharacterRestore)
	w.WriteInt32(slot)
	return w.Bytes()
}

// encodeEnterWorld builds EnterWorld as Mobius EnterWorld.readImpl reads
// it: 32 bytes, four d, 32 bytes, one d (all unknown, sent as zero), then
// five trace route hops of four bytes.
func encodeEnterWorld() []byte {
	w := packets.NewWriter(105)
	w.WriteUint8(OpEnterWorld)
	w.WriteBytes(make([]byte, 32))
	for range 4 {
		w.WriteInt32(0)
	}
	w.WriteBytes(make([]byte, 32))
	w.WriteInt32(0)
	for range 5 {
		w.WriteBytes(enterWorldHop[:])
	}
	return w.Bytes()
}

// encodeMoveToLocation builds the client MoveToLocation: the target, the
// origin and the movement mode (1 mouse, 0 arrow keys).
func encodeMoveToLocation(req MoveRequest) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpMoveToLocation)
	writePosition(w, req.Target)
	writePosition(w, req.Origin)
	w.WriteInt32(boolInt32(!req.Keyboard))
	return w.Bytes()
}

// encodeValidatePosition builds ValidatePosition: x, y, z, heading and the
// vehicle ID.
func encodeValidatePosition(pos Position, heading, vehicleID int32) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpValidatePosition)
	writePosition(w, pos)
	w.WriteInt32(heading)
	w.WriteInt32(vehicleID)
	return w.Bytes()
}

// encodeCannotMoveAnymore builds CannotMoveAnymore: x, y, z, heading.
func encodeCannotMoveAnymore(pos Position, heading int32) []byte {
	w := packets.NewWriter(17)
	w.WriteUint8(OpCannotMoveAnymore)
	writePosition(w, pos)
	w.WriteInt32(heading)
	return w.Bytes()
}

// encodeRequestActionUse builds RequestActionUse: d action, d ctrl, c
// shift.
func encodeRequestActionUse(actionID int32, ctrl, shift bool) []byte {
	w := packets.NewWriter(10)
	w.WriteUint8(OpRequestActionUse)
	w.WriteInt32(actionID)
	w.WriteInt32(boolInt32(ctrl))
	w.WriteUint8(boolByte(shift))
	return w.Bytes()
}

// encodeChangeMoveType2 builds ChangeMoveType2: d 1 to run, 0 to walk.
func encodeChangeMoveType2(run bool) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpChangeMoveType2)
	w.WriteInt32(boolInt32(run))
	return w.Bytes()
}

// encodeChangeWaitType2 builds ChangeWaitType2: d 1 to stand, 0 to sit.
func encodeChangeWaitType2(stand bool) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpChangeWaitType2)
	w.WriteInt32(boolInt32(stand))
	return w.Bytes()
}

// encodeStartRotating builds StartRotating: d degree, d side.
func encodeStartRotating(degree, side int32) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpStartRotating)
	w.WriteInt32(degree)
	w.WriteInt32(side)
	return w.Bytes()
}

// encodeFinishRotating builds FinishRotating: d degree and a word Mobius
// reads and ignores.
func encodeFinishRotating(degree int32) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpFinishRotating)
	w.WriteInt32(degree)
	w.WriteInt32(0)
	return w.Bytes()
}

// encodeSay2 builds Say2: S text, d type, and S target for a whisper only.
func encodeSay2(text string, typ ChatType, target string) []byte {
	w := packets.NewWriter(16 + 2*len(text) + 2*len(target))
	w.WriteUint8(OpSay2)
	w.WriteString(text)
	w.WriteInt32(int32(typ))
	if typ == ChatWhisper {
		w.WriteString(target)
	}
	return w.Bytes()
}

// encodeString builds a packet with one string field: SendBypassBuildCmd,
// RequestBypassToServer and RequestLinkHtml.
func encodeString(op byte, s string) []byte {
	w := packets.NewWriter(3 + 2*len(s))
	w.WriteUint8(op)
	w.WriteString(s)
	return w.Bytes()
}

// encodeInt builds a packet with one d field: BypassUserCmd and
// RequestSocialAction.
func encodeInt(op byte, v int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(op)
	w.WriteInt32(v)
	return w.Bytes()
}

// encodeAction builds Action, and AttackRequest with OpAttackRequest: d
// object ID, d x, d y, d z (the client's position), c 1 for a shift click.
func encodeAction(op byte, objectID int32, origin Position, shift bool) []byte {
	w := packets.NewWriter(18)
	w.WriteUint8(op)
	w.WriteInt32(objectID)
	writePosition(w, origin)
	w.WriteUint8(boolByte(shift))
	return w.Bytes()
}

// encodeRequestTargetCanceld builds RequestTargetCanceld: h 0 cancels a
// cast before the target, 1 only drops the target.
func encodeRequestTargetCanceld(unselect uint16) []byte {
	w := packets.NewWriter(3)
	w.WriteUint8(OpRequestTargetCanceld)
	w.WriteUint16(unselect)
	return w.Bytes()
}

// encodeEx builds an Ex client packet without fields.
func encodeEx(sub uint16) []byte {
	w := packets.NewWriter(3)
	w.WriteUint8(OpExClient)
	w.WriteUint16(sub)
	return w.Bytes()
}

func writePosition(w *packets.Writer, p Position) {
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
}
