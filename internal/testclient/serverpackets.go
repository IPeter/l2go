package testclient

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Packet is a decoded game server packet. Every server packet type of this
// package implements it with a value receiver, so a type switch or the
// generic Await, Expect and Find pick packets by their struct type.
type Packet interface {
	// PacketName returns the Mobius class name of the packet.
	PacketName() string
}

// Game server opcodes of the world, from Mobius ServerPackets. The names
// follow the Mobius enum: OpCharMoveToLocation is the server
// MoveToLocation, OpMoveToLocation the client one.
const (
	OpCharMoveToLocation      byte = 0x01
	OpNpcSay                  byte = 0x02
	OpCharInfo                byte = 0x03
	OpUserInfo                byte = 0x04
	OpAttack                  byte = 0x05
	OpDie                     byte = 0x06
	OpRevive                  byte = 0x07
	OpSpawnItem               byte = 0x0b
	OpDropItem                byte = 0x0c
	OpGetItem                 byte = 0x0d
	OpStatusUpdate            byte = 0x0e
	OpNpcHtmlMessage          byte = 0x0f
	OpSellList                byte = 0x10
	OpBuyList                 byte = 0x11
	OpDeleteObject            byte = 0x12
	OpCharSelected            byte = 0x15
	OpNpcInfo                 byte = 0x16
	OpItemList                byte = 0x1b
	OpCharDeleteSuccess       byte = 0x23
	OpCharDeleteFail          byte = 0x24
	OpActionFailed            byte = 0x25
	OpServerClose             byte = 0x26
	OpInventoryUpdate         byte = 0x27
	OpTeleportToLocation      byte = 0x28
	OpTargetSelected          byte = 0x29
	OpTargetUnselected        byte = 0x2a
	OpAutoAttackStart         byte = 0x2b
	OpAutoAttackStop          byte = 0x2c
	OpSocialAction            byte = 0x2d
	OpChangeMoveType          byte = 0x2e
	OpChangeWaitType          byte = 0x2f
	OpWareHouseDepositList    byte = 0x41
	OpWareHouseWithdrawalList byte = 0x42
	OpShortcutRegister        byte = 0x44
	OpShortcutInit            byte = 0x45
	OpStopMove                byte = 0x47
	OpMagicSkillUse           byte = 0x48
	OpMagicSkillCanceled      byte = 0x49
	OpCreatureSay             byte = 0x4a
	OpEquipUpdate             byte = 0x4b
	OpSkillList               byte = 0x58
	OpRestartResponse         byte = 0x5f
	OpMoveToPawn              byte = 0x60
	OpValidateLocation        byte = 0x61
	OpStartRotation           byte = 0x62
	OpStopRotation            byte = 0x63
	OpSystemMessage           byte = 0x64
	OpSetupGauge              byte = 0x6d
	OpMagicSkillLaunched      byte = 0x76
	OpLeaveWorld              byte = 0x7e
	OpAbnormalStatusUpdate    byte = 0x7f
	OpQuestList               byte = 0x80
	OpEnchantResult           byte = 0x81
	OpAcquireSkillList        byte = 0x8a
	OpAcquireSkillInfo        byte = 0x8b
	OpServerObjectInfo        byte = 0x8c
	OpAcquireSkillDone        byte = 0x8e
	OpShowMiniMap             byte = 0x9d
	OpMyTargetSelected        byte = 0xa6
	OpSkillCoolTime           byte = 0xc1
	OpRelationChanged         byte = 0xce
	OpMultiSellList           byte = 0xd0
	OpHennaInfo               byte = 0xe4
	OpSendMacroList           byte = 0xe7
	OpPartySpelled            byte = 0xee
	OpEtcStatusUpdate         byte = 0xf3
	OpShortBuffStatusUpdate   byte = 0xf4
	OpFriendList              byte = 0xfa
	// OpExServer starts every Ex packet. A little-endian uint16 sub-opcode
	// follows it (Mobius ServerPackets.writeId).
	OpExServer byte = 0xfe
)

// Ex server packet sub-opcodes.
const (
	ExOpAutoSoulShot    uint16 = 0x12
	ExOpStorageMaxCount uint16 = 0x2e
)

// npcTypeOffset is what Mobius adds to an NPC's display ID in NpcInfo,
// ServerObjectInfo and NpcSay.
const npcTypeOffset = 1000000

// serverPacket is a game server packet the client knows by name, and
// parses when decode is set. decode reads the fields after the opcode (and
// the Ex sub-opcode).
type serverPacket struct {
	name   string
	decode func(f *fields) Packet
}

