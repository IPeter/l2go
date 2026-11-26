package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/IPeter/l2go/internal/augment"
	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// Augmentation (Plan 14, M11d). The rules are in internal/augment; this
// feature adapts them:
//
//   - augment.go: the data, the bonuses of worn augmented items (Mobius
//     Augmentation.applyBonus and removeBonus from the item skills
//     listener, item_skill_worn.go), the option trigger skills on hits and
//     skills, the Augment bypass of the blacksmiths, //augment, and the
//     hooks other features use (RemoveAugmentBonuses for a class change,
//     OnAugmentCheck for player states such as fishing).
//   - refine.go: the packets of the refine and cancel windows (D0 29-2E)
//     with the NPC check of decision X10.

func init() { registerFeature("augment", setupAugment) }

// augmentFeature is the feature's state on the Server (Server.augment).
// Loop-only.
type augmentFeature struct {
	// tables are the augmentation data; nil without them, and then nothing
	// can be augmented and worn augmentations give nothing.
	tables *augment.Tables
	cfg    augment.Config
	// rnd is the source of the augmentation rolls (the game's; tests
	// replace it).
	rnd rnd.Source
	// players holds the bonuses each player has on.
	players map[*Player]*playerAugment
	// checks fill in the player states other features own (OnAugmentCheck).
	checks []func(p *Player, st *augment.Player)
}

// playerAugment is the augmentation state of a player.
type playerAugment struct {
	// active are the augmentations whose bonuses are on, by item object ID
	// (Mobius AugmentationStatBonus._active).
	active map[int32]augment.Augmentation
	// triggers are the option trigger skills, by skill ID (Mobius
	// Creature._triggerSkills).
	triggers map[int]augmentTrigger
}

// augmentTrigger is an option's trigger skill (Mobius OptionSkillHolder).
type augmentTrigger struct {
	data.AugmentTrigger
	skill *skills.Skill
}

// setupAugment loads the augmentation data and registers the packets, the
// GM command and the hooks.
func setupAugment(s *Server) error {
	f := &s.augment
	f.cfg = augmentConfig(s.gs.Augment)
	f.rnd = s.rnd
	f.players = make(map[*Player]*playerAugment)
	if dir := s.settings.DataDir; dir != "" {
		switch d, err := data.LoadAugment(dir); {
		case err == nil:
			f.tables = augment.NewTables(d)
			s.log.Info("augmentation data loaded", "options", len(d.Options), "lifeStones", len(d.Refine.LifeStones), "npcs", len(d.Refine.Npcs))
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no augmentation data, nothing can be augmented", "file", filepath.Join(dir, data.AugmentRefineFile))
		default:
			return fmt.Errorf("augmentation data: %w", err)
		}
	}
	s.OnPlayerLeave(func(p *Player) { delete(f.players, p) })
	return errors.Join(
		s.RegisterEx(clientpackets.ExOpRequestConfirmTargetItem, s.handleConfirmTargetItem),
		s.RegisterEx(clientpackets.ExOpRequestConfirmRefinerItem, s.handleConfirmRefinerItem),
		s.RegisterEx(clientpackets.ExOpRequestConfirmGemStone, s.handleConfirmGemStone),
		s.RegisterEx(clientpackets.ExOpRequestRefine, s.handleRefine),
		s.RegisterEx(clientpackets.ExOpRequestConfirmCancelItem, s.handleConfirmCancelItem),
		s.RegisterEx(clientpackets.ExOpRequestRefineCancel, s.handleRefineCancel),
		s.RegisterGMCommand("augment", storage.AccessLevelAdmin, s.gmAugment),
	)
}

// augmentConfig is the config's augment section as internal/augment takes
// it; the defaults of config.DefaultAugment are augment.DefaultConfig.
func augmentConfig(c config.Augment) augment.Config {
	out := augment.Config{
		Blacklist:    slices.Clone(c.BlackList),
		AllowPvP:     c.AllowPvPItems,
		AllowTrade:   c.AllowTrade,
		AllowDestroy: c.AllowDestroy,
	}
	for i, chances := range c.GradeChances() {
		copy(out.GradeChances[i][:], chances)
	}
	return out
}

