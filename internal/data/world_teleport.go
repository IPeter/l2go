package data

import "fmt"

// Teleport list types (TeleportList.Type), the Mobius TeleportType enum.
const (
	TeleportNormal      = "NORMAL"
	TeleportNoblesToken = "NOBLES_TOKEN"
	TeleportNoblesAdena = "NOBLES_ADENA"
	TeleportOther       = "OTHER"
)

// DefaultTeleportFeeItem is the fee item of a location without one: adena.
const DefaultTeleportFeeItem = 57

var teleportTypes = setOf(TeleportNormal, TeleportNoblesToken, TeleportNoblesAdena, TeleportOther)

// Teleporter holds the teleport lists of one NPC, from the Mobius
// teleporter data (dist/game/data/teleporters/**/*.xml).
type Teleporter struct {
	NpcID int            `json:"npcId"`
	Lists []TeleportList `json:"lists"`
}

// TeleportList is one <teleport> list of a teleporter. The bypass
// "teleport <list name> <location index>" picks a location.
//
// Mobius rules (TeleportHolder): a NORMAL list charges its fees only above
// the free teleport level (MaxFreeTeleportLevel) or on a subclass, at half
// price from 20:00 on Mondays and Tuesdays; the other types always charge.
// The NOBLES_* lists are for nobles only.
type TeleportList struct {
	// Name identifies the list in bypasses; it defaults to the type name.
	Name string `json:"name"`
	// Type is one of the Teleport* constants.
	Type string `json:"type"`
	// File is the source file below dist/game/data/teleporters.
	File      string             `json:"file"`
	Locations []TeleportLocation `json:"locations"`
}

// TeleportLocation is one destination of a list; its index in the list is
// its ID in bypasses.
type TeleportLocation struct {
	// Name is the text of the link; Mobius allows it to be missing.
	Name string `json:"name,omitempty"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Z    int    `json:"z"`
	// FeeItemID is the item the fee is paid with (DefaultTeleportFeeItem,
	// adena, by default; a negative ID is a Mobius pseudo item such as PC
	// café points).
	FeeItemID int `json:"feeItemId"`
	FeeCount  int `json:"feeCount,omitempty"`
	// CastleIDs are the castles whose siege blocks this destination.
	CastleIDs []int `json:"castleIds,omitempty"`
}

// TeleportLists returns the teleport lists of an NPC, or nil.
func (w *World) TeleportLists(npcID int) []TeleportList {
	i, ok := w.teleporters[npcID]
	if !ok {
		return nil
	}
	return w.Teleporters[i].Lists
}

// TeleportList returns the teleport list of an NPC by name.
func (w *World) TeleportList(npcID int, name string) (*TeleportList, bool) {
	i, ok := w.teleporters[npcID]
	if !ok {
		return nil, false
	}
	lists := w.Teleporters[i].Lists
	for j := range lists {
		if lists[j].Name == name {
			return &lists[j], true
		}
	}
	return nil, false
}

// checkTeleporters checks the teleporters and indexes them:
//
//   - NPC IDs are unique and name existing NPCs;
//   - each has lists, with unique non-empty names and a known type;
//   - fees are not negative and castle IDs are positive. Fee items are
//     only checked to be numbers (they are; the item data is checked by
//     its own domain).
func (w *World) checkTeleporters(p *problems) {
	for i := range w.Teleporters {
		t := &w.Teleporters[i]
		where := fmt.Sprintf("%s: NPC %d", TeleportersFile, t.NpcID)
		if _, dup := w.teleporters[t.NpcID]; dup {
			p.addf("%s: duplicate NPC ID", where)
			continue
		}
		w.teleporters[t.NpcID] = i
		if _, ok := w.npcs[t.NpcID]; !ok {
			p.addf("%s: does not exist in %s", where, NpcsFile)
		}
		if len(t.Lists) == 0 {
			p.addf("%s: no teleport lists", where)
		}
		names := make(map[string]bool, len(t.Lists))
		for _, l := range t.Lists {
			at := fmt.Sprintf("%s: list %q (%s)", where, l.Name, l.File)
			if l.Name == "" {
				p.addf("%s: no name", at)
			} else if names[l.Name] {
				p.addf("%s: duplicate list name", at)
			}
			names[l.Name] = true
			if !teleportTypes[l.Type] {
				p.addf("%s: unknown type %q", at, l.Type)
			}
			for j, loc := range l.Locations {
				if loc.FeeCount < 0 {
					p.addf("%s: location %d (%q): fee %d is negative", at, j, loc.Name, loc.FeeCount)
				}
				for _, c := range loc.CastleIDs {
					if c <= 0 {
						p.addf("%s: location %d (%q): castle ID %d must be positive", at, j, loc.Name, c)
					}
				}
			}
		}
	}
}
