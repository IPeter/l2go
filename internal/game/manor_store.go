package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/manor"
	"github.com/IPeter/l2go/internal/storage"
)

// manorKey is the saver's key of a castle's manor document; castle 0 is the
// manor's mode (persist.go).
func manorKey(castleID int32) saveKey { return saveKey(int64(castleID) | 21<<40) }

// manorTouches adds the keys of the manor documents a batch writes.
func manorTouches(b *storage.Batch, add func(saveKey)) {
	for _, c := range b.ManorCastles {
		add(manorKey(c.CastleID))
	}
	if b.ManorState != nil {
		add(manorKey(0))
	}
}

// mergeManor adds the manor of b, written after a, to a: a castle's lists of
// b replace a's, and b's mode replaces a's.
func mergeManor(a, b *storage.Batch) {
	for _, c := range b.ManorCastles {
		if i := slices.IndexFunc(a.ManorCastles, func(x storage.ManorCastle) bool { return x.CastleID == c.CastleID }); i >= 0 {
			a.ManorCastles[i] = c
		} else {
			a.ManorCastles = append(a.ManorCastles, c)
		}
	}
	if b.ManorState != nil {
		st := *b.ManorState
		a.ManorState = &st
	}
}

// manorCastleDoc returns a castle's stored manor lists.
func manorCastleDoc(id int32, c *manor.Castle) storage.ManorCastle {
	doc := storage.ManorCastle{CastleID: id}
	seeds := func(list []manor.Production) []storage.SeedProduction {
		var out []storage.SeedProduction
		for _, p := range list {
			out = append(out, storage.SeedProduction{SeedID: int32(p.SeedID), Amount: p.Amount, Price: p.Price, StartAmount: p.StartAmount}) //nolint:gosec // item IDs
		}
		return out
	}
	crops := func(list []manor.Procure) []storage.CropProcure {
		var out []storage.CropProcure
		for _, p := range list {
			out = append(out, storage.CropProcure{CropID: int32(p.CropID), Amount: p.Amount, Price: p.Price, StartAmount: p.StartAmount, RewardType: int32(p.Reward)}) //nolint:gosec // item IDs, 1-2
		}
		return out
	}
	doc.Production, doc.ProductionNext = seeds(c.Production), seeds(c.ProductionNext)
	doc.Procure, doc.ProcureNext = crops(c.Procure), crops(c.ProcureNext)
	return doc
}

// manorCastleOf converts stored lists, leaving out the seeds and crops the
// seed table does not know (Mobius loadDb warns and skips them). It returns
// the IDs it left out.
func manorCastleOf(doc storage.ManorCastle, t *manor.Table) (manor.Castle, []int32) {
	var c manor.Castle
	var unknown []int32
	seeds := func(list []storage.SeedProduction) []manor.Production {
		var out []manor.Production
		for _, p := range list {
			if t.Seed(int(p.SeedID)) == nil {
				unknown = append(unknown, p.SeedID)
				continue
			}
			out = append(out, manor.Production{SeedID: int(p.SeedID), Amount: p.Amount, Price: p.Price, StartAmount: p.StartAmount})
		}
		return out
	}
	crops := func(list []storage.CropProcure) []manor.Procure {
		var out []manor.Procure
		for _, p := range list {
			if !t.IsCrop(int(p.CropID)) {
				unknown = append(unknown, p.CropID)
				continue
			}
			out = append(out, manor.Procure{CropID: int(p.CropID), Amount: p.Amount, Price: p.Price, StartAmount: p.StartAmount, Reward: int(p.RewardType)})
		}
		return out
	}
	c.Production, c.ProductionNext = seeds(doc.Production), seeds(doc.ProductionNext)
	c.Procure, c.ProcureNext = crops(doc.Procure), crops(doc.ProcureNext)
	return c, unknown
}
