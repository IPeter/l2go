package testclient

import (
	"fmt"
	"strings"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// Game client packet opcodes, from Mobius ClientPackets.
const (
	OpProtocolVersion byte = 0x00
	OpAuthLogin       byte = 0x08
	OpCharacterCreate byte = 0x0b
	OpNewCharacter    byte = 0x0e
)

// Game server packet opcodes, from Mobius ServerPackets.
const (
	OpKeyPacket           byte = 0x00
	OpCharSelectionInfo   byte = 0x13
	OpGameLoginFail       byte = 0x14
	OpNewCharacterSuccess byte = 0x17
	OpCharCreateOk        byte = 0x19
	OpCharCreateFail      byte = 0x1a
)

// Protocol versions for ProtocolVersion.
const (
	// InterludeProtocol is the protocol of the Interlude client (Mobius
	// AllowedProtocolRevisions default).
	InterludeProtocol int32 = 746
	// PingProtocol makes the server close the connection at once.
	PingProtocol int32 = -2
)

// GameLoginFailReason is the reason of the game server LoginFail (Mobius
// gameserver LoginFail).
type GameLoginFailReason int32

// Game server LoginFail reasons.
const (
	GameLoginFailNoText                GameLoginFailReason = 0
	GameLoginFailSystemErrorLoginLater GameLoginFailReason = 1
	GameLoginFailPasswordDoesNotMatch  GameLoginFailReason = 2
	GameLoginFailAccessFailedTryLater  GameLoginFailReason = 4
	GameLoginFailIncorrectAccountInfo  GameLoginFailReason = 5
	GameLoginFailAccountAlreadyInUse   GameLoginFailReason = 7
)

// CreateFailReason is the reason of CharCreateFail (Mobius CharCreateFail).
type CreateFailReason int32

// CharCreateFail reasons.
const (
	// CreateFailCreationFailed: "Your character creation has failed."
	CreateFailCreationFailed CreateFailReason = 0x00
	// CreateFailTooManyCharacters: the account has the maximum number of
	// characters.
	CreateFailTooManyCharacters CreateFailReason = 0x01
	// CreateFailNameExists: "This name already exists."
	CreateFailNameExists CreateFailReason = 0x02
	// CreateFailNameTooLong: the name is empty or longer than 16
	// characters (Mobius REASON_16_ENG_CHARS).
	CreateFailNameTooLong CreateFailReason = 0x03
	// CreateFailIncorrectName: "Incorrect name. Please try again."
	CreateFailIncorrectName CreateFailReason = 0x04
	// CreateFailNotAllowed: "Characters cannot be created from this
	// server."
	CreateFailNotAllowed CreateFailReason = 0x05
	// CreateFailChooseAnotherServer: creation is restricted to servers
	// where the account already has characters.
	CreateFailChooseAnotherServer CreateFailReason = 0x06
)

var createFailNames = map[CreateFailReason]string{
	CreateFailCreationFailed:      "creation failed",
	CreateFailTooManyCharacters:   "too many characters",
	CreateFailNameExists:          "name exists",
	CreateFailNameTooLong:         "name too long",
	CreateFailIncorrectName:       "incorrect name",
	CreateFailNotAllowed:          "creation not allowed",
	CreateFailChooseAnotherServer: "choose another server",
}

func (r CreateFailReason) String() string {
	if s, ok := createFailNames[r]; ok {
		return fmt.Sprintf("0x%02x (%s)", int32(r), s)
	}
	return fmt.Sprintf("0x%02x", int32(r))
}

// Equipment slots of CharSelectionInfo, in packet order. The packet lists
// 17 object IDs and then 17 item IDs in this order.
const (
	SlotUnder = iota
	SlotREar
	SlotLEar
	SlotNeck
	SlotRFinger
	SlotLFinger
	SlotHead
	SlotRHand
	SlotLHand
	SlotGloves
	SlotChest
	SlotLegs
	SlotFeet
	SlotCloak
	// SlotRHand2 repeats the right hand: Mobius writes PAPERDOLL_RHAND
	// again here (older L2J wrote the two-handed slot).
	SlotRHand2
	SlotHair
	SlotHair2

	// NumEquipSlots is the number of equipment slots in CharSelectionInfo.
	NumEquipSlots
)

// SlotFace is where UserInfo lists the face slot (Mobius PAPERDOLL_FACE):
// the place where CharSelectionInfo lists HAIR2.
const SlotFace = SlotHair2

// KeyPacket answers ProtocolVersion (Mobius KeyPacket).
type KeyPacket struct {
	// Result is 1 when the protocol is accepted, 0 when it is not.
	Result byte
	// KeyHalf is the random first half of the XOR key.
	KeyHalf [gamecrypt.HalfSize]byte
	// Encryption is true when the server encrypts packets (Mobius
	// PacketEncryption, retail true).
	Encryption bool
	ServerID   int32
	// Unknown is a byte Mobius writes as 1.
	Unknown byte
	// ObfuscationKey is 0 in Mobius.
	ObfuscationKey int32
}

// CharacterList is the CharSelectionInfo packet.
type CharacterList struct {
	Characters []CharInfo
}

// Active returns the character marked as last used, if any.
func (l CharacterList) Active() (CharInfo, bool) {
	for _, c := range l.Characters {
		if c.Active {
			return c, true
		}
	}
	return CharInfo{}, false
}

// Find returns the character with the given name, if any.
func (l CharacterList) Find(name string) (CharInfo, bool) {
	for _, c := range l.Characters {
		if c.Name == name {
			return c, true
		}
	}
	return CharInfo{}, false
}

// CharInfo is one character of CharSelectionInfo, with every field Mobius
// CharSelectionInfo.writeImpl writes, in packet order.
type CharInfo struct {
	Name     string
	ObjectID int32
	Account  string
	// SessionID is the "account ID" field. Mobius sends the PlayOK1 key.
	SessionID int32
	ClanID    int32
	// BuilderLevel is 0 in Mobius.
	BuilderLevel int32
	Sex          int32
	Race         int32
	BaseClassID  int32
	// GameServer is the "game server name" field. Mobius writes 1.
	GameServer int32
	// X, Y and Z are written as 0 by Mobius.
	X, Y, Z   int32
	CurrentHP float64
	CurrentMP float64
	SP        int32
	Exp       int64
	Level     int32
	Karma     int32
	// Reserved holds nine words that Mobius writes as zero.
	Reserved [9]int32
	// EquipObjectIDs and EquipItemIDs are indexed by the Slot constants.
	// Empty slots are 0.
	EquipObjectIDs [NumEquipSlots]int32
	EquipItemIDs   [NumEquipSlots]int32
	HairStyle      int32
	HairColor      int32
	Face           int32
	MaxHP          float64
	MaxMP          float64
	// DeleteTimer is the number of seconds until the character is deleted,
	// 0 when it is not being deleted, or -1 when it is banned (Mobius
	// access level below 0).
	DeleteTimer int32
	ClassID     int32
	// Active marks the character played last.
	Active bool
	// Enchant is the weapon enchant level shown, at most 127.
	Enchant        uint8
	AugmentationID int32
}

// StatRange is one stat of a character template in NewCharacterSuccess.
// Mobius writes 0x46, the base value, then 0x0a for every stat.
type StatRange struct {
	Max, Base, Min int32
}

// CharTemplate is one starting class of NewCharacterSuccess.
type CharTemplate struct {
	Race    int32
	ClassID int32
	// Stats in packet order: STR, DEX, CON, INT, WIT, MEN.
	STR, DEX, CON, INT, WIT, MEN StatRange
}

// NewCharacterSuccess lists the starting classes.
type NewCharacterSuccess struct {
	Templates []CharTemplate
}

// CharCreateOk accepts CharacterCreate.
type CharCreateOk struct {
	// Result is 1 in Mobius.
	Result int32
}

// CharCreateFail refuses CharacterCreate.
type CharCreateFail struct {
	Reason CreateFailReason
}

// GameLoginFail refuses AuthLogin (game server LoginFail, 0x14).
type GameLoginFail struct {
	Reason GameLoginFailReason
}

// GameLoginFailError is returned by AuthLogin when the game server answers
// with LoginFail.
type GameLoginFailError struct {
	Reason GameLoginFailReason
}

func (e *GameLoginFailError) Error() string {
	return fmt.Sprintf("testclient: game server sent LoginFail, reason %d", e.Reason)
}

// CreateRequest holds the fields of CharacterCreate. Mobius reads Race and
// the six stats but ignores them; the client sends the template's values.
type CreateRequest struct {
	Name    string
	Race    int32
	Sex     int32 // 0 male, 1 female
	ClassID int32
	// Stats in packet order: INT, STR, CON, MEN, DEX, WIT.
	INT, STR, CON, MEN, DEX, WIT int32
	HairStyle                    int32
	HairColor                    int32
	Face                         int32
}

// encodeProtocolVersion builds ProtocolVersion.
func encodeProtocolVersion(version int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpProtocolVersion)
	w.WriteInt32(version)
	return w.Bytes()
}

