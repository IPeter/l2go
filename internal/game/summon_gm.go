package game

import (
	"errors"
	"strconv"

	"github.com/IPeter/l2go/internal/storage"
)

// The summon GM commands of Mobius AdminEditChar the demo needs (X9):
// //fullfood fills the targeted pet's meal, //unsummon sends the targeted
// summon away, //summon_setlvl <level> sets the targeted pet's level through
// its exp.

func init() { registerFeature("summongm", setupSummonGM) }

func setupSummonGM(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("fullfood", admin, s.gmFullFood),
		s.RegisterGMCommand("unsummon", admin, s.gmUnsummon),
		s.RegisterGMCommand("summon_setlvl", admin, s.gmSummonSetLevel),
	)
}

// gmTargetSummon returns the summon p targets, nil for none.
func gmTargetSummon(p *Player) *Summon {
	if n, ok := p.Target.(*Npc); ok {
		return n.summon
	}
	return nil
}

// gmFullFood is admin_fullfood: the targeted pet's meal is full.
func (s *Server) gmFullFood(p *Player, _ []string) {
	sm := gmTargetSummon(p)
	if sm == nil || sm.pet == nil {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	sm.pet.fed = sm.pet.maxMeal()
	s.summonFedChanged(sm)
}

// gmUnsummon is admin_unsummon: the targeted summon goes back.
func (s *Server) gmUnsummon(p *Player, _ []string) {
	sm := gmTargetSummon(p)
	if sm == nil {
		p.SendMessage("Usable only with Pets/Summons")
		return
	}
	if !sm.npc.Dead && sm.npc.InWorld() {
		s.unsummonNow(sm)
	}
}

// gmSummonSetLevel is admin_summon_setlvl <level>: the targeted pet gets the
// exp of the level.
func (s *Server) gmSummonSetLevel(p *Player, args []string) {
	sm := gmTargetSummon(p)
	if sm == nil || sm.pet == nil {
		p.SendMessage("Usable only with Pets")
		return
	}
	if len(args) != 1 {
		return
	}
	level, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	s.petAddExp(sm, s.petCurve(sm).ExpFor(level)-sm.pet.exp)
}