// serverPackets lists the game server packets by opcode. The entries
// without decode give their Mobius name to Unknown packets, so a recorded
// stream stays readable. KeyPacket is decoded by parseKeyPacket.
var serverPackets = map[byte]serverPacket{
	OpKeyPacket:               {name: "KeyPacket"},
	OpCharMoveToLocation:      {"MoveToLocation", readMoveToLocation},
	OpNpcSay:                  {"NpcSay", readNpcSay},
	OpCharInfo:                {"CharInfo", readCharInfoPacket},
	OpUserInfo:                {"UserInfo", readUserInfo},
	OpAttack:                  {"Attack", readAttack},
	OpDie:                     {"Die", readDie},
	OpRevive:                  {"Revive", readRevive},
	OpSpawnItem:               {"SpawnItem", readSpawnItem},
	OpDropItem:                {"DropItem", readDropItem},
	OpGetItem:                 {"GetItem", readGetItem},
	OpStatusUpdate:            {"StatusUpdate", readStatusUpdate},
	OpNpcHtmlMessage:          {"NpcHtmlMessage", readNpcHtmlMessage},
	OpSellList:                {"SellList", readSellList},
	OpBuyList:                 {"BuyList", readBuyList},
	OpDeleteObject:            {"DeleteObject", readDeleteObject},
	OpCharSelectionInfo:       {"CharSelectionInfo", readCharSelectionInfoPacket},
	OpGameLoginFail:           {"LoginFail", readGameLoginFail},
	OpCharSelected:            {"CharSelected", readCharSelected},
	OpNpcInfo:                 {"NpcInfo", readNpcInfo},
	OpNewCharacterSuccess:     {"NewCharacterSuccess", readNewCharacterSuccessPacket},
	OpCharCreateOk:            {"CharCreateOk", readCharCreateOk},
	OpCharCreateFail:          {"CharCreateFail", readCharCreateFail},
	OpItemList:                {"ItemList", readItemList},
	0x1c:                      {name: "SunRise"},
	0x1d:                      {name: "SunSet"},
	OpCharDeleteSuccess:       {"CharDeleteSuccess", readCharDeleteSuccess},
	OpCharDeleteFail:          {"CharDeleteFail", readCharDeleteFail},
	OpActionFailed:            {"ActionFailed", readActionFailed},
	OpServerClose:             {"ServerClose", readServerClose},
	OpInventoryUpdate:         {"InventoryUpdate", readInventoryUpdate},
	OpTeleportToLocation:      {"TeleportToLocation", readTeleportToLocation},
	OpTargetSelected:          {"TargetSelected", readTargetSelected},
	OpTargetUnselected:        {"TargetUnselected", readTargetUnselected},
	OpAutoAttackStart:         {"AutoAttackStart", readAutoAttackStart},
	OpAutoAttackStop:          {"AutoAttackStop", readAutoAttackStop},
	OpSocialAction:            {"SocialAction", readSocialAction},
	OpChangeMoveType:          {"ChangeMoveType", readChangeMoveType},
	OpChangeWaitType:          {"ChangeWaitType", readChangeWaitType},
	OpWareHouseDepositList:    {"WareHouseDepositList", readWareHouseDepositList},
	OpWareHouseWithdrawalList: {"WareHouseWithdrawalList", readWareHouseWithdrawalList},
	OpShortcutRegister:        {"ShortcutRegister", readShortcutRegister},
	OpShortcutInit:            {"ShortcutInit", readShortcutInit},
	OpStopMove:                {"StopMove", readStopMove},
	OpMagicSkillUse:           {"MagicSkillUse", readMagicSkillUse},
	OpMagicSkillCanceled:      {"MagicSkillCanceled", readMagicSkillCanceled},
	OpCreatureSay:             {"CreatureSay", readCreatureSay},
	OpEquipUpdate:             {"EquipUpdate", readEquipUpdate},
	0x4c:                      {name: "DoorInfo"},
	0x4d:                      {name: "DoorStatusUpdate"},
	OpSkillList:               {"SkillList", readSkillList},
	OpRestartResponse:         {"RestartResponse", readRestartResponse},
	OpMoveToPawn:              {"MoveToPawn", readMoveToPawn},
	OpValidateLocation:        {"ValidateLocation", readValidateLocation},
	OpStartRotation:           {"StartRotation", readStartRotation},
	OpStopRotation:            {"StopRotation", readStopRotation},
	OpSystemMessage:           {"SystemMessage", readSystemMessage},
	OpSetupGauge:              {"SetupGauge", readSetupGauge},
	OpMagicSkillLaunched:      {"MagicSkillLaunched", readMagicSkillLaunched},
	OpLeaveWorld:              {"LeaveWorld", readLeaveWorld},
	OpAbnormalStatusUpdate:    {"AbnormalStatusUpdate", readAbnormalStatusUpdate},
	OpQuestList:               {"QuestList", readQuestList},
	OpEnchantResult:           {"EnchantResult", readEnchantResult},
	0x83:                      {name: "PledgeInfo"},
	OpAcquireSkillList:        {"AcquireSkillList", readAcquireSkillList},
	OpAcquireSkillInfo:        {"AcquireSkillInfo", readAcquireSkillInfo},
	OpServerObjectInfo:        {"ServerObjectInfo", readServerObjectInfo},
	OpAcquireSkillDone:        {"AcquireSkillDone", readAcquireSkillDone},
	OpPlaySound:               {"PlaySound", readPlaySound},
	0x99:                      {name: "StaticObjectInfo"},
	OpShowMiniMap:             {"ShowMiniMap", readShowMiniMap},
	0xa0:                      {"TutorialShowHtml", readTutorialShowHtml},
	0xa1:                      {"TutorialShowQuestionMark", readTutorialShowQuestionMark},
	0xa2:                      {"TutorialEnableClientEvent", readTutorialEnableClientEvent},
	0xa3:                      {"TutorialCloseHtml", readTutorialCloseHtml},
	OpMyTargetSelected:        {"MyTargetSelected", readMyTargetSelected},
	0xb1:                      {name: "PetInfo"},
	OpSkillCoolTime:           {"SkillCoolTime", readSkillCoolTime},
	OpRelationChanged:         {"RelationChanged", readRelationChanged},
	OpMultiSellList:           {"MultiSellList", readMultiSellList},
	0xd3:                      {name: "NetPing"},
	OpHennaInfo:               {"HennaInfo", readHennaInfo},
	OpRadarControl:            {"RadarControl", readRadarControl},
	OpSendMacroList:           {"SendMacroList", readSendMacroList},
	0xec:                      {name: "ClientSetTime"},
	OpPartySpelled:            {"PartySpelled", readPartySpelled},
	OpEtcStatusUpdate:         {"EtcStatusUpdate", readEtcStatusUpdate},
	OpShortBuffStatusUpdate:   {"ShortBuffStatusUpdate", readShortBuffStatusUpdate},
	0xf5:                      {name: "SSQStatus"},
	0xf9:                      {name: "GameGuardQuery"},
	OpFriendList:              {"FriendList", readFriendList},
}

