package social

import (
	"strconv"

	"github.com/IPeter/l2go/internal/rnd"
)

// LootMode is Mobius PartyDistributionType: how a party shares items. The
// value is the ID the packets carry.
type LootMode int

// The loot modes, with their packet IDs.
const (
	// FindersKeepers: the player who picks up or kills keeps the item.
	FindersKeepers LootMode = 0
	// Random: a random member with room and in range; spoil stays with the
	// sweeper.
	Random LootMode = 1
	// RandomIncludingSpoil: as Random, spoil included.
	RandomIncludingSpoil LootMode = 2
	// ByTurn: the next member in party order with room and in range; spoil
	// stays with the sweeper.
	ByTurn LootMode = 3
	// ByTurnIncludingSpoil: as ByTurn, spoil included.
	ByTurnIncludingSpoil LootMode = 4
)

// lootSysStrings are the sysstring IDs of the modes (PartyDistributionType).
var lootSysStrings = [...]int{487, 488, 798, 799, 800}

// Valid reports whether m is one of the five modes
// (PartyDistributionType.findById).
func (m LootMode) Valid() bool { return m >= FindersKeepers && m <= ByTurnIncludingSpoil }

// SysString is the mode's sysstring ID, used in system messages; 0 for an
// invalid mode.
func (m LootMode) SysString() int {
	if !m.Valid() {
		return 0
	}
	return lootSysStrings[m]
}

// String names the mode as Mobius does.
func (m LootMode) String() string {
	switch m {
	case FindersKeepers:
		return "FINDERS_KEEPERS"
	case Random:
		return "RANDOM"
	case RandomIncludingSpoil:
		return "RANDOM_INCLUDING_SPOIL"
	case ByTurn:
		return "BY_TURN"
	case ByTurnIncludingSpoil:
		return "BY_TURN_INCLUDING_SPOIL"
	}
	return "LootMode(" + strconv.Itoa(int(m)) + ")"
}

// LootCandidate is a party member as the loot rules see it, in party order
// (leader first).
type LootCandidate struct {
	// HasSpace is Inventory.validateCapacityByItemId for the item: the
	// member has a free slot, or already holds a stack of it.
	HasSpace bool
	// InRange is true within AltPartyRange (3D) of the reference: the
	// monster for auto-loot, the picker for a picked-up item.
	InRange bool
}

func (c LootCandidate) ok() bool { return c.HasSpace && c.InRange }

// Loot is a party's loot state: its mode and the by-turn position
// (Party._itemLastLoot).
type Loot struct {
	// Mode is the party's loot mode, chosen at the first invitation.
	Mode LootMode
	// Turn is the index of the member who got the last by-turn item. It
	// starts at 0, so the first by-turn item goes to the second member, as
	// in Mobius. It is not reset when members leave.
	Turn int
}

// Looter is Party.getActualLooter: the index in members of the member who
// gets an item, or -1 when it stays with the player who picked it up or
// made the kill (Mobius's fallback when the mode keeps items or nobody
// qualifies). Adena never comes here: it is split (SplitAdena). Random
// modes draw src.Get(qualifying members) when anyone qualifies; by-turn
// modes move Turn on.
func (l *Loot) Looter(src rnd.Source, members []LootCandidate, spoil bool) int {
	switch l.Mode {
	case Random:
		if !spoil {
			return randomLooter(src, members)
		}
	case RandomIncludingSpoil:
		return randomLooter(src, members)
	case ByTurn:
		if !spoil {
			return l.nextLooter(members)
		}
	case ByTurnIncludingSpoil:
		return l.nextLooter(members)
	}
	return -1
}

// randomLooter is Party.getCheckedRandomMember.
func randomLooter(src rnd.Source, members []LootCandidate) int {
	var ok []int
	for i, m := range members {
		if m.ok() {
			ok = append(ok, i)
		}
	}
	if len(ok) == 0 {
		return -1
	}
	return ok[src.Get(len(ok))]
}

// nextLooter is Party.getCheckedNextLooter: up to one round of the party
// from the member after the last one, the first who qualifies.
func (l *Loot) nextLooter(members []LootCandidate) int {
	n := len(members)
	for range n {
		l.Turn++
		if l.Turn >= n {
			l.Turn = 0
		}
		if members[l.Turn].ok() {
			return l.Turn
		}
	}
	return -1
}

// SplitAdena is Party.distributeAdena: adena split evenly among the members
// in range (inRange, in party order), the remainder 1 each to consecutive
// members from a random one (src.Get(in range count), drawn only when there
// is a remainder). The result is aligned with inRange; members out of range
// get 0. Nothing is given (all 0, no draw) without adena or anyone in range.
func SplitAdena(src rnd.Source, adena int64, inRange []bool) []int64 {
	out := make([]int64, len(inRange))
	var idx []int
	for i, in := range inRange {
		if in {
			idx = append(idx, i)
		}
	}
	n := int64(len(idx))
	if n == 0 || adena <= 0 {
		return out
	}
	base, rem := adena/n, adena%n
	for _, i := range idx {
		out[i] = base
	}
	if rem > 0 {
		start := int64(src.Get(len(idx)))
		for k := range rem {
			out[idx[(start+k)%n]]++
		}
	}
	return out
}