// UseAugmentData makes d the augmentation data. Loop-only (or before Run).
func (s *Server) UseAugmentData(d *data.Augment) { s.augment.tables = augment.NewTables(d) }

// OnAugmentCheck adds a hook that fills in a player state the augmentation
// checks (AbstractRefinePacket.isValid(Player)) and another feature owns,
// such as Fishing. Call it from a feature's setup.
func (s *Server) OnAugmentCheck(fn func(p *Player, st *augment.Player)) {
	s.augment.checks = append(s.augment.checks, fn)
}

// augmentPlayer returns p's state as the augmentation checks see it.
func (s *Server) augmentPlayer(p *Player) augment.Player {
	st := augment.Player{
		Level:                 p.Level(),
		StoreMode:             p.InStoreMode(),
		Trading:               s.Trading(p),
		Dead:                  p.IsDead(),
		Paralyzed:             p.skill != nil && (p.skill.admin.paralyzed || p.skill.el.IsAffected(skills.FlagParalyzed)),
		Sitting:               p.Sitting,
		CursedWeapon:          p.CursedWeaponEquipped(),
		ProcessingTransaction: s.ProcessingTransaction(p),
	}
	if es := s.enchant.players[p]; es != nil && es.scroll != 0 {
		st.Enchanting = true // Mobius isEnchanting: an enchant scroll is active
	}
	for _, fn := range s.augment.checks {
		fn(p, &st)
	}
	return st
}

// augmentStateOf returns p's augmentation state, made on first use.
func (s *Server) augmentStateOf(p *Player) *playerAugment {
	st := s.augment.players[p]
	if st == nil {
		st = &playerAugment{active: make(map[int32]augment.Augmentation), triggers: make(map[int]augmentTrigger)}
		s.augment.players[p] = st
	}
	return st
}

// augmentLocked reports whether an item may not be traded, sold or dropped
// for its augmentation (Mobius Item.isTradeable, isSellable and isDropable
// with AltAllowAugmentTrade off, which the config validation enforces;
// inventory.Item.Depositable keeps it out of the clan warehouse).
func augmentLocked(it *inventory.Item) bool { return it.Augmented() }

// weaponAugmentation is the augmentation of the right-hand item, as
// UserInfo and CharInfo show it (Mobius getPaperdollAugmentationId).
func (p *Player) weaponAugmentation() int32 {
	if p.inv.Inventory == nil {
		return 0
	}
	if it := p.inv.Paperdoll(storage.PaperdollRHand); it != nil {
		return it.Augmentation()
	}
	return 0
}

// augmentOn is Augmentation.applyBonus, called first by the worn item
// listener (ItemSkillsListener.notifyEquiped): each option's stat
// modifiers, its active and passive skills and its trigger skills, then
// SkillList per option. A bonus that is on already stays as it is. quiet
// sends nothing (loading).
func (s *Server) augmentOn(p *Player, it *inventory.Item, quiet bool) {
	t := s.augment.tables
	if t == nil || !it.Augmented() {
		return
	}
	st := s.augmentStateOf(p)
	if _, on := st.active[it.ObjectID()]; on {
		return
	}
	aug := augment.Augmentation(it.Augmentation())
	st.active[it.ObjectID()] = aug
	opts, missing := t.Options(aug)
	for _, id := range missing {
		s.log.Warn("game: an augmentation has an option the data does not have", append(p.logArgs(), "item", it.ItemID(), "option", id)...)
	}
	for _, o := range opts {
		s.optionOn(p, st, o, quiet)
	}
}

// augmentOff is Augmentation.removeBonus, called first by the worn item
// listener (ItemSkillsListener.notifyUnequiped): it takes off what
// augmentOn put on for the item.
func (s *Server) augmentOff(p *Player, it *inventory.Item, quiet bool) {
	st := s.augment.players[p]
	if st == nil || s.augment.tables == nil {
		return
	}
	aug, on := st.active[it.ObjectID()]
	if !on {
		return
	}
	delete(st.active, it.ObjectID())
	opts, _ := s.augment.tables.Options(aug)
	for _, o := range opts {
		s.optionOff(p, st, o, quiet)
	}
}

