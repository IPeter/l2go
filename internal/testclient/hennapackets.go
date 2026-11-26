package testclient

import "context"

// The symbol maker's windows (M11c): the henna client packets and the
// server packets Mobius answers with. HennaInfo is in worldpackets.go.

// Henna client opcodes (Mobius ClientPackets REQUEST_HENNA_*).
const (
	OpRequestHennaItemList       byte = 0xba
	OpRequestHennaItemInfo       byte = 0xbb
	OpRequestHennaEquip          byte = 0xbc
	OpRequestHennaRemoveList     byte = 0xbd
	OpRequestHennaItemRemoveInfo byte = 0xbe
	OpRequestHennaRemove         byte = 0xbf
)

// Henna server opcodes (Mobius ServerPackets HENNA_*).
const (
	OpHennaEquipList      byte = 0xe2
	OpHennaItemInfo       byte = 0xe3 // Mobius HennaItemDrawInfo
	OpHennaRemoveList     byte = 0xe5
	OpHennaItemRemoveInfo byte = 0xe6
)

func init() {
	for op, name := range map[byte]string{
		OpRequestHennaItemList:       "RequestHennaItemList",
		OpRequestHennaItemInfo:       "RequestHennaItemInfo",
		OpRequestHennaEquip:          "RequestHennaEquip",
		OpRequestHennaRemoveList:     "RequestHennaRemoveList",
		OpRequestHennaItemRemoveInfo: "RequestHennaItemRemoveInfo",
		OpRequestHennaRemove:         "RequestHennaRemove",
	} {
		if _, dup := clientPacketNames[op]; dup {
			panic("testclient: client opcode registered twice: " + name)
		}
		clientPacketNames[op] = name
	}
	for op, p := range map[byte]serverPacket{
		OpHennaEquipList:      {"HennaEquipList", readHennaEquipList},
		OpHennaItemInfo:       {"HennaItemDrawInfo", readHennaItemDrawInfo},
		OpHennaRemoveList:     {"HennaRemoveList", readHennaRemoveList},
		OpHennaItemRemoveInfo: {"HennaItemRemoveInfo", readHennaItemRemoveInfo},
	} {
		if _, dup := serverPackets[op]; dup {
			panic("testclient: server opcode registered twice: " + p.name)
		}
		serverPackets[op] = p
	}
}

// HennaEntry is one dye of HennaEquipList or HennaRemoveList.
type HennaEntry struct {
	DyeID, DyeItemID int32
	// Count and Fee are the dyes and adena drawing takes, or the dyes
	// removing gives back and the adena it takes.
	Count, Fee int32
	// Allowed is 1 in Mobius.
	Allowed int32
}

// HennaEquipList is the "draw a symbol" window (0xe2).
type HennaEquipList struct {
	Adena int32
	Slots int32 // 3 in Mobius
	// Listed is the number of dyes the class may wear; Mobius sends only
	// those whose dye item the player carries, so Entries can be fewer. The
	// entries are read up to the end of the packet.
	Listed  int32
	Entries []HennaEntry
}

// PacketName implements Packet.
func (HennaEquipList) PacketName() string { return "HennaEquipList" }

func readHennaEquipList(f *fields) Packet {
	var p HennaEquipList
	p.Adena = f.i32()
	p.Slots = f.i32()
	p.Listed = f.i32()
	p.Entries = readHennaEntries(f)
	return p
}

// HennaRemoveList is the "delete a symbol" window (0xe5).
type HennaRemoveList struct {
	Adena      int32
	EmptySlots int32
	// Used is |EmptySlots - 3|, which Mobius writes where a count would be;
	// the entries are read up to the end of the packet.
	Used    int32
	Entries []HennaEntry
}

// PacketName implements Packet.
func (HennaRemoveList) PacketName() string { return "HennaRemoveList" }

func readHennaRemoveList(f *fields) Packet {
	var p HennaRemoveList
	p.Adena = f.i32()
	p.EmptySlots = f.i32()
	p.Used = f.i32()
	p.Entries = readHennaEntries(f)
	return p
}

// readHennaEntries reads entries of 5 ints until the packet ends.
func readHennaEntries(f *fields) []HennaEntry {
	var out []HennaEntry
	for f.r.Len() > 0 && f.ok() {
		out = append(out, HennaEntry{DyeID: f.i32(), DyeItemID: f.i32(), Count: f.i32(), Fee: f.i32(), Allowed: f.i32()})
	}
	return out
}

// HennaStat is a base stat of the dye details: the value now and with the
// dye drawn or removed (a byte in Mobius).
type HennaStat struct {
	Now   int32
	After uint8
}

