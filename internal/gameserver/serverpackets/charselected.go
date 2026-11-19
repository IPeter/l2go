package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpCharSelected is the opcode of CharSelected.
const OpCharSelected = 0x15

// gameMinutesPerDay is the length of a game day in game minutes.
const gameMinutesPerDay = 24 * 60

// CharSelected answers CharacterSelect with the chosen character. The client
// shows the loading screen and sends EnterWorld.
type CharSelected struct {
	Name      string
	ObjectID  int32
	Title     string
	SessionID int32 // the PlayOK1 session key
	ClanID    int32
	Female    bool
	Race      int32
	ClassID   int32
	X, Y, Z   int32
	CurHP     float64
	CurMP     float64
	SP        int32
	Exp       int64
	Level     int32
	Karma     int32
	PKKills   int32
	INT       int32
	STR       int32
	CON       int32
	MEN       int32
	DEX       int32
	WIT       int32
	// GameTime is the game time in game minutes. Only the time of day is
	// sent: GameTime modulo 1,440, as Mobius does.
	GameTime int32
}

// NewCharSelectedPacket builds CharSelected (Mobius CharSelected.writeImpl).
// Note the stat order: INT, STR, CON, MEN, DEX, WIT.
func NewCharSelectedPacket(c CharSelected) []byte {
	w := packets.NewWriter(301 + 2*(len(c.Name)+len(c.Title)+2))
	w.WriteUint8(OpCharSelected)
	w.WriteString(c.Name)
	w.WriteInt32(c.ObjectID)
	w.WriteString(c.Title)
	w.WriteInt32(c.SessionID)
	w.WriteInt32(c.ClanID)
	w.WriteInt32(0) // unknown
	putBool32(w, c.Female)
	w.WriteInt32(c.Race)
	w.WriteInt32(c.ClassID)
	w.WriteInt32(1) // active
	w.WriteInt32(c.X)
	w.WriteInt32(c.Y)
	w.WriteInt32(c.Z)
	w.WriteFloat64(c.CurHP)
	w.WriteFloat64(c.CurMP)
	w.WriteInt32(c.SP)
	putInt64(w, c.Exp)
	w.WriteInt32(c.Level)
	w.WriteInt32(c.Karma)
	w.WriteInt32(c.PKKills)
	w.WriteInt32(c.INT)
	w.WriteInt32(c.STR)
	w.WriteInt32(c.CON)
	w.WriteInt32(c.MEN)
	w.WriteInt32(c.DEX)
	w.WriteInt32(c.WIT)
	for range 32 {
		w.WriteInt32(0)
	}
	w.WriteInt32(c.GameTime % gameMinutesPerDay)
	w.WriteInt32(0)
	w.WriteInt32(c.ClassID)
	for range 12 {
		w.WriteInt32(0)
	}
	return w.Bytes()
}