// optionOn is Options.apply.
func (s *Server) optionOn(p *Player, st *playerAugment, o *data.AugmentOption, quiet bool) {
	if len(o.Funcs) > 0 {
		mods, err := augment.Modifiers(o)
		if err == nil {
			err = p.stats.AddModifiers(mods...)
		}
		if err != nil {
			s.log.Warn("game: augmentation option stats refused", append(p.logArgs(), "option", o.ID, "err", err)...)
		}
	}
	for _, ref := range []*data.SkillRef{o.ActiveSkill, o.PassiveSkill} {
		if ref == nil {
			continue
		}
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			s.log.Warn("game: an augmentation option has a skill the data does not have", append(p.logArgs(), "option", o.ID, "skill", ref.ID, "level", ref.Level)...)
			continue
		}
		s.addOptionSkill(p, sk, quiet)
	}
	for _, tr := range o.Triggers {
		sk, ok := s.Skill(tr.ID, tr.Level)
		if !ok {
			s.log.Warn("game: an augmentation option has a skill the data does not have", append(p.logArgs(), "option", o.ID, "skill", tr.ID, "level", tr.Level)...)
			continue
		}
		st.triggers[sk.ID] = augmentTrigger{AugmentTrigger: tr, skill: sk}
	}
	if !quiet {
		p.SendSkillList()
	}
}

// addOptionSkill is Options.addSkill: the skill is learned (not saved); an
// active one keeps the reuse it has left, and SkillCoolTime follows.
func (s *Server) addOptionSkill(p *Player, sk *skills.Skill, quiet bool) {
	s.AddSkill(p, sk, false)
	if !sk.IsActive() {
		return
	}
	if left := s.SkillReuseLeft(p, sk); left > 0 {
		s.AddSkillReuse(p, sk, left)
		s.DisableSkill(p, sk, left)
	}
	if !quiet && p.InGame() {
		p.Send(s.SkillCoolTimePacket(p))
	}
}

// optionOff is Options.remove: the option's modifiers (removeStatsOwner),
// its active skill (its effect stays) and passive skill (its effect ends),
// its trigger skills, then SkillList.
func (s *Server) optionOff(p *Player, st *playerAugment, o *data.AugmentOption, quiet bool) {
	if len(o.Funcs) > 0 {
		p.stats.RemoveOwner(augment.OptionOwner(o.ID))
	}
	if o.ActiveSkill != nil {
		s.RemoveSkill(p, o.ActiveSkill.ID, false, false)
	}
	if o.PassiveSkill != nil {
		s.RemoveSkill(p, o.PassiveSkill.ID, false, true)
	}
	for _, tr := range o.Triggers {
		delete(st.triggers, tr.ID)
	}
	if !quiet {
		p.SendSkillList()
	}
}

// RemoveAugmentBonuses takes the bonuses of p's worn augmented items off,
// as Mobius Player.setActiveClass does before a class change. Mobius puts
// them on again only when an item is put on again; ApplyAugmentBonuses
// does it at once. The caller refreshes the stats. Loop-only.
func (s *Server) RemoveAugmentBonuses(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	for _, it := range p.inv.Items() {
		if it.Equipped() && it.Augmented() {
			s.augmentOff(p, it, true)
		}
	}
}

// ApplyAugmentBonuses puts the bonuses of p's worn augmented items on
// (those already on stay), quietly. The caller refreshes the stats and
// the skill list. Loop-only.
func (s *Server) ApplyAugmentBonuses(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	for _, it := range p.inv.Items() {
		if it.Equipped() && it.Augmented() {
			s.augmentOn(p, it, true)
		}
	}
}

