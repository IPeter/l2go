package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// Soulshots and spiritshots (decision K14). The charge lives on the weapon
// item (inventory.Item.ChargedShot), so it goes with the weapon. The shots a
// player uses automatically are a set of item IDs in Mobius's map order,
// not saved. A shot is charged by using it, by equipping a weapon, after
// each auto-attack hit and by casting (RechargeShots, for Phase 7.3's
// skill hit and finalizer); a landed swing spends the soulshot, a skill the
// shot it used (skills.Skill.Activate).

// System messages of the shots (Mobius SystemMessageId).
const (
	sysMsgSoulshotGrade      serverpackets.SystemMessageID = 337  // The soulshot you are attempting to use does not match the grade of your equipped weapon.
	sysMsgNotEnoughSoulshots serverpackets.SystemMessageID = 338  // You do not have enough soulshots for that.
	sysMsgCannotUseSoulshots serverpackets.SystemMessageID = 339  // Cannot use soulshots.
	sysMsgSpiritsEnabled     serverpackets.SystemMessageID = 342  // Power of the spirits enabled.
	sysMsgSpiritshotGrade    serverpackets.SystemMessageID = 530  // The Spiritshot does not match the weapon's grade.
	sysMsgNotEnoughSpirits   serverpackets.SystemMessageID = 531  // You do not have enough Spiritshots for that.
	sysMsgCannotUseSpirits   serverpackets.SystemMessageID = 532  // You may not use Spiritshots.
	sysMsgManaEnabled        serverpackets.SystemMessageID = 533  // Power of Mana enabled.
	sysMsgAutoUseActivated   serverpackets.SystemMessageID = 1433 // The automatic use of $s1 has been activated.
	sysMsgAutoUseDeactivated serverpackets.SystemMessageID = 1434 // The automatic use of $s1 has been deactivated.
	sysMsgNoServitorAutoUse  serverpackets.SystemMessageID = 1676 // You do not have a servitor or pet and therefore cannot use the automatic-use function.
)

// The shot item handlers (Mobius handlers/items).
const (
	handlerSoulShots         = "SoulShots"
	handlerSpiritShot        = "SpiritShot"
	handlerBlessedSpiritShot = "BlessedSpiritShot"
)

// shotBroadcastRange is how far a soulshot's glow is shown
// (broadcastToSelfAndVisiblePlayersInRange, 600).
const shotBroadcastRange = 600

// autoShotSet is a player's automatic shots by item ID (Mobius
// _activeSoulShots), in map order. The zero value is empty.
type autoShotSet struct {
	on    map[int32]bool
	order chmOrder
}

func (a *autoShotSet) add(id int32) {
	if a.on == nil {
		a.on = make(map[int32]bool)
	}
	if !a.on[id] {
		a.on[id] = true
		a.order.put(int(id))
	}
}

// remove reports whether id was on.
func (a *autoShotSet) remove(id int32) bool {
	if !a.on[id] {
		return false
	}
	delete(a.on, id)
	a.order.remove(int(id))
	return true
}

func (a *autoShotSet) has(id int32) bool { return a.on[id] }

// list returns the item IDs in map order.
func (a *autoShotSet) list() []int32 {
	keys := a.order.keys()
	out := make([]int32, len(keys))
	for i, k := range keys {
		out[i] = int32(k) //nolint:gosec // item IDs
	}
	return out
}

// AutoShots returns the item IDs of the shots p uses automatically, in
// Mobius's order. Loop-only.
func (p *Player) AutoShots() []int32 { return p.itemSkill.autoShots.list() }

// activeWeaponInstance is Mobius getActiveWeaponInstance: the item in the
// right hand, nil for none.
func activeWeaponInstance(p *Player) *inventory.Item {
	if p.inv.Inventory == nil {
		return nil
	}
	return p.inv.Paperdoll(storage.PaperdollRHand)
}

