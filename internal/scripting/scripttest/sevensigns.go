package scripttest

import "github.com/IPeter/l2go/internal/scripting"

// The Seven Signs of the fake world (scripting.SevenSigns): a test sets the
// period, the seals and the cabals in World.SevenSigns; the scripts' joins
// and contributions change them; items move in the fake players' Items.

var _ scripting.SevenSigns = (*World)(nil)

// SevenSignsState is the fake Seven Signs.
type SevenSignsState struct {
	Period, Cycle int
	// Winner is the leading cabal (highest_cabal).
	Winner     int
	SealOwners [4]int
	// Cabals, Seals and Contributions are by player object ID; Rewards the
	// ancient adena each may collect.
	Cabals, Seals, Contributions, Rewards map[int32]int
	// Contributed are the stones contributed: {blue, green, red} per call.
	Contributed [][3]int
	Config      scripting.SevenSignsConfig
	// ClanCastles are the castles of the players' clans; Tickets the castles'
	// Dawn ticket sales.
	ClanCastles map[int32]int
	Tickets     map[int]int
	// Disappeared records send_disappeared: {item, count}.
	Disappeared [][2]int64
}

func (w *World) ss() *SevenSignsState {
	s := &w.SevenSigns
	if s.Cabals == nil {
		s.Cabals, s.Seals, s.Contributions, s.Rewards = map[int32]int{}, map[int32]int{}, map[int32]int{}, map[int32]int{}
		s.ClanCastles, s.Tickets = map[int32]int{}, map[int]int{}
	}
	return s
}

func (w *World) SevenSignsPeriod() int           { return w.ss().Period }
func (w *World) SevenSignsCycle() int            { return w.ss().Cycle }
func (w *World) SevenSignsHighestCabal() int     { return w.ss().Winner }
func (w *World) SevenSignsSealOwner(seal int) int {
	if seal < 1 || seal > 3 {
		return 0
	}
	return w.ss().SealOwners[seal]
}

func (w *World) SevenSignsPlayerCabal(p scripting.PlayerRef) int { return w.ss().Cabals[p.ObjectID()] }
func (w *World) SevenSignsPlayerSeal(p scripting.PlayerRef) int  { return w.ss().Seals[p.ObjectID()] }
func (w *World) SevenSignsContribution(p scripting.PlayerRef) int {
	return w.ss().Contributions[p.ObjectID()]
}

// SevenSignsJoin records the cabal and seal.
func (w *World) SevenSignsJoin(p scripting.PlayerRef, cabal, seal int) {
	s := w.ss()
	s.Cabals[p.ObjectID()], s.Seals[p.ObjectID()] = cabal, seal
}

// SevenSignsContribute adds the stones' score within Config.MaxPlayerContrib.
func (w *World) SevenSignsContribute(p scripting.PlayerRef, blue, green, red int) int {
	s := w.ss()
	score := blue*3 + green*5 + red*10
	id := p.ObjectID()
	if s.Contributions[id]+score > s.Config.MaxPlayerContrib {
		return -1
	}
	s.Contributions[id] += score
	s.Rewards[id] += score
	s.Contributed = append(s.Contributed, [3]int{blue, green, red})
	return score
}

// SevenSignsTakeReward returns and clears the player's reward.
func (w *World) SevenSignsTakeReward(p scripting.PlayerRef) int {
	s := w.ss()
	n := s.Rewards[p.ObjectID()]
	delete(s.Rewards, p.ObjectID())
	return n
}

func (w *World) SevenSignsConfig() scripting.SevenSignsConfig { return w.ss().Config }

// SevenSignsDestroyItem takes items from the fake player.
func (w *World) SevenSignsDestroyItem(p scripting.PlayerRef, itemID int, count int64, _ bool) bool {
	pl := playerOf(p)
	if pl == nil || count <= 0 || pl.Items[itemID] < count {
		return false
	}
	pl.Items[itemID] -= count
	return true
}

// SevenSignsAddItem gives items to the fake player.
func (w *World) SevenSignsAddItem(p scripting.PlayerRef, itemID int, count int64, _ bool) bool {
	pl := playerOf(p)
	if pl == nil || count <= 0 {
		return false
	}
	if pl.Items == nil {
		pl.Items = map[int]int64{}
	}
	pl.Items[itemID] += count
	return true
}

// SevenSignsReduceAdena takes adena (57).
func (w *World) SevenSignsReduceAdena(p scripting.PlayerRef, count int64) bool {
	return w.SevenSignsDestroyItem(p, 57, count, true)
}

// SevenSignsSendDisappeared records the message.
func (w *World) SevenSignsSendDisappeared(_ scripting.PlayerRef, itemID int, count int64) {
	s := w.ss()
	s.Disappeared = append(s.Disappeared, [2]int64{int64(itemID), count})
}

// SevenSignsHasRoom is always true.
func (w *World) SevenSignsHasRoom(scripting.PlayerRef, int) bool { return true }

func (w *World) SevenSignsClanCastle(p scripting.PlayerRef) int { return w.ss().ClanCastles[p.ObjectID()] }
func (w *World) SevenSignsTicketCount(castleID int) int         { return w.ss().Tickets[castleID] }
func (w *World) SevenSignsSetTicketCount(castleID, n int)       { w.ss().Tickets[castleID] = n }
