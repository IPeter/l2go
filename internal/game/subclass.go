package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/subclass"
)

// Subclasses, noblesse and skill enchanting (M11e, Plan 14). The rules are
// in internal/subclass; this feature adapts them:
//
//   - subclass.go: the feature, the class-index migration at load, the
//     subclass state of a player and the hooks of other features;
//   - subclass_switch.go: adding, replacing and switching a subclass
//     (Mobius Player.addSubClass, modifySubClass, setActiveClass);
//   - subclass_vm.go: the village master's "Subclass" pages;
//   - noble.go: noblesse (Player.setNoble);
//   - enchantskill.go: skill enchanting at the trainers.
//
// The class-index data (decision X3): skills, saved effects, shortcuts and
// hennas carry the class index in storage; the player keeps the rows of the
// inactive classes as loaded and the active class's live. ClassID, Level,
// Exp and SP of the character are the active class's; the base class's
// progress waits in BaseLevel, BaseExp and BaseSP while a subclass is
// active, each subclass's in its Subclasses entry.

func init() {
	registerFeature("subclass", setupSubclass)
	registerVillageMasterCommand("Subclass", (*Server).vmSubclass) // subclass_vm.go
	// The quests these features gate (decision Q10): subclasses and
	// noblesse now, and the grand bosses Fate's Whisper needs (Baium, M10).
	questFeatures["subclass"] = true
	questFeatures["noblesse"] = true
	questFeatures["grand_bosses"] = true
}

// subclassFeature is the subclass state on the Server (Server.subclass).
type subclassFeature struct {
	cfg subclass.Config
	// enchant is the skill enchant data, nil without it.
	enchant *data.EnchantSkillTree
	// rand is the source of the enchant roll: the game's (decision X7);
	// tests script it alone.
	rand  subclass.Rand
	hooks SubclassHooks
}

// SubclassHooks are the parts of a class switch that other features own
// (Mobius Player.setActiveClass and modifySubClass, decision X3). Each
// feature sets its hook in its setup (Server.SubclassHooks); a nil hook is
// skipped. The augmentation (RemoveAugmentBonuses), summon (servitors),
// cubic (StopCubics) and henna (SwapHennas) features are called directly,
// and the crafting feature's dwarven recipe book follows the class index by
// itself.
type SubclassHooks struct {
	// LeaveOlympiad takes p off the Olympiad's waiting list before a
	// subclass page (Olympiad, M11h; Mobius Olympiad.unRegisterNoble).
	LeaveOlympiad func(p *Player)
}

// SubclassHooks returns the hooks of the class switch for other features to
// set while the server is built. Not safe once it runs.
func (s *Server) SubclassHooks() *SubclassHooks { return &s.subclass.hooks }

var _ scripting.PlayerSubclass = (*Player)(nil)

// setupSubclass loads the skill enchant data and registers the enchant
// packets.
func setupSubclass(s *Server) error {
	f := &s.subclass
	f.cfg = subclass.DefaultConfig()
	f.rand = s.rnd
	if dir := s.settings.DataDir; dir != "" {
		switch t, err := data.LoadEnchantSkillTree(dir); {
		case err == nil:
			f.enchant = t
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no skill enchant data, no skill can be enchanted", "file", filepath.Join(dir, data.EnchantSkillTreeFile))
		default:
			return fmt.Errorf("skill enchant data: %w", err)
		}
	}
	return errors.Join(
		s.RegisterEx(clientpackets.ExOpRequestExEnchantSkillInfo, s.handleEnchantSkillInfo), // enchantskill.go
		s.RegisterEx(clientpackets.ExOpRequestExEnchantSkill, s.handleEnchantSkill),
	)
}

// migrateCharacter brings a loaded character to the subclass layout
// (subclass.Migrate): the legacy noble variable becomes the flag, the noble
// skills leave the stored skills, a lost subclass falls back to the base
// class. It runs at character selection before the player is built; a
// change is logged and written with the next save. Safe on any goroutine.
func (s *Server) migrateCharacter(c *storage.Character) {
	if subclass.Migrate(c, s.nobleSkillIDs()) {
		s.log.Info("game: character moved to the subclass layout", "player", c.Name, "objectId", c.ObjectID,
			"classIndex", c.ClassIndex, "class", c.ClassID, "baseClass", c.BaseClassID, "noble", c.Noble)
	}
}