// ChargedShot reports whether c has the shot charged: a player on its
// weapon, an NPC on itself. Loop-only.
func (s *Server) ChargedShot(c Creature, t skills.ShotType) bool {
	if sc := s.skillCreatureOf(c); sc != nil {
		return sc.ChargedShot(t)
	}
	return false
}

// SetChargedShot charges or spends a shot of c. Loop-only.
func (s *Server) SetChargedShot(c Creature, t skills.ShotType, charged bool) {
	if sc := s.skillCreatureOf(c); sc != nil {
		sc.SetChargedShot(t, charged)
	}
}

// playerChargedShot is Player.isChargedShot: the shot on the weapon in the
// right hand.
func playerChargedShot(p *Player, t skills.ShotType) bool {
	w := activeWeaponInstance(p)
	return w != nil && w.ChargedShot(uint8(t))
}

// setPlayerChargedShot is Player.setChargedShot: nothing without a weapon.
func setPlayerChargedShot(p *Player, t skills.ShotType, charged bool) {
	if w := activeWeaponInstance(p); w != nil {
		w.SetChargedShot(uint8(t), charged)
	}
}

// RechargeShots is Player.rechargeShots: each automatic shot still in the
// inventory runs its handler (spiritshots with magic, soulshots with
// physical); one that is gone leaves the set. Casting (Phase 7.3) calls it
// at the skill hit and the finalizer with the skill's UseSoulShot and
// UseSpiritShot. NPCs have no automatic shots. Loop-only.
func (s *Server) RechargeShots(c Creature, physical, magic bool) {
	p, ok := c.(*Player)
	if !ok || p.inv.Inventory == nil {
		return
	}
	for _, id := range p.itemSkill.autoShots.list() {
		it := p.inv.ByItemID(id)
		if it == nil {
			p.itemSkill.autoShots.remove(id)
			continue
		}
		t := it.Template()
		if t.Etc == nil {
			continue
		}
		if (magic && t.DefaultAction == data.ActionSpiritshot) || (physical && t.DefaultAction == data.ActionSoulshot) {
			s.useShot(p, it)
		}
	}
}

// disableAutoShot is Player.disableAutoShot: a shot that was automatic is
// turned off, with ExAutoSoulShot and "The automatic use of S1 has been
// deactivated."; it reports whether it was on.
func (s *Server) disableAutoShot(p *Player, id int32) bool {
	if !p.itemSkill.autoShots.remove(id) {
		return false
	}
	p.Send(serverpackets.NewExAutoSoulShotPacket(serverpackets.ExAutoSoulShot{ItemID: id}))
	p.SendSystemMessage(sysMsgAutoUseDeactivated, s.itemNameParam(id))
	return true
}

// useShot runs the shot handler of it (SoulShots, SpiritShot or
// BlessedSpiritShot); other items do nothing.
func (s *Server) useShot(p *Player, it *inventory.Item) bool {
	switch it.Template().Etc.Handler {
	case handlerSoulShots:
		return s.useSoulShot(p, it)
	case handlerSpiritShot:
		return s.useSpiritShot(p, it, false)
	case handlerBlessedSpiritShot:
		return s.useSpiritShot(p, it, true)
	}
	return false
}

// normalSkills returns the item's NORMAL skills (ItemTemplate.getSkills).
func normalSkills(t *data.Item) []data.ItemSkill {
	var out []data.ItemSkill
	for _, sk := range t.Skills {
		if sk.Type == data.ItemSkillNormal {
			out = append(out, sk)
		}
	}
	return out
}

// shotUsePacket is the MagicSkillUse of a shot's glow: the shot's first skill
// on p, without cast time or reuse.
func shotUsePacket(p *Player, ref data.ItemSkill) []byte {
	pos := p.Position()
	return serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
		CasterID: p.ObjectID(), TargetID: p.ObjectID(),
		SkillID: int32(ref.ID), SkillLevel: int32(ref.Level), //nolint:gosec // skill IDs are small
		X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: pos.X, TargetY: pos.Y, TargetZ: pos.Z,
	})
}

