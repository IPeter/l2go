// Package subclass holds the rules of subclasses, noblesse and skill
// enchanting (Milestone 11e, Plan 14): which subclasses a village master
// offers (Mobius VillageMaster), when a character may add one, the progress
// of a subclass (SubClassHolder), the class-index migration of stored
// characters (decision X3), and the skill enchant rules
// (EnchantSkillTreeData, RequestExEnchantSkill). It has no game state;
// internal/game adapts it to the world loop, packets and storage.
package subclass

import (
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Config are the subclass settings (Mobius Player.ini, decision X8 keeps
// the Mobius defaults: DefaultConfig).
type Config struct {
	// MaxSubclass is how many subclasses a character may have (MaxSubclass).
	MaxSubclass int
	// BaseLevel is the level a new subclass starts at and the lowest it
	// drops to (BaseSubclassLevel).
	BaseLevel int
	// MaxLevel caps a subclass's level (MaxSubclassLevel), together with the
	// experience table (SubClassHolder.MAX_LEVEL).
	MaxLevel int
	// WithoutQuests adds subclasses without Fate's Whisper and Mimir's
	// Elixir (AltSubclassWithoutQuests).
	WithoutQuests bool
	// Everywhere lets every village master offer every subclass
	// (AltSubclassEverywhere).
	Everywhere bool
	// StoreSkillCooltime keeps the effects of the class left at a switch
	// (SubclassStoreSkillCooltime); otherwise only its skill reuse is kept.
	StoreSkillCooltime bool
	// EnchantBookNeeded asks for a Secret Book of Giants for the first
	// enchant level of each route (EnchantSkillSpBookNeeded).
	EnchantBookNeeded bool
}

// DefaultConfig returns the Mobius defaults.
func DefaultConfig() Config {
	return Config{
		MaxSubclass:       3,
		BaseLevel:         40,
		MaxLevel:          80,
		EnchantBookNeeded: true,
	}
}

// RequiredLevel is the level the active class and every subclass must have
// reached before a subclass is added (VillageMaster "Subclass 4").
const RequiredLevel = 75

// The quests a character completes before it adds a subclass, unless it is
// a noble (VillageMaster.checkQuests).
const (
	QuestFatesWhisper = "Q00234_FatesWhisper"
	QuestMimirsElixir = "Q00235_MimirsElixir"
)

// QuestRepentYourSins is the quest a class switch ends (Player.setActiveClass).
const QuestRepentYourSins = "Q00422_RepentYourSins"

// Sub is a subclass of a character as the rules see it.
type Sub struct {
	Index   int // 1-3
	ClassID int
	Level   int
}

// Owner is a character as the subclass rules see it.
type Owner struct {
	Race data.Race
	// BaseClass is the base class, ActiveClass the class the character is
	// playing now (Mobius getBaseClass and getPlayerClass).
	BaseClass, ActiveClass int
	// Level is the level of the active class.
	Level int
	// Subs are the subclasses, in index order.
	Subs []Sub
}

// hasSub reports whether a subclass of class id exists.
func (o *Owner) hasSub(id int) bool {
	return slices.ContainsFunc(o.Subs, func(s Sub) bool { return s.ClassID == id })
}

// TeachType is the kind of classes a village master teaches (Mobius
// ClassType: the VillageMasterPriest and VillageMasterMystic classes, every
// other village master teaches fighters).
type TeachType int

// Teach types.
const (
	TeachFighter TeachType = iota
	TeachMystic
	TeachPriest
)

// Master is a village master as the rules see it.
type Master struct {
	Race  data.Race
	Teach TeachType
}

// Categories are the class categories the rules read (data.Categories).
type Categories interface {
	InCategory(t data.CategoryType, id int) bool
}

// neverSubclassed are the second classes no one may take as a subclass
// (VillageMaster.neverSubclassed).
var neverSubclassed = []data.PlayerClass{data.ClassOverlord, data.ClassWarsmith}

// The second classes that exclude each other (VillageMaster.subclasseSet1
// to 5): a knight, a rogue, an archer, a summoner and a nuker of another
// race are the same class to the subclass list.
var (
	excludeKnights   = []data.PlayerClass{data.ClassDarkAvenger, data.ClassPaladin, data.ClassTempleKnight, data.ClassShillienKnight}
	excludeRogues    = []data.PlayerClass{data.ClassTreasureHunter, data.ClassAbyssWalker, data.ClassPlainsWalker}
	excludeArchers   = []data.PlayerClass{data.ClassHawkeye, data.ClassSilverRanger, data.ClassPhantomRanger}
	excludeSummoners = []data.PlayerClass{data.ClassWarlock, data.ClassElementalSummoner, data.ClassPhantomSummoner}
	excludeNukers    = []data.PlayerClass{data.ClassSorcerer, data.ClassSpellsinger, data.ClassSpellhowler}
)

// excludedBy returns the classes a second class excludes (subclassSetMap),
// nil for a class without a set.
func excludedBy(c data.PlayerClass) []data.PlayerClass {
	for _, set := range [][]data.PlayerClass{excludeKnights, excludeRogues, excludeArchers, excludeSummoners, excludeNukers} {
		if slices.Contains(set, c) {
			return set
		}
	}
	return nil
}

// priestClasses are the priest classes (PlayerClass._priestSet).
var priestClasses = []data.PlayerClass{
	data.ClassCleric, data.ClassBishop, data.ClassProphet, data.ClassOracle, data.ClassElder,
	data.ClassShillienOracle, data.ClassShillienElder, data.ClassCardinal, data.ClassHierophant,
	data.ClassEvaSaint, data.ClassShillienSaint,
}

// IsOfType is PlayerClass.isOfType: fighters are the classes that are not
// mages, priests the priest classes, mystics the other mages.
func IsOfType(c data.PlayerClass, t TeachType) bool {
	switch t {
	case TeachFighter:
		return !c.IsMage()
	case TeachPriest:
		return c.IsMage() && slices.Contains(priestClasses, c)
	default:
		return c.IsMage() && !slices.Contains(priestClasses, c)
	}
}

// EqualsOrChildOf is PlayerClass.equalsOrChildOf: c is of or comes from
// class of.
func EqualsOrChildOf(c, of data.PlayerClass) bool {
	for {
		if c == of {
			return true
		}
		parent, ok := c.Parent()
		if !ok {
			return false
		}
		c = parent
	}
}

// mainSubclasses is VillageMaster.mainSubclassSet: the second classes
// (category THIRD_CLASS_GROUP) but the Overlord and the Warsmith, in class
// ID order (Mobius keeps an EnumSet).
func mainSubclasses(cats Categories) []data.PlayerClass {
	var out []data.PlayerClass
	for _, c := range data.PlayerClasses() {
		if cats.InCategory(data.CategoryThirdClassGroup, int(c)) && !slices.Contains(neverSubclassed, c) {
			out = append(out, c)
		}
	}
	slices.Sort(out)
	return out
}

// Subclasses is VillageMaster.getSubclasses(player, classID): for a second
// or third class, every main subclass but classID itself, the classes of the
// other elf race for an elf or a dark elf, and the classes classID
// excludes; then the classes the owner's active class is or comes from go.
// It returns nil for any other class (Mobius returns null), in class ID
// order.
func Subclasses(cats Categories, o *Owner, classID int) []data.PlayerClass {
	if !cats.InCategory(data.CategoryThirdClassGroup, classID) && !cats.InCategory(data.CategoryFourthClassGroup, classID) {
		return nil
	}
	p := data.PlayerClass(classID)
	excluded := excludedBy(p)
	var out []data.PlayerClass
	for _, c := range mainSubclasses(cats) {
		switch {
		case c == p:
		case o.Race == data.RaceElf && c.Race() == data.RaceDarkElf:
		case o.Race == data.RaceDarkElf && c.Race() == data.RaceElf:
		case slices.Contains(excluded, c):
		case EqualsOrChildOf(data.PlayerClass(o.ActiveClass), c):
		default:
			out = append(out, c)
		}
	}
	if out == nil {
		out = []data.PlayerClass{} // a set, empty but not null
	}
	return out
}

// Available is VillageMaster.getAvailableSubClasses: the subclasses of the
// base class (Subclasses) without those the owner has, without the parents
// of its third-class subclasses and its base class once it has one, and,
// unless every master teaches everything, those master does not teach: a
// human or elf master teaches the human and elf classes of its type, any
// other master the classes of its race. It returns nil when the base class
// has no list.
func Available(cats Categories, o *Owner, m Master, cfg Config) []data.PlayerClass {
	avail := Subclasses(cats, o, o.BaseClass)
	if avail == nil {
		return nil
	}
	out := avail[:0:0]
	for _, c := range avail {
		if o.hasSub(int(c)) {
			continue
		}
		taken := false
		for _, s := range o.Subs {
			id := s.ClassID
			if id >= 88 {
				if parent, ok := data.PlayerClass(id).Parent(); ok {
					id = int(parent)
				}
			}
			if int(c) == id || int(c) == o.BaseClass {
				taken = true
				break
			}
		}
		if taken {
			continue
		}
		if !cfg.Everywhere {
			if m.Race == data.RaceHuman || m.Race == data.RaceElf {
				if !IsOfType(c, m.Teach) || (c.Race() != data.RaceHuman && c.Race() != data.RaceElf) {
					continue
				}
			} else if c.Race() != m.Race {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// ValidNew is VillageMaster.isValidNewSubClass: no subclass of the owner is
// or comes from classID, and classID is among the subclasses of the base
// class's second class (Subclasses).
func ValidNew(cats Categories, o *Owner, classID int) bool {
	cid := data.PlayerClass(classID)
	for _, s := range o.Subs {
		if EqualsOrChildOf(data.PlayerClass(s.ClassID), cid) {
			return false
		}
	}
	base := o.BaseClass
	if c := data.PlayerClass(base); c.Level() > 2 {
		if parent, ok := c.Parent(); ok {
			base = int(parent)
		}
	}
	return slices.Contains(Subclasses(cats, o, base), cid)
}

// CanAdd reports whether the owner may add a subclass now (VillageMaster
// "Subclass 4" before the quest check): fewer than MaxSubclass subclasses,
// the active class at RequiredLevel, and every subclass too.
func CanAdd(o *Owner, cfg Config) bool {
	if len(o.Subs) >= cfg.MaxSubclass || o.Level < RequiredLevel {
		return false
	}
	for _, s := range o.Subs {
		if s.Level < RequiredLevel {
			return false
		}
	}
	return true
}

// QuestsDone is VillageMaster.checkQuests: a noble needs no quests; anyone
// else must have completed Fate's Whisper and Mimir's Elixir.
func QuestsDone(noble bool, completed func(quest string) bool) bool {
	return noble || (completed(QuestFatesWhisper) && completed(QuestMimirsElixir))
}

// Command is a parsed "Subclass <choice> [<one> [<two>]]" bypass.
type Command struct {
	Choice   int
	One, Two int
	// Bad is set when Mobius would log "Wrong numeric values": the numbers
	// parsed until then are kept, the rest are 0.
	Bad bool
}

// ParseCommand reads a "Subclass" bypass as VillageMaster.onBypassFeedback
// does: the choice is the one character at index 9, the first parameter
// runs from index 11 to the next space, the second is the rest. Mobius
// parses with substring and trim, so "Subclass 4 12" is choice 4 and class
// 12.
func ParseCommand(command string) Command {
	var c Command
	atoi := func(s string) (int, bool) {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		return n, err == nil
	}
	if len(command) < 10 {
		c.Bad = true
		return c
	}
	choice, ok := atoi(command[9:10])
	if !ok {
		c.Bad = true
		return c
	}
	c.Choice = choice
	end := -1
	if len(command) > 11 {
		end = strings.IndexByte(command[11:], ' ')
		if end >= 0 {
			end += 11
		}
	}
	if end == -1 {
		end = len(command)
	}
	if len(command) > 11 {
		one, ok := atoi(command[11:end])
		if !ok {
			c.Bad = true
			return c
		}
		c.One = one
		if len(command) > end {
			two, ok := atoi(command[end:])
			if !ok {
				c.Bad = true
				return c
			}
			c.Two = two
		}
	}
	return c
}

// MaxSubLevel is SubClassHolder.MAX_LEVEL: MaxSubclassLevel while it is
// below the experience table's maximum level (Mobius getMaxLevel, the
// game's MaxLevel), else one below that maximum.
func (cfg Config) MaxSubLevel(tableMax int) int {
	if cfg.MaxLevel < tableMax {
		return cfg.MaxLevel
	}
	return tableMax - 1
}

// ClampLevel is SubClassHolder.setLevel: a subclass level stays within
// BaseLevel and MaxSubLevel.
func (cfg Config) ClampLevel(level, tableMax int) int {
	return max(min(level, cfg.MaxSubLevel(tableMax)), cfg.BaseLevel)
}