// loadSubclass is the part of Mobius Player.load the subclass feature does
// on the loop at character selection, before the skills: the dyes of the
// active class (the henna feature loads the base class's) and the noble
// skills (setNoble(true) comes before restoreSkills).
func (s *Server) loadSubclass(p *Player) {
	if idx := p.classIndex(); idx != baseClassIndex {
		s.SwapHennas(p, idx) // henna.go: no packet before EnterWorld
	}
	if p.char.Noble {
		s.giveNobleSkills(p)
	}
}

// storeSubclass puts the live progress of an active subclass into its entry
// of a snapshot (Mobius storeCharSub): class, level, exp and SP.
func (p *Player) storeSubclass(c *storage.Character) {
	if c.ClassIndex == baseClassIndex || len(c.Subclasses) == 0 {
		return
	}
	subs := slices.Clone(c.Subclasses)
	for i := range subs {
		if subs[i].ClassIndex == c.ClassIndex {
			subs[i].ClassID, subs[i].Level, subs[i].Exp, subs[i].SP = c.ClassID, c.Level, c.Exp, c.SP
		}
	}
	c.Subclasses = subs
}

// IsSubClassActive reports whether p plays a subclass (Mobius
// isSubClassActive; scripting.PlayerSubclass).
func (p *Player) IsSubClassActive() bool { return p.char.ClassIndex > baseClassIndex }

// ClassIndex returns the active class index: 0 for the base class, 1-3 for
// a subclass (Mobius getClassIndex).
func (p *Player) ClassIndex() int { return int(p.char.ClassIndex) }

// TotalSubClasses returns how many subclasses p has (Mobius
// getTotalSubClasses).
func (p *Player) TotalSubClasses() int { return len(p.char.Subclasses) }

// Subclasses returns p's subclasses in index order, with the live progress
// of the active one. Loop-only.
func (p *Player) Subclasses() []storage.Subclass {
	c := storage.Character{ClassIndex: p.char.ClassIndex, ClassID: p.char.ClassID, Level: int32(p.Level()), Exp: p.char.Exp, SP: p.char.SP, Subclasses: p.char.Subclasses} //nolint:gosec // levels are small
	p.storeSubclass(&c)
	return slices.Clone(c.Subclasses)
}

// subclassOf returns p's subclass of an index, or nil.
func (p *Player) subclassOf(index int32) *storage.Subclass {
	for i := range p.char.Subclasses {
		if p.char.Subclasses[i].ClassIndex == index {
			return &p.char.Subclasses[i]
		}
	}
	return nil
}

// subclassOwner describes p for the subclass rules.
func subclassOwner(p *Player) *subclass.Owner {
	o := &subclass.Owner{
		Race:        data.Race(p.char.Race), //nolint:gosec // races are small
		BaseClass:   p.BaseClassID(),
		ActiveClass: p.ClassID(),
		Level:       p.Level(),
	}
	for _, sub := range p.Subclasses() {
		o.Subs = append(o.Subs, subclass.Sub{Index: int(sub.ClassIndex), ClassID: int(sub.ClassID), Level: int(sub.Level)})
	}
	return o
}

// classCategories returns the class categories for the subclass rules; a
// server without category data has none.
func (s *Server) classCategories() subclass.Categories {
	if s.items.categories == nil {
		return noCategories{}
	}
	return s.items.categories
}

// noCategories is a category table without categories.
type noCategories struct{}

func (noCategories) InCategory(data.CategoryType, int) bool { return false }

// minPlayerLevel is the lowest level p's active class drops to: the base
// subclass level on a subclass (SubClassHolder.setLevel), else 1.
func (s *Server) minPlayerLevel(p *Player) int {
	if p.IsSubClassActive() {
		return s.subclass.cfg.BaseLevel
	}
	return 1
}
