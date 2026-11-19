package game

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// This file turns the game data into the stat engine's inputs. The stat
// engine does not import internal/data, so this is the one place where the
// two meet.

// newRules returns the stat rules: Mobius's defaults with the configured
// run speed cap, and the base stat bonus tables of the data.
func newRules(d *data.Data, maxRunSpeed int) (*stats.Rules, error) {
	bonus, err := bonusTables(d.StatBonuses)
	if err != nil {
		return nil, err
	}
	cfg := stats.DefaultConfig()
	cfg.MaxRunSpeed = maxRunSpeed
	r, err := stats.NewRules(cfg, bonus)
	if err != nil {
		return nil, fmt.Errorf("game: stat rules: %w", err)
	}
	return r, nil
}

// bonusTables copies statBonus.xml into the stat engine's tables.
func bonusTables(t data.StatBonusTable) (stats.BonusTables, error) {
	var b stats.BonusTables
	for _, col := range []struct {
		name string
		src  []float64
		dst  *[stats.BaseStatValues]float64
	}{
		{"STR", t.STR, &b.STR}, {"INT", t.INT, &b.INT}, {"DEX", t.DEX, &b.DEX},
		{"WIT", t.WIT, &b.WIT}, {"CON", t.CON, &b.CON}, {"MEN", t.MEN, &b.MEN},
	} {
		if len(col.src) != stats.BaseStatValues {
			return stats.BonusTables{}, fmt.Errorf("game: the %s bonus table has %d values, want %d", col.name, len(col.src), stats.BaseStatValues)
		}
		copy(col.dst[:], col.src)
	}
	return b, nil
}

// statsClassTemplates converts every class template of the data.
func statsClassTemplates(d *data.Data) (map[int]*stats.ClassTemplate, error) {
	out := make(map[int]*stats.ClassTemplate, len(d.Classes))
	for i := range d.Classes {
		c := &d.Classes[i]
		t := statsClassTemplate(c)
		if err := t.Validate(); err != nil {
			return nil, fmt.Errorf("game: class %d (%s): %w", c.ClassID, c.Name, err)
		}
		out[c.ClassID] = t
	}
	return out, nil
}

// statsClassTemplate converts a class template (Mobius PlayerTemplate). The
// per-level values stop at the first level the data lacks.
func statsClassTemplate(c *data.ClassTemplate) *stats.ClassTemplate {
	t := &stats.ClassTemplate{
		ClassID: c.ClassID,
		Race:    int(c.Race),
		Base: stats.BaseStats{
			STR: c.Stats.STR, CON: c.Stats.CON, DEX: c.Stats.DEX,
			INT: c.Stats.INT, WIT: c.Stats.WIT, MEN: c.Stats.MEN,
		},
		PAtk:        c.PAtk,
		MAtk:        c.MAtk,
		PAtkSpd:     c.PAtkSpd,
		MAtkSpd:     c.MAtkSpd,
		CritRate:    c.CritRate,
		AttackRange: c.AttackRange,
		PDef: stats.ArmorDefs{
			Chest: c.PDef.Chest, Legs: c.PDef.Legs, Head: c.PDef.Head, Feet: c.PDef.Feet,
			Gloves: c.PDef.Gloves, Underwear: c.PDef.Underwear, Cloak: c.PDef.Cloak,
		},
		MDef: stats.JewelDefs{
			REar: c.MDef.REar, LEar: c.MDef.LEar, RFinger: c.MDef.RFinger,
			LFinger: c.MDef.LFinger, Neck: c.MDef.Neck,
		},
		RunSpeed:      c.MoveSpeed.Run,
		WalkSpeed:     c.MoveSpeed.Walk,
		SwimRunSpeed:  c.MoveSpeed.FastSwim,
		SwimWalkSpeed: c.MoveSpeed.SlowSwim,
		Male:          stats.Collision{Radius: c.CollisionMale.Radius, Height: c.CollisionMale.Height},
		Female:        stats.Collision{Radius: c.CollisionFemale.Radius, Height: c.CollisionFemale.Height},
	}
	for level := 1; level <= c.MaxLevel(); level++ {
		l, ok := c.Level(level)
		if !ok {
			break
		}
		t.Levels = append(t.Levels, stats.LevelGain{
			HP: l.HP, MP: l.MP, CP: l.CP,
			HPRegen: l.HPRegen, MPRegen: l.MPRegen, CPRegen: l.CPRegen,
		})
	}
	return t
}

// statsItem returns an item template, enchanted to enchant, as the stat
// engine needs it. The data lists the enchant functions Mobius adds to
// weapons and armor with stats (op "enchant", order 0); they are left out
// here because stats.ItemModifiers computes them from the enchant level
// itself, at the same order 0 Mobius DocumentItem gives every item stat.
func statsItem(t *data.Item, enchant int) stats.Item {
	it := stats.Item{
		Type:     t.Type,
		BodyPart: t.BodyPart,
		Grade:    t.Grade.String(),
		Enchant:  enchant,
	}
	if t.Weapon != nil {
		it.WeaponType = t.Weapon.Type.String()
	}
	if t.Armor != nil {
		it.ArmorType = t.Armor.Type.String()
	}
	for _, st := range t.Stats {
		if st.Op == data.OpEnchant || st.Op == data.OpEnchantHP {
			continue
		}
		it.Stats = append(it.Stats, stats.ItemStat{Stat: st.Stat.String(), Value: st.Value})
	}
	return it
}
