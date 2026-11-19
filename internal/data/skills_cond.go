package data

import (
	"errors"
	"fmt"
	"strings"
)

// Condition is a resolved condition, Mobius Condition: and, or and not
// over Conds, or a leaf whose value is in the fields its kind uses:
//
//   - Bool: the yes/no kinds (player.resting, player.ishero, game.night, ...)
//     and the last part of player.npcidradius;
//   - Int: the number kinds (player.level, player.hp, target.mindistance,
//     using.skill, game.chance, ...), the bit masks of using.kind,
//     target.using (Item.Mask bits) and using.slot (BodyPart.Mask bits),
//     the radius of player.npcidradius, and the level of
//     player.checkabnormal (-1 for any) and player.elementseed;
//   - Ints: the ID lists (player.clanhall, player.insidezoneid,
//     target.npcid, ...), the (min, max) of the levelrange kinds, the (ID,
//     level) of the active_*_lvl kinds, the (item, slot, enchant) of
//     using.slotitem and the NPC IDs of player.npcidradius;
//   - Races, Categories, NpcTypes, Abnormal and Element: the named values
//     of player.races and target.race, player.categorytype, target.npctype,
//     player.checkabnormal and player.elementseed.
//
// player.cantakecastle and target.playable have no value.
type Condition struct {
	Kind       ConditionKind
	Conds      []Condition
	Bool       bool
	Int        int
	Ints       []int
	Races      []Race
	Categories []CategoryType
	NpcTypes   []InstanceType
	Abnormal   AbnormalType
	Element    SeedElement
}

// IsLogic reports whether the kind is and, or or not.
func (k ConditionKind) IsLogic() bool { return k == CondAnd || k == CondOr || k == CondNot }

// valueFunc resolves a part of an attribute value: a #table reference
// becomes the table's entry where Mobius allows one, and is an error
// elsewhere.
type valueFunc func(s string) (string, error)

// condParser parses the attribute value of a leaf kind into c. tables says
// whether Mobius resolves #table references in it (DocumentBase calls
// getValue with the skill as template).
type condParser struct {
	tables bool
	parse  func(c *Condition, v string, get valueFunc) error
}

// resolve parses the condition. ti resolves #table references where Mobius
// allows them; nil means none are allowed, as in the conditions of stat
// functions.
func (d *ConditionDef) resolve(ti *tableIndex) (Condition, error) {
	c := Condition{Kind: d.Kind}
	switch {
	case !d.Kind.Valid():
		return c, fmt.Errorf("unknown condition %d", d.Kind)
	case d.Kind.IsLogic():
		if d.Value != "" {
			return c, fmt.Errorf("%s: a logic condition has no value", d.Kind)
		}
		if len(d.Conds) == 0 || (d.Kind == CondNot && len(d.Conds) != 1) {
			return c, fmt.Errorf("%s: %d operands", d.Kind, len(d.Conds))
		}
		c.Conds = make([]Condition, len(d.Conds))
		for i := range d.Conds {
			sub, err := d.Conds[i].resolve(ti)
			if err != nil {
				return c, err
			}
			c.Conds[i] = sub
		}
		return c, nil
	case len(d.Conds) > 0:
		return c, fmt.Errorf("%s: a leaf condition has operands", d.Kind)
	}
	p, ok := condParsers[d.Kind]
	if !ok {
		return c, fmt.Errorf("%s: datagen knows this condition but cannot read its value", d.Kind)
	}
	get := func(s string) (string, error) {
		if !usesTable(s) {
			return s, nil
		}
		if !p.tables || ti == nil {
			return "", fmt.Errorf("table %s: Mobius takes no table here", s)
		}
		return ti.value(s)
	}
	if err := p.parse(&c, d.Value, get); err != nil {
		return c, fmt.Errorf("%s %q: %w", d.Kind, d.Value, err)
	}
	return c, nil
}

// Helpers for the condition parsers. Their names give the Mobius parsing:
// decode is Integer.decode after getValue, plain values skip getValue.

func condBool() condParser {
	return condParser{parse: func(c *Condition, v string, _ valueFunc) (err error) {
		c.Bool, err = parseJavaBool(v)
		return err
	}}
}

func condDecode(tables bool) condParser {
	return condParser{tables: tables, parse: func(c *Condition, v string, get valueFunc) error {
		s, err := get(v)
		if err != nil {
			return err
		}
		c.Int, err = parseJavaDecode(s)
		return err
	}}
}

// condDecodeList reads a comma-separated list of ints (Mobius StringTokenizer
// with trimmed items, each through getValue without a template).
func condDecodeList() condParser {
	return condParser{parse: func(c *Condition, v string, get valueFunc) error {
		for _, item := range tokens(v, ",") {
			s, err := get(strings.TrimSpace(item))
			if err != nil {
				return err
			}
			n, err := parseJavaDecode(s)
			if err != nil {
				return err
			}
			c.Ints = append(c.Ints, n)
		}
		if len(c.Ints) == 0 {
			return errors.New("no values")
		}
		return nil
	}}
}

