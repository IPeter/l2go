package manor

import "github.com/IPeter/l2go/internal/rnd"

// The Strong type passives (skills 4303-4310) multiply a harvest by 2 to 9
// (Attackable.setSeeded).
const (
	strongTypeFirst = 4303
	strongTypeLast  = 4310
)

// SowChance is the Sow effect's calcSuccess chance in percent: 90 (20 for an
// alternative seed), less 5 per level the target is outside the seed's level
// ± 5, less 5 per level the player is more than 5 levels from the target.
//
// Mobius quirk, kept: its Math.max(basicSuccess, 1) throws the result away,
// so the chance can fall to 0 or below (never sown).
func SowChance(s *Seed, playerLevel, targetLevel int) int {
	minLevel, maxLevel := s.Level-5, s.Level+5
	chance := 90
	if s.Alternative {
		chance = 20
	}
	if targetLevel < minLevel {
		chance -= 5 * (minLevel - targetLevel)
	}
	if targetLevel > maxLevel {
		chance -= 5 * (targetLevel - maxLevel)
	}
	if diff := abs(playerLevel - targetLevel); diff > 5 {
		chance -= 5 * (diff - 5)
	}
	return chance
}

// SowSucceeds rolls the sowing: Rnd.get(99) < chance.
func SowSucceeds(r rnd.Source, s *Seed, playerLevel, targetLevel int) bool {
	return r.Get(99) < SowChance(s, playerLevel, targetLevel)
}

// HarvestChance is the Harvesting effect's calcSuccess chance in percent:
// 100, less 5 per level the player is more than 5 levels from the target, at
// least 1.
func HarvestChance(playerLevel, targetLevel int) int {
	chance := 100
	if diff := abs(playerLevel - targetLevel); diff > 5 {
		chance -= (diff - 5) * 5
	}
	return max(chance, 1)
}

// HarvestSucceeds rolls the harvest: Rnd.get(99) < chance.
func HarvestSucceeds(r rnd.Source, playerLevel, targetLevel int) bool {
	return r.Get(99) < HarvestChance(playerLevel, targetLevel)
}

// HarvestCount is how many crops a sown monster gives (Attackable.setSeeded
// with the seeder): 1, times 2-9 for each Strong type passive (4303-4310) of
// the monster, plus the levels the monster is more than 5 above the seed,
// times the drop rate.
func HarvestCount(s *Seed, monsterLevel int, monsterSkills []int, dropRate int) int64 {
	count := 1
	for _, id := range monsterSkills {
		if id >= strongTypeFirst && id <= strongTypeLast {
			count *= id - strongTypeFirst + 2
		}
	}
	if diff := monsterLevel - s.Level - 5; diff > 0 {
		count += diff
	}
	return int64(count) * int64(dropRate)
}

// KeptWhenSeeded reports whether a death drop stays in the drops of a sown
// monster (NpcTemplate.calculateDrops): adena and the seal stones only.
func KeptWhenSeeded(itemID int) bool {
	switch itemID {
	case 57, 6360, 6361, 6362:
		return true
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
