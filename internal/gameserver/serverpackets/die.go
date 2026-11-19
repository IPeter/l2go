package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpDie is the opcode of Die.
const OpDie = 0x06

// Die shows a creature dying. The player who died also gets the restart
// window, whose buttons send clientpackets.RequestRestartPoint.
type Die struct {
	ObjectID int32
	// CanTeleport shows the "to village" button (Mobius canRevive &&
	// !isPendingRevive). Without it the clan buttons are hidden too.
	CanTeleport bool
	// ToClanHall, ToCastle and ToSiegeHQ show the clan buttons. Mobius sets
	// them only for a clan member who is not in jail: the clan owns a clan
	// hall, owns a castle or defends the castle under siege, or attacks a
	// siege and has a flag (headquarters).
	ToClanHall bool
	ToCastle   bool
	ToSiegeHQ  bool
	// Sweepable makes a dead monster glow blue: its spoil can be swept.
	Sweepable bool
	// FixedRes shows the "fixed" button, to get up where the player died:
	// with a self-resurrection item, or with the access level's right.
	FixedRes bool
}

// NewDiePacket builds Die (Mobius Die.writeImpl). As in Mobius, the clan
// buttons are sent only with CanTeleport.
func NewDiePacket(d Die) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpDie)
	w.WriteInt32(d.ObjectID)
	putBool32(w, d.CanTeleport)
	putBool32(w, d.CanTeleport && d.ToClanHall)
	putBool32(w, d.CanTeleport && d.ToCastle)
	putBool32(w, d.CanTeleport && d.ToSiegeHQ)
	putBool32(w, d.Sweepable)
	putBool32(w, d.FixedRes)
	return w.Bytes()
}