// condRange reads "min;max".
func condRange() condParser {
	return condParser{tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		s, err := get(v)
		if err != nil {
			return err
		}
		parts := strings.Split(s, ";")
		if len(parts) != 2 {
			return errors.New("want min;max (Mobius ignores anything else)")
		}
		for _, part := range parts {
			n, err := parseJavaDecode(part)
			if err != nil {
				return err
			}
			c.Ints = append(c.Ints, n)
		}
		return nil
	}}
}

// condIDLevel reads "id,level" (the active_*_lvl kinds).
func condIDLevel() condParser {
	return condParser{tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		s, err := get(v)
		if err != nil {
			return err
		}
		parts := strings.Split(s, ",")
		if len(parts) < 2 {
			return errors.New("want id,level")
		}
		for _, part := range parts[:2] {
			p, err := get(part)
			if err != nil {
				return err
			}
			n, err := parseJavaDecode(p)
			if err != nil {
				return err
			}
			c.Ints = append(c.Ints, n)
		}
		return nil
	}}
}

// condMask reads weapon and armor type names into an item type mask.
func condMask() condParser {
	return condParser{parse: func(c *Condition, v string, _ valueFunc) error {
		for _, item := range tokens(v, ",") {
			name := strings.TrimSpace(item)
			w, wok := ParseWeaponType(name)
			a, aok := ParseArmorType(name)
			if !wok && !aok {
				return fmt.Errorf("unknown weapon or armor type %q", name)
			}
			if wok {
				c.Int |= w.Mask()
			}
			if aok {
				c.Int |= a.Mask()
			}
		}
		return nil
	}}
}

func condNone() condParser {
	return condParser{parse: func(*Condition, string, valueFunc) error { return nil }}
}

// tokens splits s like a Java StringTokenizer: on any of the delimiters,
// without empty tokens.
func tokens(s, delims string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(delims, r) })
}

