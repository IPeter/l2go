package testclient

// The siege window packets of a castle. Every reader follows the writeImpl
// of the Mobius class of the same name (castle branch), field by field.

// Game server opcodes of the siege window packets (Mobius ServerPackets).
const (
	OpSiegeInfo         byte = 0xc9
	OpSiegeAttackerList byte = 0xca
	OpSiegeDefenderList byte = 0xcb
)

// The type of a clan on the defender list (Mobius SiegeDefenderList).
const (
	SiegeDefenderOwner    int32 = 1
	SiegeDefenderWaiting  int32 = 2 // waiting for the owner's approval
	SiegeDefenderApproved int32 = 3
)

// The decoder entries are added here rather than in the shared tables, so
// the siege packets stay in one file.
func init() {
	serverPackets[OpSiegeInfo] = serverPacket{"SiegeInfo", readSiegeInfo}
	serverPackets[OpSiegeAttackerList] = serverPacket{"SiegeAttackerList", readSiegeAttackerList}
	serverPackets[OpSiegeDefenderList] = serverPacket{"SiegeDefenderList", readSiegeDefenderList}
}

// SiegeInfo is the siege window of a castle (0xc9).
type SiegeInfo struct {
	CastleID int32
	// OwnerLeader is set when the viewer leads the owner clan.
	OwnerLeader bool
	// OwnerClanID is 0 without an owner; the names are empty then.
	OwnerClanID     int32
	OwnerClanName   string
	OwnerLeaderName string
	OwnerAllyID     int32
	OwnerAllyName   string
	Now             int32 // the server time in Unix seconds
	// SiegeDate is the siege start in Unix seconds, 0 while the owner's
	// leader chooses the hour from TimeChoices.
	SiegeDate   int32
	TimeChoices []int32
}

// PacketName implements Packet.
func (SiegeInfo) PacketName() string { return "SiegeInfo" }

func readSiegeInfo(f *fields) Packet {
	var p SiegeInfo
	p.CastleID = f.i32()
	p.OwnerLeader = f.flag32("owner leader flag")
	p.OwnerClanID = f.i32()
	p.OwnerClanName = f.str()
	p.OwnerLeaderName = f.str()
	p.OwnerAllyID = f.i32()
	p.OwnerAllyName = f.str()
	p.Now = f.i32()
	p.SiegeDate = f.i32()
	n := f.count32("siege time choice count")
	if p.SiegeDate != 0 && n != 0 {
		f.failf("siege date %d with %d time choices", p.SiegeDate, n)
	}
	for i := 0; i < n && f.ok(); i++ {
		p.TimeChoices = append(p.TimeChoices, f.i32())
	}
	return p
}

// SiegeClan is one clan line of the siege lists.
type SiegeClan struct {
	ClanID     int32
	ClanName   string
	LeaderName string
	CrestID    int32
	SignedTime int32 // 0 in Mobius
	// Type is one of the SiegeDefender values; the attacker list has none.
	Type           int32
	AllyID         int32
	AllyName       string
	AllyLeaderName string // empty in Mobius
	AllyCrestID    int32
}

// siegeList reads the head and clan lines of the two lists. Mobius writes
// d castle, d 0, d 1, d 0 and the clan count twice.
func (f *fields) siegeList(withType bool) (castleID int32, head [3]int32, clans []SiegeClan) {
	castleID = f.i32()
	f.i32s(head[:])
	n := f.count32("clan count")
	if n2 := f.count32("second clan count"); f.ok() && n2 != n {
		f.failf("clan counts %d and %d differ", n, n2)
	}
	for i := 0; i < n && f.ok(); i++ {
		var c SiegeClan
		c.ClanID = f.i32()
		c.ClanName = f.str()
		c.LeaderName = f.str()
		c.CrestID = f.i32()
		c.SignedTime = f.i32()
		if withType {
			c.Type = f.i32()
		}
		c.AllyID = f.i32()
		c.AllyName = f.str()
		c.AllyLeaderName = f.str()
		c.AllyCrestID = f.i32()
		clans = append(clans, c)
	}
	return castleID, head, clans
}

// SiegeAttackerList is the attacker tab of the siege window (0xca).
type SiegeAttackerList struct {
	CastleID int32
	// Head is the three words after the castle, 0, 1, 0 in Mobius.
	Head  [3]int32
	Clans []SiegeClan
}

// PacketName implements Packet.
func (SiegeAttackerList) PacketName() string { return "SiegeAttackerList" }

// Clan returns the line of the clan name.
func (p SiegeAttackerList) Clan(name string) (SiegeClan, bool) { return siegeClan(p.Clans, name) }

func readSiegeAttackerList(f *fields) Packet {
	var p SiegeAttackerList
	p.CastleID, p.Head, p.Clans = f.siegeList(false)
	return p
}

// SiegeDefenderList is the defender tab of the siege window (0xcb).
type SiegeDefenderList struct {
	CastleID int32
	// Head is the three words after the castle, 0, 1, 0 in Mobius.
	Head  [3]int32
	Clans []SiegeClan
}

// PacketName implements Packet.
func (SiegeDefenderList) PacketName() string { return "SiegeDefenderList" }

// Clan returns the line of the clan name.
func (p SiegeDefenderList) Clan(name string) (SiegeClan, bool) { return siegeClan(p.Clans, name) }

func readSiegeDefenderList(f *fields) Packet {
	var p SiegeDefenderList
	p.CastleID, p.Head, p.Clans = f.siegeList(true)
	return p
}

func siegeClan(clans []SiegeClan, name string) (SiegeClan, bool) {
	for _, c := range clans {
		if c.ClanName == name {
			return c, true
		}
	}
	return SiegeClan{}, false
}
