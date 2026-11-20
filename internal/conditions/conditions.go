// Package conditions evaluates Mobius condition trees: and, or, not and the
// player conditions that items use in their <conditions> (Mobius
// mechanics/conditions, parsed as util/DocumentBase does). It is the first
// slice of the condition tree; skills extend it later.
//
// The package is pure: the game supplies the player through Subject.
package conditions

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Kind is the kind of a condition.
type Kind uint8

// The condition kinds.
const (
	KindAnd         Kind = iota + 1 // all Conds hold (ConditionLogicAnd)
	KindOr                          // one of Conds holds (ConditionLogicOr)
	KindNot                         // Conds[0] does not hold (ConditionLogicNot)
	KindRaces                       // player.races: the race is one of Races
	KindLevel                       // player.level: the level is at least Int
	KindSex                         // player.sex: 1 for female, 0 for male, is Int
	KindPKCount                     // player.pkCount: the PK count is at most Int
	KindHero                        // player.isHero: being a hero is Bool
	KindPvPFlagged                  // player.isPvpFlagged: being flagged is Bool
	KindCategory                    // player.categoryType: in one of Categories
	KindPledgeClass                 // player.pledgeClass: clan leader, or pledge class at least Int (-1: leader only)
	KindCastle                      // player.castle: the clan's castle is Int (0: none, -1: any)
	KindClanHall                    // player.clanHall: the clan's hall is one of Ints ([0]: none, [-1]: any)
)

var kindNames = [...]string{
	KindAnd: "and", KindOr: "or", KindNot: "not", KindRaces: "races", KindLevel: "level",
	KindSex: "sex", KindPKCount: "pkCount", KindHero: "isHero", KindPvPFlagged: "isPvpFlagged",
	KindCategory: "categoryType", KindPledgeClass: "pledgeClass", KindCastle: "castle", KindClanHall: "clanHall",
}

// String returns the XML name of k.
func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// Condition is a condition tree. The logic kinds use Conds; the leaves use
// the field their Kind names.
type Condition struct {
	Kind       Kind
	Conds      []Condition
	Int        int
	Bool       bool
	Ints       []int
	Races      []data.Race
	Categories []data.CategoryType
}

// Clan is what the clan conditions need to know of the subject's clan. The
// zero value is a player without a clan.
type Clan struct {
	ID          int32 // 0 when not in a clan
	Leader      bool
	PledgeClass int
	CastleID    int // 0 when the clan has no castle
	HideoutID   int // the clan hall, 0 when the clan has none
}

// Subject is the player a condition is tested for (the Mobius effector).
type Subject interface {
	Race() data.Race
	Level() int
	Female() bool
	PKKills() int
	Hero() bool
	PvPFlagged() bool
	// InCategory reports whether the player's class is in the category
	// (Creature.isInCategory, from CategoryData).
	InCategory(t data.CategoryType) bool
	Clan() Clan
}

// Test reports whether the condition holds for s, as the Mobius testImpl
// of each kind. An unknown kind fails.
func (c *Condition) Test(s Subject) bool {
	switch c.Kind {
	case KindAnd:
		for i := range c.Conds {
			if !c.Conds[i].Test(s) {
				return false
			}
		}
		return true
	case KindOr:
		for i := range c.Conds {
			if c.Conds[i].Test(s) {
				return true
			}
		}
		return false
	case KindNot:
		return len(c.Conds) == 1 && !c.Conds[0].Test(s)
	case KindRaces:
		return slices.Contains(c.Races, s.Race())
	case KindLevel:
		return s.Level() >= c.Int
	case KindSex:
		sex := 0
		if s.Female() {
			sex = 1
		}
		return sex == c.Int
	case KindPKCount:
		return s.PKKills() <= c.Int
	case KindHero:
		return s.Hero() == c.Bool
	case KindPvPFlagged:
		return s.PvPFlagged() == c.Bool
	case KindCategory:
		return slices.ContainsFunc(c.Categories, s.InCategory)
	case KindPledgeClass:
		clan := s.Clan()
		if clan.ID == 0 || (c.Int == -1 && !clan.Leader) {
			return false
		}
		return clan.Leader || clan.PledgeClass >= c.Int
	case KindCastle:
		clan := s.Clan()
		switch {
		case clan.ID == 0:
			return c.Int == 0
		case c.Int == -1:
			return clan.CastleID > 0
		}
		return clan.CastleID == c.Int
	case KindClanHall:
		clan := s.Clan()
		switch {
		case clan.ID == 0:
			return len(c.Ints) == 1 && c.Ints[0] == 0
		case len(c.Ints) == 1 && c.Ints[0] == -1:
			return clan.HideoutID > 0
		}
		return slices.Contains(c.Ints, clan.HideoutID)
	}
	return false
}

