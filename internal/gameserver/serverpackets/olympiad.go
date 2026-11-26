package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The Olympiad and hero packets (Plan 14, M11h; Mobius ServerPackets
// EX_HERO_LIST, EX_OLYMPIAD_*, OBSERVATION_MODE, OBSERVATION_RETURN).
const (
	ExOpHeroList            = 0x23
	ExOpOlympiadUserInfo    = 0x29
	ExOpOlympiadSpelledInfo = 0x2a
	ExOpOlympiadMode        = 0x2b
	ExOpOlympiadMatchEnd    = 0x2c

	// OpObservationEnter (OBSERVATION_MODE) and OpObservationExit
	// (OBSERVATION_RETURN) move the camera of an observer.
	OpObservationEnter = 0xdf
	OpObservationExit  = 0xe0
)

// The modes of ExOlympiadMode.
const (
	// OlympiadModeNone ends the olympiad view.
	OlympiadModeNone uint8 = 0
	// OlympiadModeFighter puts a noble's client into the match view
	// (Mobius sends 2 for both sides).
	OlympiadModeFighter uint8 = 2
	// OlympiadModeSpectator is the view of a spectator.
	OlympiadModeSpectator uint8 = 3
)

// HeroListEntry is a hero of ExHeroList.
type HeroListEntry struct {
	Name        string
	ClassID     int32
	ClanName    string
	ClanCrestID int32
	AllyName    string
	AllyCrestID int32
	Count       int32
}

// NewExHeroListPacket builds ExHeroList (Mobius ExHeroList.writeImpl): d
// count, then per hero S name, d class, S clan name, d clan crest, S
// alliance name, d alliance crest, d times a hero.
func NewExHeroListPacket(heroes []HeroListEntry) []byte {
	w := newExWriter(ExOpHeroList, 7+len(heroes)*80)
	w.WriteInt32(int32(len(heroes))) //nolint:gosec // a few dozen heroes
	for _, h := range heroes {
		w.WriteString(h.Name)
		w.WriteInt32(h.ClassID)
		w.WriteString(h.ClanName)
		w.WriteInt32(h.ClanCrestID)
		w.WriteString(h.AllyName)
		w.WriteInt32(h.AllyCrestID)
		w.WriteInt32(h.Count)
	}
	return w.Bytes()
}

// OlympiadUserInfo is a fighter's bar in the match window.
type OlympiadUserInfo struct {
	// Side is 1 or 2 (Mobius sends 1 to the opponent, the side to
	// spectators).
	Side     uint8
	ObjectID int32
	Name     string
	ClassID  int32
	HP       int32
	MaxHP    int32
	CP       int32
	MaxCP    int32
}

// NewExOlympiadUserInfoPacket builds ExOlympiadUserInfo (Mobius
// ExOlympiadUserInfo.writeImpl): c side, d object, S name, d class, d HP,
// d max HP, d CP, d max CP.
func NewExOlympiadUserInfoPacket(u OlympiadUserInfo) []byte {
	w := newExWriter(ExOpOlympiadUserInfo, 40+2*len(u.Name))
	w.WriteUint8(u.Side)
	w.WriteInt32(u.ObjectID)
	w.WriteString(u.Name)
	w.WriteInt32(u.ClassID)
	w.WriteInt32(u.HP)
	w.WriteInt32(u.MaxHP)
	w.WriteInt32(u.CP)
	w.WriteInt32(u.MaxCP)
	return w.Bytes()
}

// SpelledEffect is an effect of ExOlympiadSpelledInfo.
type SpelledEffect struct {
	SkillID int32
	Level   int16
	// Seconds is the time left, -1 for an endless one.
	Seconds int32
}

// NewExOlympiadSpelledInfoPacket builds ExOlympiadSpelledInfo (Mobius
// ExOlympiadSpelledInfo.writeImpl): d the fighter, d count, then per effect
// d skill, h level, d seconds left. Spectators get it with the fighters'
// effects.
func NewExOlympiadSpelledInfoPacket(objectID int32, effects []SpelledEffect) []byte {
	w := newExWriter(ExOpOlympiadSpelledInfo, 11+len(effects)*10)
	w.WriteInt32(objectID)
	w.WriteInt32(int32(len(effects))) //nolint:gosec // a few dozen effects
	for _, e := range effects {
		w.WriteInt32(e.SkillID)
		w.WriteUint16(uint16(e.Level)) //nolint:gosec // same bits
		w.WriteInt32(e.Seconds)
	}
	return w.Bytes()
}

// NewExOlympiadModePacket builds ExOlympiadMode (Mobius
// ExOlympiadMode.writeImpl): c mode.
func NewExOlympiadModePacket(mode uint8) []byte {
	w := newExWriter(ExOpOlympiadMode, 4)
	w.WriteUint8(mode)
	return w.Bytes()
}

// NewExOlympiadMatchEndPacket builds ExOlympiadMatchEnd (Mobius
// ExOlympiadMatchEnd.writeImpl): the sub-opcode only.
func NewExOlympiadMatchEndPacket() []byte { return newExWriter(ExOpOlympiadMatchEnd, 3).Bytes() }

// NewObservationEnterPacket builds ObservationEnter (Mobius
// ObservationEnter.writeImpl, OBSERVATION_MODE): d x, d y, d z, c 0, c
// 0xC0, c 0.
func NewObservationEnterPacket(x, y, z int32) []byte {
	w := packets.NewWriter(16)
	w.WriteUint8(OpObservationEnter)
	w.WriteInt32(x)
	w.WriteInt32(y)
	w.WriteInt32(z)
	w.WriteUint8(0)
	w.WriteUint8(0xc0)
	w.WriteUint8(0)
	return w.Bytes()
}

// NewObservationExitPacket builds ObservationExit (Mobius
// ObservationExit.writeImpl, OBSERVATION_RETURN): d x, d y, d z.
func NewObservationExitPacket(x, y, z int32) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpObservationExit)
	w.WriteInt32(x)
	w.WriteInt32(y)
	w.WriteInt32(z)
	return w.Bytes()
}