// exServerPackets lists the Ex server packets by sub-opcode.
var exServerPackets = map[uint16]serverPacket{
	0x01:                {name: "ExRegenMax"},
	ExOpAutoSoulShot:    {"ExAutoSoulShot", readExAutoSoulShot},
	0x1a:                {"ExShowQuestMark", readExShowQuestMark},
	0x1b:                {name: "ExSendManorList"},
	ExOpStorageMaxCount: {"ExStorageMaxCount", readExStorageMaxCount},
	0x31:                {name: "ExPCCafePointInfo"},
	0x32:                {name: "ExSetCompassZoneCode"},
	0x38:                {"ExShowScreenMessage", readExShowScreenMessage},
}

// serverPacketName returns the Mobius name of a raw server packet (opcode
// first), or "" when the client does not know it.
func serverPacketName(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == OpExServer {
		if len(raw) < 3 {
			return ""
		}
		return exServerPackets[binary.LittleEndian.Uint16(raw[1:])].name
	}
	return serverPackets[raw[0]].name
}

// DecodeServerPacket parses a decrypted game server packet, opcode
// included, into one of the packet types of this package. A packet the
// client does not parse comes back as Unknown with a nil error. A packet
// whose layout differs from the Mobius writeImpl (too short, extra bytes,
// a boolean that is not 0 or 1, a negative count) gives an error wrapping
// ErrMalformed.
func DecodeServerPacket(b []byte) (Packet, error) {
	if len(b) == 0 {
		return nil, malformed("game packet", "empty body")
	}
	op := b[0]
	if op == OpKeyPacket {
		k, err := parseKeyPacket(b)
		if err != nil {
			return nil, err
		}
		return k, nil
	}
	r := packets.NewReader(b[1:])
	info, ok := serverPackets[op]
	if op == OpExServer {
		sub := r.ReadUint16()
		if err := r.Err(); err != nil {
			return nil, readerError("Ex packet", err)
		}
		info, ok = exServerPackets[sub]
	}
	if !ok || info.decode == nil {
		return Unknown{Opcode: op, Body: bytes.Clone(b[1:])}, nil
	}
	f := &fields{r: r, name: info.name}
	p := info.decode(f)
	if f.err != nil {
		return nil, f.err
	}
	if err := r.Err(); err != nil {
		return nil, readerError(info.name, err)
	}
	if r.Len() != 0 {
		return nil, malformed(info.name, "%d bytes after the last field", r.Len())
	}
	return p, nil
}