// useSoulShot is Mobius SoulShots.onItemUse: the weapon needs soulshots and
// the shot's grade (the messages only when the shot is not automatic); a
// charged soulshot stays; the weapon's count (or, by its chance, its
// reduced count) is used up, else the automatic use goes off or "not
// enough"; then the weapon is charged, "Power of the spirits enabled." and
// the glow to everyone within 600.
func (s *Server) useSoulShot(p *Player, it *inventory.Item) bool {
	t := it.Template()
	refs := normalSkills(t)
	if len(refs) == 0 {
		s.log.Warn("game: SoulShots: the item has no skill", append(p.logArgs(), "item", it.ItemID())...)
		return false
	}
	auto := p.itemSkill.autoShots.has(it.ItemID())
	w := activeWeaponInstance(p)
	if w == nil || w.Template().Weapon == nil || w.Template().Weapon.SoulShots == 0 {
		if !auto {
			p.SendSystemMessage(sysMsgCannotUseSoulshots)
		}
		return false
	}
	if t.DefaultAction != data.ActionSoulshot || w.Template().Grade != t.Grade {
		if !auto {
			p.SendSystemMessage(sysMsgSoulshotGrade)
		}
		return false
	}
	if w.ChargedShot(uint8(skills.Soulshot)) {
		return false
	}
	wi := w.Template().Weapon
	count := int64(wi.SoulShots)
	if wi.ReducedSoulShot > 0 && s.rnd.Get(100) < wi.ReducedSoulShotChance {
		count = int64(wi.ReducedSoulShot)
	}
	if it.Count() < count || !s.DestroyItem(p, it, count, false) {
		if !s.disableAutoShot(p, it.ItemID()) {
			p.SendSystemMessage(sysMsgNotEnoughSoulshots)
		}
		return false
	}
	w.SetChargedShot(uint8(skills.Soulshot), true)
	p.SendSystemMessage(sysMsgSpiritsEnabled)
	s.world.BroadcastInRange(p, shotUsePacket(p, refs[0]), shotBroadcastRange, true)
	return true
}

// useSpiritShot is Mobius SpiritShot.onItemUse and, with blessed,
// BlessedSpiritShot.onItemUse: the weapon needs spiritshots (a blessed
// spiritshot can charge over a spiritshot), the grade must match, the
// weapon's count is used up, then "Power of Mana enabled.", the charge and
// the glow to everyone around.
func (s *Server) useSpiritShot(p *Player, it *inventory.Item, blessed bool) bool {
	t := it.Template()
	refs := normalSkills(t)
	if len(refs) == 0 {
		s.log.Warn("game: SpiritShot: the item has no skill", append(p.logArgs(), "item", it.ItemID())...)
		return false
	}
	auto := p.itemSkill.autoShots.has(it.ItemID())
	w := activeWeaponInstance(p)
	if w == nil || w.Template().Weapon == nil || w.Template().Weapon.SpiritShots == 0 {
		if !auto {
			p.SendSystemMessage(sysMsgCannotUseSpirits)
		}
		return false
	}
	kind := skills.Spiritshot
	if blessed {
		kind = skills.BlessedSpiritshot
	}
	if w.ChargedShot(uint8(kind)) {
		return false
	}
	if t.DefaultAction != data.ActionSpiritshot || w.Template().Grade != t.Grade {
		if !auto {
			p.SendSystemMessage(sysMsgSpiritshotGrade)
		}
		return false
	}
	count := int64(w.Template().Weapon.SpiritShots)
	if it.Count() < count || !s.DestroyItem(p, it, count, false) {
		if !s.disableAutoShot(p, it.ItemID()) {
			p.SendSystemMessage(sysMsgNotEnoughSpirits)
		}
		return false
	}
	w.SetChargedShot(uint8(kind), true)
	p.SendSystemMessage(sysMsgManaEnabled)
	s.world.Broadcast(p, shotUsePacket(p, refs[0]), true)
	return true
}