// ErrUnsupported is returned for condition elements and attributes this
// package does not know yet.
var ErrUnsupported = errors.New("conditions: unsupported condition")

// Parse parses one condition element: and, or, not or player (Mobius
// DocumentBase.parseCondition). The attributes of one player element are
// joined with and, in name order.
func Parse(e *data.XMLElement) (Condition, error) {
	switch strings.ToLower(e.Name) {
	case "and", "or":
		kind := KindAnd
		if strings.EqualFold(e.Name, "or") {
			kind = KindOr
		}
		if len(e.Children) == 0 {
			return Condition{}, fmt.Errorf("conditions: empty <%s>", e.Name)
		}
		c := Condition{Kind: kind, Conds: make([]Condition, 0, len(e.Children))}
		for i := range e.Children {
			sub, err := Parse(&e.Children[i])
			if err != nil {
				return Condition{}, err
			}
			c.Conds = append(c.Conds, sub)
		}
		return c, nil
	case "not":
		if len(e.Children) == 0 {
			return Condition{}, errors.New("conditions: empty <not>")
		}
		// Mobius uses the first child only.
		sub, err := Parse(&e.Children[0])
		if err != nil {
			return Condition{}, err
		}
		return Condition{Kind: KindNot, Conds: []Condition{sub}}, nil
	case "player":
		return parsePlayer(e)
	}
	return Condition{}, fmt.Errorf("%w: <%s>", ErrUnsupported, e.Name)
}

// parsePlayer parses a <player> element (DocumentBase.parsePlayerCondition).
func parsePlayer(e *data.XMLElement) (Condition, error) {
	names := make([]string, 0, len(e.Attrs))
	for name := range e.Attrs {
		names = append(names, name)
	}
	if len(names) == 0 {
		return Condition{}, errors.New("conditions: <player> without attributes")
	}
	slices.Sort(names)
	var leaves []Condition
	for _, name := range names {
		leaf, err := parsePlayerAttr(name, e.Attrs[name])
		if err != nil {
			return Condition{}, fmt.Errorf("conditions: <player %s=%q>: %w", name, e.Attrs[name], err)
		}
		leaves = append(leaves, leaf)
	}
	if len(leaves) == 1 {
		return leaves[0], nil
	}
	return Condition{Kind: KindAnd, Conds: leaves}, nil
}

func parsePlayerAttr(name, v string) (Condition, error) {
	var (
		c   Condition
		err error
	)
	switch strings.ToLower(name) {
	case "races":
		c.Kind = KindRaces
		for _, s := range strings.Split(v, ",") {
			r, ok := data.ParseRace(s)
			if !ok {
				return c, fmt.Errorf("unknown race %q", s)
			}
			c.Races = append(c.Races, r)
		}
	case "level":
		c.Kind = KindLevel
		c.Int, err = decodeInt(v)
	case "sex":
		c.Kind = KindSex
		c.Int, err = decodeInt(v)
	case "pkcount":
		c.Kind = KindPKCount
		c.Int, err = decodeInt(v)
	case "ishero":
		c.Kind, c.Bool = KindHero, parseBool(v)
	case "ispvpflagged":
		c.Kind, c.Bool = KindPvPFlagged, parseBool(v)
	case "categorytype":
		c.Kind = KindCategory
		for _, s := range strings.Split(v, ",") {
			t, ok := data.ParseCategoryType(s)
			if !ok {
				return c, fmt.Errorf("unknown category %q", s)
			}
			c.Categories = append(c.Categories, t)
		}
	case "pledgeclass":
		c.Kind = KindPledgeClass
		c.Int, err = decodeInt(v)
	case "castle":
		c.Kind = KindCastle
		c.Int, err = decodeInt(v)
	case "clanhall":
		c.Kind = KindClanHall
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s == "" {
				continue // StringTokenizer skips empty tokens
			}
			n, err := decodeInt(s)
			if err != nil {
				return c, err
			}
			c.Ints = append(c.Ints, n)
		}
	default:
		return c, ErrUnsupported
	}
	return c, err
}

// decodeInt parses an int as Java Integer.decode: decimal, 0x/0X/# hex or
// 0 octal, with an optional sign.
func decodeInt(s string) (int, error) {
	t := s
	neg := false
	if t != "" && (t[0] == '-' || t[0] == '+') {
		neg = t[0] == '-'
		t = t[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x"), strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case strings.HasPrefix(t, "#"):
		base, t = 16, t[1:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	if t == "" || t[0] == '-' || t[0] == '+' {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	n, err := strconv.ParseInt(t, base, 64)
	if neg {
		n = -n
	}
	if err != nil || n < -1<<31 || n > 1<<31-1 {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return int(n), nil
}

// parseBool is Java Boolean.parseBoolean: true only for "true", ignoring
// case.
func parseBool(s string) bool { return strings.EqualFold(s, "true") }