// encodeAuthLogin builds AuthLogin in Mobius AuthLogin.readImpl order: the
// account name in lower case, then PlayOK2, PlayOK1, LoginOK1, LoginOK2.
func encodeAuthLogin(account string, k session.Keys) []byte {
	w := packets.NewWriter(64)
	w.WriteUint8(OpAuthLogin)
	w.WriteString(strings.ToLower(account))
	w.WriteInt32(k.PlayOK2)
	w.WriteInt32(k.PlayOK1)
	w.WriteInt32(k.LoginOK1)
	w.WriteInt32(k.LoginOK2)
	return w.Bytes()
}

// encodeNewCharacter builds NewCharacter, which has no fields.
func encodeNewCharacter() []byte {
	return []byte{OpNewCharacter}
}

// encodeCharacterCreate builds CharacterCreate in Mobius
// CharacterCreate.readImpl order.
func encodeCharacterCreate(req CreateRequest) []byte {
	w := packets.NewWriter(96)
	w.WriteUint8(OpCharacterCreate)
	w.WriteString(req.Name)
	for _, v := range []int32{
		req.Race, req.Sex, req.ClassID,
		req.INT, req.STR, req.CON, req.MEN, req.DEX, req.WIT,
		req.HairStyle, req.HairColor, req.Face,
	} {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// PacketName implements Packet.
func (KeyPacket) PacketName() string { return "KeyPacket" }

// PacketName implements Packet.
func (CharacterList) PacketName() string { return "CharSelectionInfo" }

// PacketName implements Packet.
func (GameLoginFail) PacketName() string { return "LoginFail" }

// PacketName implements Packet.
func (NewCharacterSuccess) PacketName() string { return "NewCharacterSuccess" }

// PacketName implements Packet.
func (CharCreateOk) PacketName() string { return "CharCreateOk" }

// PacketName implements Packet.
func (CharCreateFail) PacketName() string { return "CharCreateFail" }

// unexpectedGame builds the error for a game packet the call did not want.
func unexpectedGame(body []byte, want string) *UnexpectedPacketError {
	e := &UnexpectedPacketError{Server: "game", Want: want}
	if len(body) > 0 {
		e.Opcode = body[0]
		e.Name = serverPacketName(body)
		e.Body = append([]byte(nil), body[1:]...)
	}
	return e
}

// parseKeyPacket parses KeyPacket as Mobius KeyPacket.writeImpl writes it:
// c result, 8 key bytes, d encryption, d server ID, c 1, d obfuscation key.
func parseKeyPacket(body []byte) (KeyPacket, error) {
	const name = "KeyPacket"
	var k KeyPacket
	if len(body) == 0 || body[0] != OpKeyPacket {
		return k, unexpectedGame(body, name)
	}
	r := packets.NewReader(body[1:])
	k.Result = r.ReadUint8()
	copy(k.KeyHalf[:], r.ReadBytes(gamecrypt.HalfSize))
	enc := r.ReadInt32()
	k.ServerID = r.ReadInt32()
	k.Unknown = r.ReadUint8()
	k.ObfuscationKey = r.ReadInt32()
	if err := r.Err(); err != nil {
		return k, readerError(name, err)
	}
	if r.Len() != 0 {
		return k, malformed(name, "%d bytes after the last field", r.Len())
	}
	if k.Result > 1 {
		return k, malformed(name, "result is %d, want 0 or 1", k.Result)
	}
	var err error
	if k.Encryption, err = boolValue(name, "encryption flag", enc); err != nil {
		return k, err
	}
	return k, nil
}

// decodeGamePacket parses a decrypted game packet that followed KeyPacket,
// the way the request methods do: DecodeServerPacket, with an unknown
// opcode or a second KeyPacket as an *UnexpectedPacketError.
func decodeGamePacket(body []byte) (any, error) {
	p, err := DecodeServerPacket(body)
	p, err = received{raw: body, p: p, err: err}.known("")
	if err != nil {
		return nil, err
	}
	return p, nil
}

// The packets of the character screen, for the decoder table.

func readCharSelectionInfoPacket(f *fields) Packet {
	l, err := readCharSelectionInfo(f.r)
	f.setErr(err)
	return l
}

func readNewCharacterSuccessPacket(f *fields) Packet {
	s, err := readNewCharacterSuccess(f.r)
	f.setErr(err)
	return s
}

func readGameLoginFail(f *fields) Packet {
	return GameLoginFail{Reason: GameLoginFailReason(f.i32())}
}

func readCharCreateOk(f *fields) Packet { return CharCreateOk{Result: f.i32()} }

func readCharCreateFail(f *fields) Packet {
	return CharCreateFail{Reason: CreateFailReason(f.i32())}
}

// readCharSelectionInfo parses CharSelectionInfo after its opcode: a count
// and then one entry per character.
func readCharSelectionInfo(r *packets.Reader) (CharacterList, error) {
	const name = "CharSelectionInfo"
	var l CharacterList
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return l, readerError(name, err)
	}
	if n < 0 {
		return l, malformed(name, "character count %d", n)
	}
	for i := range int(n) {
		c, err := readCharInfo(r)
		if err != nil {
			return CharacterList{}, fmt.Errorf("character %d of %d: %w", i, n, err)
		}
		l.Characters = append(l.Characters, c)
	}
	return l, nil
}

// readCharInfo parses one character entry in Mobius
// CharSelectionInfo.writeImpl order.
func readCharInfo(r *packets.Reader) (CharInfo, error) {
	const name = "CharSelectionInfo"
	var c CharInfo
	c.Name = r.ReadString()
	c.ObjectID = r.ReadInt32()
	c.Account = r.ReadString()
	c.SessionID = r.ReadInt32()
	c.ClanID = r.ReadInt32()
	c.BuilderLevel = r.ReadInt32()
	c.Sex = r.ReadInt32()
	c.Race = r.ReadInt32()
	c.BaseClassID = r.ReadInt32()
	c.GameServer = r.ReadInt32()
	c.X = r.ReadInt32()
	c.Y = r.ReadInt32()
	c.Z = r.ReadInt32()
	c.CurrentHP = r.ReadFloat64()
	c.CurrentMP = r.ReadFloat64()
	c.SP = r.ReadInt32()
	c.Exp = int64(r.ReadUint64()) //nolint:gosec // same bits as the Java long
	c.Level = r.ReadInt32()
	c.Karma = r.ReadInt32()
	for i := range c.Reserved {
		c.Reserved[i] = r.ReadInt32()
	}
	for i := range c.EquipObjectIDs {
		c.EquipObjectIDs[i] = r.ReadInt32()
	}
	for i := range c.EquipItemIDs {
		c.EquipItemIDs[i] = r.ReadInt32()
	}
	c.HairStyle = r.ReadInt32()
	c.HairColor = r.ReadInt32()
	c.Face = r.ReadInt32()
	c.MaxHP = r.ReadFloat64()
	c.MaxMP = r.ReadFloat64()
	c.DeleteTimer = r.ReadInt32()
	c.ClassID = r.ReadInt32()
	active := r.ReadInt32()
	c.Enchant = r.ReadUint8()
	c.AugmentationID = r.ReadInt32()
	if err := r.Err(); err != nil {
		return c, readerError(name, err)
	}
	var err error
	if c.Active, err = boolValue(name, "active flag", active); err != nil {
		return c, err
	}
	return c, nil
}

// readNewCharacterSuccess parses NewCharacterSuccess after its opcode: a
// count, then per template race, class and six (0x46, base, 0x0a) stat
// triples in the order STR, DEX, CON, INT, WIT, MEN.
func readNewCharacterSuccess(r *packets.Reader) (NewCharacterSuccess, error) {
	const name = "NewCharacterSuccess"
	var s NewCharacterSuccess
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return s, readerError(name, err)
	}
	if n < 0 {
		return s, malformed(name, "template count %d", n)
	}
	for i := range int(n) {
		t := CharTemplate{Race: r.ReadInt32(), ClassID: r.ReadInt32()}
		for _, st := range []*StatRange{&t.STR, &t.DEX, &t.CON, &t.INT, &t.WIT, &t.MEN} {
			st.Max = r.ReadInt32()
			st.Base = r.ReadInt32()
			st.Min = r.ReadInt32()
		}
		if err := r.Err(); err != nil {
			return NewCharacterSuccess{}, readerError(fmt.Sprintf("%s template %d of %d", name, i, n), err)
		}
		s.Templates = append(s.Templates, t)
	}
	return s, nil
}