// beastShot reports the beast soulshots and spiritshots, which
// RequestAutoSoulShot turns on only for a summon.
func beastShot(id int32) bool { return (id >= 6645 && id <= 6647) || (id >= 20332 && id <= 20334) }

// spiritshotID reports the spiritshot item IDs of RequestAutoSoulShot's
// grade message.
func spiritshotID(id int32) bool {
	return (id >= 2509 && id <= 2514) || (id >= 3947 && id <= 3952) || id == 5790 || (id >= 22072 && id <= 22081)
}

// handleRequestAutoSoulShot handles RequestAutoSoulShot (D0 05) as Mobius
// does: on (1) a shot of the weapon's grade goes on; one of another grade
// (or without a weapon) gets the grade message and goes on anyway (a Mobius
// quirk); then "The automatic use of S1 has been activated." and both kinds
// recharge. Fishing shots are never automatic; beast shots need a summon,
// which does not exist yet. Off (0) turns it off with its message. A dead
// player or one without the item gets nothing.
func (s *Server) handleRequestAutoSoulShot(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAutoSoulShot(body)
	if err != nil {
		s.log.Debug("game: bad RequestAutoSoulShot", append(p.logArgs(), "err", err)...)
		return
	}
	if p.IsDead() || p.inv.Inventory == nil {
		return
	}
	it := p.inv.ByItemID(req.ItemID)
	if it == nil {
		return
	}
	switch req.Type {
	case clientpackets.AutoSoulShotOn:
		if req.ItemID >= 6535 && req.ItemID <= 6540 {
			return // fishing shots
		}
		if beastShot(req.ItemID) {
			p.SendSystemMessage(sysMsgNoServitorAutoUse)
			return
		}
		w := activeWeaponInstance(p)
		if w == nil || w.Template().Grade != it.Template().Grade {
			if spiritshotID(req.ItemID) {
				p.SendSystemMessage(sysMsgSpiritshotGrade)
			} else {
				p.SendSystemMessage(sysMsgSoulshotGrade)
			}
		}
		p.itemSkill.autoShots.add(req.ItemID)
		p.Send(serverpackets.NewExAutoSoulShotPacket(serverpackets.ExAutoSoulShot{ItemID: req.ItemID, Enabled: true}))
		p.SendSystemMessage(sysMsgAutoUseActivated, s.itemNameParam(req.ItemID))
		s.RechargeShots(p, true, true)
	case clientpackets.AutoSoulShotOff:
		p.itemSkill.autoShots.remove(req.ItemID)
		p.Send(serverpackets.NewExAutoSoulShotPacket(serverpackets.ExAutoSoulShot{ItemID: req.ItemID}))
		p.SendSystemMessage(sysMsgAutoUseDeactivated, s.itemNameParam(req.ItemID))
	}
}

// weaponEquipped is the part of useEquippableItem after a right-hand or
// two-hand weapon went on: both kinds of shot recharge.
func (s *Server) weaponEquipped(p *Player, it *inventory.Item) {
	switch it.Template().Part() {
	case data.BodyPartRHand, data.BodyPartLRHand:
		s.RechargeShots(p, true, true)
	}
}

// spendSoulshot is doAttack's discharge after a swing that hit.
func (s *Server) spendSoulshot(a Creature) { s.SetChargedShot(a, skills.Soulshot, false) }

// attackSoulshot is what an Attack shows of a's soulshot: whether it is
// charged, and the grade of the weapon (0 without one).
func (s *Server) attackSoulshot(a Creature) (bool, int32) {
	if !s.ChargedShot(a, skills.Soulshot) {
		return false, 0
	}
	if w := s.activeWeaponItem(a); w != nil {
		return true, int32(w.Grade.Level()) //nolint:gosec // 0-5
	}
	return true, 0
}
