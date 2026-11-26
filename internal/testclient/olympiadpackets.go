package testclient

import "context"

// The Olympiad and hero packets (Plan 14, M11h; Mobius ServerPackets
// EX_HERO_LIST, EX_OLYMPIAD_*, OBSERVATION_MODE and OBSERVATION_RETURN).
const (
	ExOpHeroList            uint16 = 0x23
	ExOpOlympiadUserInfo    uint16 = 0x29
	ExOpOlympiadSpelledInfo uint16 = 0x2a
	ExOpOlympiadMode        uint16 = 0x2b
	ExOpOlympiadMatchEnd    uint16 = 0x2c

	OpObservationEnter = 0xdf
	OpObservationExit  = 0xe0
)

// The client's Olympiad requests (D0 0C, D0 12, D0 13).
const (
	ExOpRequestWriteHeroWords      uint16 = 0x0c
	ExOpRequestOlympiadObserverEnd uint16 = 0x12
	ExOpRequestOlympiadMatchList   uint16 = 0x13
)

func init() {
	exServerPackets[ExOpHeroList] = serverPacket{"ExHeroList", readExHeroList}
	exServerPackets[ExOpOlympiadUserInfo] = serverPacket{"ExOlympiadUserInfo", readExOlympiadUserInfo}
	exServerPackets[ExOpOlympiadSpelledInfo] = serverPacket{"ExOlympiadSpelledInfo", readExOlympiadSpelledInfo}
	exServerPackets[ExOpOlympiadMode] = serverPacket{"ExOlympiadMode", readExOlympiadMode}
	exServerPackets[ExOpOlympiadMatchEnd] = serverPacket{"ExOlympiadMatchEnd", func(*fields) Packet { return ExOlympiadMatchEnd{} }}
	serverPackets[OpObservationEnter] = serverPacket{"ObservationEnter", readObservationEnter}
	serverPackets[OpObservationExit] = serverPacket{"ObservationExit", readObservationExit}
}

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

// ExHeroList lists the heroes (FE 23).
type ExHeroList struct {
	Heroes []HeroListEntry
}

// PacketName implements Packet.
func (ExHeroList) PacketName() string { return "ExHeroList" }

func readExHeroList(f *fields) Packet {
	var p ExHeroList
	n := f.count32("hero count")
	for i := 0; i < n && f.ok(); i++ {
		var h HeroListEntry
		h.Name = f.str()
		h.ClassID = f.i32()
		h.ClanName = f.str()
		h.ClanCrestID = f.i32()
		h.AllyName = f.str()
		h.AllyCrestID = f.i32()
		h.Count = f.i32()
		p.Heroes = append(p.Heroes, h)
	}
	return p
}

// ExOlympiadUserInfo is a fighter's bar in the match window (FE 29).
type ExOlympiadUserInfo struct {
	Side     uint8
	ObjectID int32
	Name     string
	ClassID  int32
	HP       int32
	MaxHP    int32
	CP       int32
	MaxCP    int32
}

// PacketName implements Packet.
func (ExOlympiadUserInfo) PacketName() string { return "ExOlympiadUserInfo" }

func readExOlympiadUserInfo(f *fields) Packet {
	var p ExOlympiadUserInfo
	p.Side = f.u8()
	p.ObjectID = f.i32()
	p.Name = f.str()
	p.ClassID = f.i32()
	p.HP, p.MaxHP, p.CP, p.MaxCP = f.i32(), f.i32(), f.i32(), f.i32()
	return p
}

// SpelledEffect is an effect of ExOlympiadSpelledInfo.
type SpelledEffect struct {
	SkillID int32
	Level   int16
	Seconds int32
}

// ExOlympiadSpelledInfo shows a fighter's effects to the spectators (FE
// 2A).
type ExOlympiadSpelledInfo struct {
	ObjectID int32
	Effects  []SpelledEffect
}

// PacketName implements Packet.
func (ExOlympiadSpelledInfo) PacketName() string { return "ExOlympiadSpelledInfo" }

func readExOlympiadSpelledInfo(f *fields) Packet {
	var p ExOlympiadSpelledInfo
	p.ObjectID = f.i32()
	n := f.count32("effect count")
	for i := 0; i < n && f.ok(); i++ {
		p.Effects = append(p.Effects, SpelledEffect{SkillID: f.i32(), Level: f.i16(), Seconds: f.i32()})
	}
	return p
}

// ExOlympiadMode switches the olympiad view: 0 off, 2 a fighter, 3 a
// spectator (FE 2B).
type ExOlympiadMode struct {
	Mode uint8
}

// PacketName implements Packet.
func (ExOlympiadMode) PacketName() string { return "ExOlympiadMode" }

func readExOlympiadMode(f *fields) Packet { return ExOlympiadMode{Mode: f.u8()} }

// ExOlympiadMatchEnd ends the match window (FE 2C).
type ExOlympiadMatchEnd struct{}

// PacketName implements Packet.
func (ExOlympiadMatchEnd) PacketName() string { return "ExOlympiadMatchEnd" }

// ObservationEnter moves an observer's camera (0xDF).
type ObservationEnter struct {
	X, Y, Z int32
}

// PacketName implements Packet.
func (ObservationEnter) PacketName() string { return "ObservationEnter" }

func readObservationEnter(f *fields) Packet {
	p := ObservationEnter{X: f.i32(), Y: f.i32(), Z: f.i32()}
	f.u8()
	f.u8()
	f.u8()
	return p
}

// ObservationExit brings an observer's camera back (0xE0).
type ObservationExit struct {
	X, Y, Z int32
}

// PacketName implements Packet.
func (ObservationExit) PacketName() string { return "ObservationExit" }

func readObservationExit(f *fields) Packet {
	return ObservationExit{X: f.i32(), Y: f.i32(), Z: f.i32()}
}

// RequestWriteHeroWords sets the hero's words (D0 0C).
func (c *GameConn) RequestWriteHeroWords(ctx context.Context, words string) error {
	w := clanExWriter(ExOpRequestWriteHeroWords, 2+2*len(words))
	w.WriteString(words)
	return c.send(ctx, w.Bytes())
}

// RequestOlympiadObserverEnd leaves the olympiad observation (D0 12).
func (c *GameConn) RequestOlympiadObserverEnd(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestOlympiadObserverEnd))
}

// RequestOlympiadMatchList asks an observer's arena list (D0 13).
func (c *GameConn) RequestOlympiadMatchList(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestOlympiadMatchList))
}
