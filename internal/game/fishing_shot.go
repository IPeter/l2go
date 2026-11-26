package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/skills"
)

// handlerFishShots is the item handler of the fishing shots (6535-6540).
const handlerFishShots = "FishShots"

// useFishShot is Mobius FishShots.onItemUse: only with a fishing rod in
// hand that is not charged yet; a shot of another grade gets "That is the
// wrong grade of soulshot for that fishing pole."; otherwise the rod is
// charged (Pumping and Reeling double their damage and use it up), one shot
// is used, and everyone around sees the glow.
func (s *Server) useFishShot(p *Player, it *inventory.Item) bool {
	w := activeWeaponInstance(p)
	if w == nil || w.Template().Weapon == nil || w.Template().Weapon.Type != data.WeaponFishingrod {
		return false
	}
	if playerChargedShot(p, skills.FishSoulshot) {
		return false
	}
	t := it.Template()
	refs := normalSkills(t)
	if len(refs) == 0 {
		s.log.Warn("game: FishShots: the item has no skill", append(p.logArgs(), "item", it.ItemID())...)
		return false
	}
	if t.DefaultAction != data.ActionFishingshot || w.Template().Grade != t.Grade {
		sendFishingMessage(p, fishing.MsgWrongShotGrade, 0)
		return false
	}
	if it.Count() < 1 {
		return false
	}
	setPlayerChargedShot(p, skills.FishSoulshot, true)
	s.DestroyItem(p, it, 1, false)
	s.world.Broadcast(p, shotUsePacket(p, refs[0]), true)
	return true
}