// condParsers holds the parser of each leaf kind, following the matching
// case of Mobius DocumentBase.
var condParsers = map[ConditionKind]condParser{
	CondPlayerRaces: {parse: func(c *Condition, v string, _ valueFunc) error {
		for _, name := range strings.Split(v, ",") {
			r, ok := ParseRace(name)
			if !ok {
				return fmt.Errorf("unknown race %q", name)
			}
			c.Races = append(c.Races, r)
		}
		return nil
	}},
	CondPlayerLevel:              condDecode(true),
	CondPlayerLevelrange:         condRange(),
	CondPlayerResting:            condBool(),
	CondPlayerFlying:             condBool(),
	CondPlayerMoving:             condBool(),
	CondPlayerRunning:            condBool(),
	CondPlayerStanding:           condBool(),
	CondPlayerBehind:             condBool(),
	CondPlayerFront:              condBool(),
	CondPlayerChaotic:            condBool(),
	CondPlayerOlympiad:           condBool(),
	CondPlayerIshero:             condBool(),
	CondPlayerIspvpflagged:       condBool(),
	CondPlayerIsridingstrider:    condBool(),
	CondPlayerHP:                 condDecode(false),
	CondPlayerMP:                 condDecode(false),
	CondPlayerCP:                 condDecode(false),
	CondPlayerGrade:              condDecode(true),
	CondPlayerPkcount:            condDecode(true),
	CondPlayerSiegezone:          condDecode(false),
	CondPlayerSiegeside:          condDecode(false),
	CondPlayerCharges:            condDecode(true),
	CondPlayerWeight:             condDecode(false),
	CondPlayerInvsize:            condDecode(false),
	CondPlayerIsclanleader:       condBool(),
	CondPlayerPledgeclass:        condDecode(false),
	CondPlayerClanhall:           condDecodeList(),
	CondPlayerCastle:             condDecode(false),
	CondPlayerSex:                condDecode(false),
	CondPlayerVehiclemounted:     condBool(),
	CondPlayerActiveEffectID:     condDecode(true),
	CondPlayerActiveEffectIDLvl:  condIDLevel(),
	CondPlayerActiveSkillID:      condDecode(true),
	CondPlayerActiveSkillIDLvl:   condIDLevel(),
	CondPlayerClassIDRestriction: condDecodeList(),
	CondPlayerSubclass:           condBool(),
	CondPlayerInstanceid:         condDecodeList(),
	CondPlayerAgathionid: {parse: func(c *Condition, v string, _ valueFunc) (err error) {
		c.Int, err = parseJavaDecode(v)
		return err
	}},
	CondPlayerCloakstatus:   condBool(),
	CondPlayerHaspet:        condDecodeList(),
	CondPlayerServitornpcid: condDecodeList(),
	CondPlayerNpcidradius: {tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		parts := tokens(v, ",")
		if len(parts) != 3 {
			return errors.New("want ids,radius,true|false (Mobius ignores anything else)")
		}
		for _, id := range strings.Split(parts[0], ";") {
			s, err := get(id)
			if err != nil {
				return err
			}
			n, err := parseJavaInt(s)
			if err != nil {
				return err
			}
			c.Ints = append(c.Ints, n)
		}
		var err error
		if c.Int, err = parseJavaInt(parts[1]); err != nil {
			return err
		}
		c.Bool, err = parseJavaBool(parts[2])
		return err
	}},
	CondPlayerCallpc:              condBool(),
	CondPlayerCancreatebase:       condBool(),
	CondPlayerCancreateoutpost:    condBool(),
	CondPlayerCanescape:           condBool(),
	CondPlayerCanresurrect:        condBool(),
	CondPlayerCansummon:           condBool(),
	CondPlayerCansummonsiegegolem: condBool(),
	CondPlayerCansweep:            condBool(),
	CondPlayerCantakecastle:       condNone(),
	CondPlayerInsidezoneid:        condDecodeList(),
	CondPlayerCheckabnormal: {tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		name, level, hasLevel := strings.Cut(v, ",")
		t, ok := ParseAbnormalType(name)
		if !ok {
			return fmt.Errorf("unknown abnormal type %q", name)
		}
		c.Abnormal, c.Int = t, -1
		if hasLevel {
			level, _, _ = strings.Cut(level, ",")
			s, err := get(level)
			if err != nil {
				return err
			}
			if c.Int, err = parseJavaDecode(s); err != nil {
				return err
			}
		}
		return nil
	}},
	CondPlayerCategorytype: {parse: func(c *Condition, v string, get valueFunc) error {
		for _, name := range strings.Split(v, ",") {
			s, err := get(name)
			if err != nil {
				return err
			}
			t, ok := ParseCategoryType(s)
			if !ok {
				return fmt.Errorf("unknown category %q", s)
			}
			c.Categories = append(c.Categories, t)
		}
		return nil
	}},
	CondPlayerElementseed: {tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		s, err := get(v)
		if err != nil {
			return err
		}
		parts := strings.Split(s, ",")
		if len(parts) < 2 {
			return errors.New("want element,count")
		}
		e, ok := ParseSeedElement(parts[0])
		if !ok {
			return fmt.Errorf("unknown element %q", parts[0])
		}
		n, err := get(parts[1])
		if err != nil {
			return err
		}
		c.Element = e
		c.Int, err = parseJavaDecode(n)
		return err
	}},
	CondPlayerElementseeds: {parse: func(c *Condition, v string, _ valueFunc) (err error) {
		c.Int, err = parseJavaInt(v)
		return err
	}},

	CondTargetAggro:              condBool(),
	CondTargetSiegezone:          condDecode(false),
	CondTargetLevel:              condDecode(true),
	CondTargetLevelrange:         condRange(),
	CondTargetMypartyexceptme:    condBool(),
	CondTargetPlayable:           condNone(),
	CondTargetClassIDRestriction: condDecodeList(),
	CondTargetActiveEffectID:     condDecode(true),
	CondTargetActiveEffectIDLvl:  condIDLevel(),
	CondTargetActiveSkillID:      condDecode(true),
	CondTargetActiveSkillIDLvl:   condIDLevel(),
	CondTargetAbnormal:           condDecode(true),
	CondTargetMindistance:        condDecode(false),
	CondTargetRace: {parse: func(c *Condition, v string, _ valueFunc) error {
		r, ok := ParseRace(v)
		if !ok {
			return fmt.Errorf("unknown race %q", v)
		}
		c.Races = []Race{r}
		return nil
	}},
	CondTargetUsing: condMask(),
	CondTargetNpcid: condDecodeList(),
	CondTargetNpctype: {tables: true, parse: func(c *Condition, v string, get valueFunc) error {
		s, err := get(v)
		if err != nil {
			return err
		}
		for _, name := range strings.Split(strings.TrimSpace(s), ",") {
			t, ok := ParseInstanceType(name)
			if !ok {
				return fmt.Errorf("unknown instance type %q", name)
			}
			c.NpcTypes = append(c.NpcTypes, t)
		}
		return nil
	}},
	CondTargetWeight:  condDecode(false),
	CondTargetInvsize: condDecode(false),

	CondUsingKind: condMask(),
	CondUsingSlot: {parse: func(c *Condition, v string, _ valueFunc) error {
		for _, item := range tokens(v, ",") {
			name := strings.TrimSpace(item)
			p, ok := BodyPartByName(name)
			if !ok {
				return fmt.Errorf("unknown body part %q", name)
			}
			c.Int |= p.Mask()
		}
		return nil
	}},
	CondUsingSkill: {parse: func(c *Condition, v string, _ valueFunc) (err error) {
		c.Int, err = parseJavaInt(v)
		return err
	}},
	CondUsingSlotitem: {parse: func(c *Condition, v string, _ valueFunc) error {
		parts := tokens(v, ";")
		if len(parts) < 2 {
			return errors.New("want item;slot[;enchant]")
		}
		if len(parts) == 2 {
			parts = append(parts, "0")
		}
		for _, part := range parts[:3] {
			n, err := parseJavaInt(strings.TrimSpace(part))
			if err != nil {
				return err
			}
			c.Ints = append(c.Ints, n)
		}
		return nil
	}},

	CondGameSkill:  condBool(),
	CondGameNight:  condBool(),
	CondGameChance: condDecode(false),
}