// HennaItemInfo is the body of HennaItemDrawInfo (0xe3) and
// HennaItemRemoveInfo (0xe6).
type HennaItemInfo struct {
	DyeID, DyeItemID int32
	Count, Fee       int32
	Allowed          bool
	Adena            int32
	// The stats in packet order.
	INT, STR, CON, MEN, DEX, WIT HennaStat
}

// HennaItemDrawInfo is what drawing a dye costs and does (0xe3).
type HennaItemDrawInfo struct{ HennaItemInfo }

// PacketName implements Packet.
func (HennaItemDrawInfo) PacketName() string { return "HennaItemDrawInfo" }

// HennaItemRemoveInfo is what removing a dye gives back, costs and does
// (0xe6).
type HennaItemRemoveInfo struct{ HennaItemInfo }

// PacketName implements Packet.
func (HennaItemRemoveInfo) PacketName() string { return "HennaItemRemoveInfo" }

func readHennaItemDrawInfo(f *fields) Packet {
	return HennaItemDrawInfo{readHennaItemInfo(f)}
}

func readHennaItemRemoveInfo(f *fields) Packet {
	return HennaItemRemoveInfo{readHennaItemInfo(f)}
}

func readHennaItemInfo(f *fields) HennaItemInfo {
	var p HennaItemInfo
	p.DyeID = f.i32()
	p.DyeItemID = f.i32()
	p.Count = f.i32()
	p.Fee = f.i32()
	p.Allowed = f.flag32("allowed")
	p.Adena = f.i32()
	for _, st := range []*HennaStat{&p.INT, &p.STR, &p.CON, &p.MEN, &p.DEX, &p.WIT} {
		st.Now = f.i32()
		st.After = f.u8()
	}
	return p
}

// RequestHennaItemList asks for the "draw a symbol" window again
// (HennaEquipList comes back).
func (c *GameConn) RequestHennaItemList(ctx context.Context) error {
	return c.send(ctx, encodeInt(OpRequestHennaItemList, 0))
}

// RequestHennaItemInfo asks what drawing a dye costs and does
// (HennaItemDrawInfo comes back, ActionFailed for an unknown dye).
func (c *GameConn) RequestHennaItemInfo(ctx context.Context, dyeID int32) error {
	return c.send(ctx, encodeInt(OpRequestHennaItemInfo, dyeID))
}

// RequestHennaEquip draws a dye. Mobius answers with HennaInfo, UserInfo,
// the item updates and "The symbol has been added.", or with a system
// message and ActionFailed.
func (c *GameConn) RequestHennaEquip(ctx context.Context, dyeID int32) error {
	return c.send(ctx, encodeInt(OpRequestHennaEquip, dyeID))
}

// RequestHennaRemoveList asks for the "delete a symbol" window again
// (HennaRemoveList comes back).
func (c *GameConn) RequestHennaRemoveList(ctx context.Context) error {
	return c.send(ctx, encodeInt(OpRequestHennaRemoveList, 0))
}

// RequestHennaItemRemoveInfo asks what removing a dye gives back, costs and
// does (HennaItemRemoveInfo comes back).
func (c *GameConn) RequestHennaItemRemoveInfo(ctx context.Context, dyeID int32) error {
	return c.send(ctx, encodeInt(OpRequestHennaItemRemoveInfo, dyeID))
}

// RequestHennaRemove removes a worn dye. Mobius answers with HennaInfo,
// UserInfo, the item updates and "The symbol has been deleted.", or with
// ActionFailed.
func (c *GameConn) RequestHennaRemove(ctx context.Context, dyeID int32) error {
	return c.send(ctx, encodeInt(OpRequestHennaRemove, dyeID))
}

// DrawHenna draws a dye and returns the HennaInfo that shows it. A refused
// draw ends with the context; negative tests use RequestHennaEquip.
func (c *GameConn) DrawHenna(ctx context.Context, dyeID int32) (HennaInfo, error) {
	if err := c.RequestHennaEquip(ctx, dyeID); err != nil {
		return HennaInfo{}, err
	}
	return Await[HennaInfo](ctx, c, nil)
}

// RemoveHenna removes a worn dye and returns the HennaInfo without it. A
// refused removal ends with the context; negative tests use
// RequestHennaRemove.
func (c *GameConn) RemoveHenna(ctx context.Context, dyeID int32) (HennaInfo, error) {
	if err := c.RequestHennaRemove(ctx, dyeID); err != nil {
		return HennaInfo{}, err
	}
	return Await[HennaInfo](ctx, c, nil)
}
