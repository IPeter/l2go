package game

import (
	"context"
	"errors"
	"math"

	"github.com/IPeter/l2go/internal/storage"
)

// Summon persistence (Plan 14 M11f): the pets keyed by their collars, the
// servitor to bring back and the servitors' saved effects per character.
// The state lives on the loop; storage.SummonStore writes go through a
// background writer in the order they were made (the W13 pattern), and the
// loads happen at character select, off the loop, or when a collar is used.

// The Mobius defaults of the summon settings (Player.ini; decision W16).
const (
	restorePetOnReconnect      = true // RestorePetOnReconnect
	restoreServitorOnReconnect = true // RestoreServitorOnReconnect
	summonStoreSkillCooltime   = true // SummonStoreSkillCooltime
)

// javaRound is Java's Math.round(double).
func javaRound(v float64) int64 {
	if v != v {
		return 0
	}
	return int64(math.Floor(v + 0.5))
}

// summonLoad is what SelectCharacter loads of a character's summons.
type summonLoad struct {
	saved storage.OwnerSummons
	pet   *storage.Pet
	pets  map[int32]*storage.Pet
}

// loadSummons loads p's summon state off the loop, at character select:
// the stored servitor and effects, and the pet to bring back. Failures are
// logged and leave nothing to restore.
func (s *Server) loadSummons(ctx context.Context, p *Player) summonLoad {
	out := summonLoad{saved: storage.OwnerSummons{OwnerID: p.ObjectID()}}
	store := s.summons.store
	if store == nil {
		return out
	}
	saved, err := store.LoadOwnerSummons(ctx, p.ObjectID())
	if err != nil {
		s.log.Warn("game: cannot load the summons", append(p.logArgs(), "err", err)...)
		return out
	}
	out.saved = saved
	// The pets of the collars the player carries (Mobius loads a pet's row
	// when its collar is used).
	out.pets = make(map[int32]*storage.Pet)
	for _, id := range s.collarIDs(p) {
		pet, err := store.LoadPet(ctx, id)
		switch {
		case err == nil:
			out.pets[id] = &pet
		case !errors.Is(err, storage.ErrNotFound):
			s.log.Warn("game: cannot load a pet", append(p.logArgs(), "collar", id, "err", err)...)
		}
	}
	if pet := out.pets[saved.RestorePet]; pet != nil && pet.Restore {
		out.pet = pet
	}
	return out
}

// collarIDs returns the object IDs of the pet collars p carries. It reads
// the player's inventory before the player is on the loop.
func (s *Server) collarIDs(p *Player) []int32 {
	d := s.summons.data
	if d == nil || p.inv.Inventory == nil {
		return nil
	}
	var out []int32
	for _, it := range p.inv.Items() {
		if d.IsPetItem(int(it.ItemID())) {
			out = append(out, it.ObjectID())
		}
	}
	return out
}

// summonsSelected keeps what loadSummons loaded, on the loop.
func (s *Server) summonsSelected(p *Player, l summonLoad) {
	o := s.summonOwnerOf(p)
	o.saved = l.saved
	o.saved.OwnerID = p.ObjectID()
	o.restorePet = l.pet
	o.pets = l.pets
}

// writeOwnerSummons writes p's summon state in the background.
func (s *Server) writeOwnerSummons(p *Player) {
	f := &s.summons
	if f.store == nil {
		return
	}
	o := s.summonOwnerOf(p)
	saved := o.saved
	saved.Effects = append([]storage.ServitorEffects(nil), saved.Effects...)
	if saved.Servitor != nil {
		sv := *saved.Servitor
		saved.Servitor = &sv
	}
	f.writer.enqueue("summons", func(ctx context.Context) error { return f.store.SaveOwnerSummons(ctx, saved) })
}

// writePet writes a pet in the background.
func (s *Server) writePet(pet storage.Pet) {
	f := &s.summons
	if f.store == nil {
		return
	}
	pet.Effects = append([]storage.SavedSkill(nil), pet.Effects...)
	f.writer.enqueue("pet", func(ctx context.Context) error { return f.store.SavePet(ctx, pet) })
}

// deletePet deletes a pet's stored state in the background.
func (s *Server) deletePet(controlID int32) {
	f := &s.summons
	if f.store == nil {
		return
	}
	f.writer.enqueue("pet", func(ctx context.Context) error { return f.store.DeletePet(ctx, controlID) })
}

// summonStoreMe is Mobius storeMe: a servitor's restore entry, a pet's row.
func (s *Server) summonStoreMe(sm *Summon) {
	if sm.pet != nil {
		s.petStoreMe(sm) // pet.go
		return
	}
	s.servitorStore(sm)
}

// summonStoreEffects is Mobius storeEffect(store).
func (s *Server) summonStoreEffects(sm *Summon, store bool) {
	if sm.pet != nil {
		s.petStoreEffects(sm, store) // pet.go
		return
	}
	s.servitorStoreEffects(sm, store)
}

// WaitSummonWrites waits until the summon writes handed over so far are
// done. For tests and the shutdown.
func (s *Server) WaitSummonWrites(ctx context.Context) error { return s.summons.writer.wait(ctx) }