// Unknown is a game server packet the client does not parse.
type Unknown struct {
	Opcode byte
	// Body is the packet after the opcode. For an Ex packet (OpExServer) it
	// starts with the two-byte sub-opcode.
	Body []byte
}

// PacketName returns the Mobius name when the client knows the opcode,
// else "Unknown(0x..)", or "UnknownEx(0x....)" for an Ex packet.
func (u Unknown) PacketName() string {
	if sub, ok := u.ExOpcode(); ok {
		if name := exServerPackets[sub].name; name != "" {
			return name
		}
		return fmt.Sprintf("UnknownEx(0x%04x)", sub)
	}
	if name := serverPackets[u.Opcode].name; name != "" {
		return name
	}
	return fmt.Sprintf("Unknown(0x%02x)", u.Opcode)
}

// ExOpcode returns the sub-opcode of an Ex packet.
func (u Unknown) ExOpcode() (uint16, bool) {
	if u.Opcode != OpExServer || len(u.Body) < 2 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(u.Body), true
}

// fields reads the fields of one server packet, named after the Mobius
// WriteBuffer calls: c (u8), h (u16), d (i32), q (i64), f (f64), S (str).
// A read past the end leaves a zero value and an error in the reader; err
// holds the first flag or count error. Loops over lists stop at the first
// error (ok).
type fields struct {
	r    *packets.Reader
	name string
	err  error
}

func (f *fields) setErr(err error) {
	if f.err == nil && err != nil {
		f.err = err
	}
}

func (f *fields) failf(format string, args ...any) {
	f.setErr(malformed(f.name, format, args...))
}

// ok reports whether every read so far worked.
func (f *fields) ok() bool { return f.err == nil && f.r.Err() == nil }

func (f *fields) u8() uint8    { return f.r.ReadUint8() }
func (f *fields) i8() int8     { return int8(f.r.ReadUint8()) } //nolint:gosec // same bits as the Java byte
func (f *fields) u16() uint16  { return f.r.ReadUint16() }
func (f *fields) i16() int16   { return int16(f.r.ReadUint16()) } //nolint:gosec // same bits as the Java short
func (f *fields) i32() int32   { return f.r.ReadInt32() }
func (f *fields) i64() int64   { return int64(f.r.ReadUint64()) } //nolint:gosec // same bits as the Java long
func (f *fields) f64() float64 { return f.r.ReadFloat64() }
func (f *fields) str() string  { return f.r.ReadString() }

func (f *fields) i32s(dst []int32) {
	for i := range dst {
		dst[i] = f.i32()
	}
}

func (f *fields) u16s(dst []uint16) {
	for i := range dst {
		dst[i] = f.u16()
	}
}

// flag32 reads a Java boolean written with writeInt.
func (f *fields) flag32(field string) bool {
	v := f.i32()
	if v != 0 && v != 1 {
		f.failf("%s is %d, want 0 or 1", field, v)
	}
	return v == 1
}

// flag16 reads a Java boolean written with writeShort.
func (f *fields) flag16(field string) bool {
	v := f.u16()
	if v > 1 {
		f.failf("%s is %d, want 0 or 1", field, v)
	}
	return v == 1
}

// flag8 reads a Java boolean written with writeByte.
func (f *fields) flag8(field string) bool {
	v := f.u8()
	if v > 1 {
		f.failf("%s is %d, want 0 or 1", field, v)
	}
	return v == 1
}

// count32 reads a list length written with writeInt.
func (f *fields) count32(field string) int {
	n := f.i32()
	if n < 0 {
		f.failf("%s %d", field, n)
		return 0
	}
	return int(n)
}

// count16 reads a list length written with writeShort.
func (f *fields) count16() int { return int(f.u16()) }

// count8 reads a list length written with writeByte.
func (f *fields) count8() int { return int(f.u8()) }

// cubics reads the cubic list of UserInfo and CharInfo: h count, then h
// per cubic ID.
func (f *fields) cubics() []uint16 {
	var ids []uint16
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		ids = append(ids, f.u16())
	}
	return ids
}