// sortedTriggers returns p's trigger skills by skill ID. Mobius walks a
// hash map keyed by skill ID; the order only matters for two triggers at
// once.
func (s *Server) sortedTriggers(p *Player) []augmentTrigger {
	st := s.augment.players[p]
	if st == nil || len(st.triggers) == 0 {
		return nil
	}
	ids := make([]int, 0, len(st.triggers))
	for id := range st.triggers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]augmentTrigger, len(ids))
	for i, id := range ids {
		out[i] = st.triggers[id]
	}
	return out
}

// augmentHitTriggers is the trigger part of Creature.onHitTimer for a hit
// that did damage: attack skills on a normal hit and critical skills on a
// critical one fire with their chance at the target.
func (s *Server) augmentHitTriggers(a, t Creature, crit bool) {
	p, ok := a.(*Player)
	if !ok {
		return
	}
	target := s.skillCreatureOf(t)
	for _, tr := range s.sortedTriggers(p) {
		if augment.FiresOnHit(tr.AugmentTrigger, crit) && augment.Rolled(s.attack.rnd, tr.AugmentTrigger) {
			p.skill.MakeTriggerCast(tr.skill, target, false)
		}
	}
}

// augmentSkillTriggers is the trigger part of Creature.callSkill for each
// target of a skill that is not static: magic skills on a magic skill and
// attack skills on a physical one fire with their chance at the target.
func (s *Server) augmentSkillTriggers(c Creature, sk *skills.Skill, t Creature) {
	p, ok := c.(*Player)
	if !ok || sk.IsStatic() {
		return
	}
	var target skills.Creature
	for _, tr := range s.sortedTriggers(p) {
		if augment.FiresOnSkill(tr.AugmentTrigger, sk.IsMagic(), sk.IsPhysical()) && augment.Rolled(s.rnd, tr.AugmentTrigger) {
			if target == nil {
				target = s.skillCreatureOf(t)
			}
			p.skill.MakeTriggerCast(tr.skill, target, false)
		}
	}
}

// bypassAugment handles "Augment <n>" of the blacksmiths (Mobius
// handlers/bypass/npc/Augment): 1 opens the augmentation window, 2 the one
// that removes augmentations.
func (s *Server) bypassAugment(p *Player, command string) {
	if len(command) < 9 {
		return
	}
	switch command[8] {
	case '1':
		p.Send(serverpackets.NewExShowVariationMakeWindowPacket())
	case '2':
		p.Send(serverpackets.NewExShowVariationCancelWindowPacket())
	}
}

// sysMsgInvalidTarget is "Invalid target.".
const sysMsgInvalidTarget serverpackets.SystemMessageID = 109

// gmAugment is //augment <item id> <augmentation> (Mobius AdminAugment):
// the worn item of that template of the target (or the GM) gets the
// augmentation, an int of two options, replacing any it had. Unlike
// Mobius, which leaves the old bonuses on until the item comes off, the
// bonuses follow at once.
func (s *Server) gmAugment(p *Player, args []string) {
	if len(args) < 2 {
		p.SendMessage("Usage: //augment <item id> <option id>")
		return
	}
	target := p
	if p.Target != nil {
		tp, ok := p.Target.(*Player)
		if !ok {
			p.SendSystemMessage(sysMsgInvalidTarget)
			return
		}
		target = tp
	}
	itemID, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		p.SendMessage("No item with id " + args[0])
		return
	}
	var it *inventory.Item
	for slot := range storage.PaperdollSlot(storage.PaperdollTotalSlots) {
		if w := target.inv.Paperdoll(slot); w != nil && int64(w.ItemID()) == itemID {
			it = w
			break
		}
	}
	if it == nil {
		p.SendMessage("No equipped item with id " + args[0])
		return
	}
	aug, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		p.SendMessage("No option with id " + args[1])
		return
	}
	s.augmentOff(target, it, false)
	it.SetAugmentation(int32(aug)) //nolint:gosec // parsed as a 32-bit int
	s.augmentOn(target, it, false)
	s.itemsChanged(target, false, nil)
	s.SendItemList(target, false)
	target.BroadcastUserInfo()
}
