package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// npcXMLDir holds the NPC templates. Only the files directly in it are
// read: custom/ is left out (Mobius loads it only with CustomNpcData on).
const npcXMLDir = "dist/game/data/stats/npcs"

// readNpcs reads the NPC templates, sorted by ID. Fake player NPCs, which
// Mobius skips unless fake players are enabled, are left out.
func readNpcs(o options) ([]data.NpcTemplate, error) {
	names, err := fs.Glob(o.src, npcXMLDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no NPC files found", npcXMLDir)
	}
	var (
		npcs     []data.NpcTemplate
		seen     = make(map[int]string) // NPC ID -> file
		errs     []error
		fakes    int
		repeated []string // "NPC id: name" of repeated parameters
	)
	for _, name := range names {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range list.children() {
			if e.name != "npc" {
				e.unknown()
				continue
			}
			n, fake, rep := parseNpc(e)
			if fake {
				fakes++
				continue
			}
			for _, r := range rep {
				repeated = append(repeated, fmt.Sprintf("NPC %d: %s", n.ID, r))
			}
			if prev, dup := seen[n.ID]; dup {
				e.errorf("NPC %d is also defined in %s", n.ID, prev)
				continue
			}
			seen[n.ID] = name
			npcs = append(npcs, n)
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if fakes > 0 {
		o.warnf("npcs: left out %d fake player NPCs (Mobius skips them unless fake players are enabled)", fakes)
	}
	if len(repeated) > 0 {
		o.warnf("npcs: parameters given twice, the last value kept as in Mobius: %s", strings.Join(repeated, "; "))
	}
	slices.SortFunc(npcs, func(a, b data.NpcTemplate) int { return cmp.Compare(a.ID, b.ID) })
	return npcs, nil
}

// parseNpc reads one <npc> element with the defaults of Mobius's NpcData,
// NpcTemplate and CreatureTemplate. fake reports a fake player NPC;
// repeated lists parameter names given more than once.
func parseNpc(e *mxElem) (n data.NpcTemplate, fake bool, repeated []string) {
	id := e.reqInt("id")
	n = data.NpcTemplate{
		ID:                   id,
		DisplayID:            e.intAttr("displayId", id),
		Level:                e.intAttr("level", 85),
		Type:                 e.str("type", "Folk"),
		Name:                 e.str("name", ""),
		UsingServerSideName:  e.boolAttr("usingServerSideName", false),
		Title:                e.str("title", ""),
		UsingServerSideTitle: e.boolAttr("usingServerSideTitle", false),
		Race:                 data.RaceNone,
		Sex:                  "ETC",
		AI: data.NpcAI{
			Type:           "FIGHTER",
			IsAggressive:   true,
			MinSkillChance: 7,
			MaxSkillChance: 15,
		},
	}
	n.Stats.Attack = data.NpcAttack{Critical: 4, Speed: 300, Type: data.WeaponFist, Range: 40, Width: 120}
	var speed npcSpeeds
	var status npcFlags
	for _, c := range e.children() {
		switch c.name {
		case "parameters":
			n.Parameters, repeated = parseNpcParameters(c)
		case "race":
			r, ok := data.ParseRace(strings.ToUpper(c.text))
			if !ok {
				c.errorf("unknown race %q", c.text)
			}
			n.Race = r
		case "sex":
			n.Sex = strings.ToUpper(c.text)
		case "equipment":
			n.Equipment = data.NpcEquipment{
				Chest:         c.intAttr("chest", 0),
				RHand:         c.intAttr("rhand", 0),
				LHand:         c.intAttr("lhand", 0),
				WeaponEnchant: c.intAttr("weaponEnchant", 0),
			}
		case "acquire":
			n.Acquire = data.NpcAcquire{
				Exp:        c.floatAttr("exp", 0),
				SP:         c.floatAttr("sp", 0),
				RaidPoints: c.floatAttr("raidPoints", 0),
			}
		case "stats":
			parseNpcStats(c, &n.Stats, &speed)
		case "status":
			status.parse(c)
		case "fakePlayer":
			c.skip()
			fake = true
		case "skillList":
			for _, s := range c.children() {
				if s.name != "skill" {
					s.unknown()
					continue
				}
				n.Skills = append(n.Skills, data.NpcSkill{ID: s.reqInt("id"), Level: s.reqInt("level")})
			}
		case "shots":
			n.Shots = data.NpcShots{
				Soul:         c.intAttr("soul", 0),
				Spirit:       c.intAttr("spirit", 0),
				ShotChance:   c.intAttr("shotChance", 0),
				SpiritChance: c.intAttr("spiritChance", 0),
			}
		case "corpseTime":
			n.CorpseTime = c.textInt()
		case "exCrtEffect":
			n.ExCrtEffect = c.textBool()
		case "sNpcPropHpRate":
			n.SNpcPropHpRate = c.textFloat()
		case "ai":
			parseNpcAI(c, &n.AI)
		case "dropLists":
			n.Drops = parseNpcDrops(c)
		case "collision":
			for _, k := range c.children() {
				switch k.name {
				case "radius":
					n.Collision.Radius = k.floatAttr("normal", 0)
					n.Collision.RadiusGrown = k.floatAttr("grown", 0)
				case "height":
					n.Collision.Height = k.floatAttr("normal", 0)
					n.Collision.HeightGrown = k.floatAttr("grown", 0)
				default:
					k.unknown()
				}
			}
		default:
			c.unknown()
		}
	}
	n.Stats.Speed = speed.resolve()
	n.Status = status.resolve(n.Type, n.Stats.Speed.Walk)
	return n, fake, repeated
}

// npcSpeeds collects the <speed> values; nil means not given.
type npcSpeeds struct {
	walk, run, swimWalk, swimRun, flyWalk, flyRun *float64
}

// resolve applies Mobius's speed defaults (NpcData and CreatureTemplate):
// walk 50 and run 120 when not given, a ground speed of 0 or less becomes
// 0.1, and swim and fly speeds default to the ground speeds.
func (s npcSpeeds) resolve() data.NpcSpeed {
	val := func(p *float64, def float64) float64 {
		if p == nil {
			return def
		}
		return *p
	}
	ground := func(p *float64, def float64) float64 {
		if p != nil && *p <= 0 {
			return 0.1
		}
		return val(p, def)
	}
	walk, run := ground(s.walk, 50), ground(s.run, 120)
	return data.NpcSpeed{
		Walk:     walk,
		Run:      run,
		SwimWalk: val(s.swimWalk, walk),
		SwimRun:  val(s.swimRun, run),
		FlyWalk:  val(s.flyWalk, walk),
		FlyRun:   val(s.flyRun, run),
	}
}

func parseNpcStats(e *mxElem, st *data.NpcStats, speed *npcSpeeds) {
	st.STR = e.intAttr("str", 0)
	st.INT = e.intAttr("int", 0)
	st.DEX = e.intAttr("dex", 0)
	st.WIT = e.intAttr("wit", 0)
	st.CON = e.intAttr("con", 0)
	st.MEN = e.intAttr("men", 0)
	for _, c := range e.children() {
		switch c.name {
		case "vitals":
			st.Vitals = data.NpcVitals{
				HP:      c.floatAttr("hp", 0),
				HPRegen: c.floatAttr("hpRegen", 0),
				MP:      c.floatAttr("mp", 0),
				MPRegen: c.floatAttr("mpRegen", 0),
			}
		case "attack":
			a := &st.Attack
			a.Physical = c.floatAttr("physical", 0)
			a.Magical = c.floatAttr("magical", 0)
			a.Random = c.intAttr("random", 0)
			a.Critical = c.intAttr("critical", 4)
			a.Accuracy = c.floatAttr("accuracy", 0)
			a.Speed = c.intAttr("attackSpeed", 300)
			a.ReuseDelay = c.intAttr("reuseDelay", 0)
			if v, ok := c.attr("type"); ok {
				t, known := data.ParseWeaponType(v)
				if !known {
					c.errorf("unknown attack type %q", v)
				}
				a.Type = t
			}
			a.Range = c.intAttr("range", 40)
			a.Distance = c.intAttr("distance", 0)
			a.Width = c.intAttr("width", 120)
		case "defence":
			st.Defence = data.NpcDefence{
				Physical:   c.floatAttr("physical", 0),
				Magical:    c.floatAttr("magical", 0),
				Evasion:    c.intAttr("evasion", 0),
				Shield:     c.intAttr("shield", 0),
				ShieldRate: c.intAttr("shieldRate", 0),
			}
		case "attribute":
			for _, k := range c.children() {
				switch k.name {
				case "attack":
					st.Attribute.AttackType = strings.ToUpper(k.reqStr("type"))
					st.Attribute.AttackValue = k.intAttr("value", 0)
				case "defence":
					st.Attribute.Fire = k.intAttr("fire", 0)
					st.Attribute.Water = k.intAttr("water", 0)
					st.Attribute.Wind = k.intAttr("wind", 0)
					st.Attribute.Earth = k.intAttr("earth", 0)
					st.Attribute.Holy = k.intAttr("holy", 0)
					st.Attribute.Dark = k.intAttr("dark", 0)
					st.Attribute.Default = k.intAttr("default", 0)
				default:
					k.unknown()
				}
			}
		case "speed":
			for _, k := range c.children() {
				var ground, swim, fly **float64
				switch k.name {
				case "walk":
					ground, swim, fly = &speed.walk, &speed.swimWalk, &speed.flyWalk
				case "run":
					ground, swim, fly = &speed.run, &speed.swimRun, &speed.flyRun
				default:
					k.unknown()
					continue
				}
				// Mobius fails on a <walk> or <run> without ground.
				g := k.reqFloat("ground")
				*ground = &g
				if _, ok := k.attr("swim"); ok {
					v := k.floatAttr("swim", 0)
					*swim = &v
				}
				if _, ok := k.attr("fly"); ok {
					v := k.floatAttr("fly", 0)
					*fly = &v
				}
			}
		case "hitTime":
			// Mobius reads the text of <stats> here by mistake and does not
			// use the value yet; this keeps the element's own value.
			st.HitTime = c.textInt()
		default:
			c.unknown()
		}
	}
}

// npcFlags collects the <status> flags; nil means not given.
type npcFlags struct {
	unique, attackable, targetable, talkable, undying, showName, randomWalk,
	randomAnimation, flying, canMove, noSleepMode, passableDoor, hasSummoner,
	canBeSown *bool
}

func (f *npcFlags) parse(e *mxElem) {
	for name, p := range map[string]**bool{
		"unique": &f.unique, "attackable": &f.attackable, "targetable": &f.targetable,
		"talkable": &f.talkable, "undying": &f.undying, "showName": &f.showName,
		"randomWalk": &f.randomWalk, "randomAnimation": &f.randomAnimation, "flying": &f.flying,
		"canMove": &f.canMove, "noSleepMode": &f.noSleepMode, "passableDoor": &f.passableDoor,
		"hasSummoner": &f.hasSummoner, "canBeSown": &f.canBeSown,
	} {
		if _, ok := e.attr(name); ok {
			v := e.boolAttr(name, false)
			*p = &v
		}
	}
}

// resolve applies NpcTemplate's defaults, which depend on the type and the
// walk speed.
func (f npcFlags) resolve(typ string, walk float64) data.NpcStatus {
	val := func(p *bool, def bool) bool {
		if p == nil {
			return def
		}
		return *p
	}
	return data.NpcStatus{
		Unique:          val(f.unique, false),
		Attackable:      val(f.attackable, true),
		Targetable:      val(f.targetable, true),
		Talkable:        val(f.talkable, true),
		Undying:         val(f.undying, typ != "Monster" && typ != "RaidBoss" && typ != "GrandBoss"),
		ShowName:        val(f.showName, true),
		RandomWalk:      val(f.randomWalk, typ != "Guard"),
		RandomAnimation: val(f.randomAnimation, true),
		Flying:          val(f.flying, false),
		// NpcTemplate: canMove = baseWalkSpd <= 0.1 || canMove (default true).
		CanMove:      walk <= 0.1 || val(f.canMove, true),
		NoSleepMode:  val(f.noSleepMode, false),
		PassableDoor: val(f.passableDoor, false),
		HasSummoner:  val(f.hasSummoner, false),
		CanBeSown:    val(f.canBeSown, false),
	}
}

// parseNpcParameters reads <parameters>. Mobius keeps all three kinds in
// one map, so a name given twice keeps its last value, whatever its kind;
// repeated returns such names.
func parseNpcParameters(e *mxElem) (p data.NpcParameters, repeated []string) {
	use := func(name string) {
		_, a := p.Params[name]
		_, b := p.Skills[name]
		_, c := p.Minions[name]
		if a || b || c {
			repeated = append(repeated, name)
			delete(p.Params, name)
			delete(p.Skills, name)
			delete(p.Minions, name)
		}
	}
	for _, c := range e.children() {
		switch c.name {
		case "param":
			name, value := c.reqStr("name"), c.reqStr("value")
			use(name)
			if p.Params == nil {
				p.Params = make(map[string]string)
			}
			p.Params[name] = value
		case "skill":
			name := c.reqStr("name")
			sk := data.NpcSkill{ID: c.reqInt("id"), Level: c.reqInt("level")}
			use(name)
			if p.Skills == nil {
				p.Skills = make(map[string]data.NpcSkill)
			}
			p.Skills[name] = sk
		case "minions":
			name := c.reqStr("name")
			var list []data.NpcMinion
			for _, m := range c.children() {
				if m.name != "npc" {
					m.unknown()
					continue
				}
				list = append(list, data.NpcMinion{
					NpcID:       m.reqInt("id"),
					Count:       m.reqInt("count"),
					Max:         m.intAttr("max", 0),
					RespawnTime: m.reqInt("respawnTime"),
					WeightPoint: m.intAttr("weightPoint", 0),
				})
			}
			// Mobius does not store an empty minion list.
			if len(list) > 0 {
				use(name)
				if p.Minions == nil {
					p.Minions = make(map[string][]data.NpcMinion)
				}
				p.Minions[name] = list
			}
		default:
			c.unknown()
		}
	}
	// Maps emptied by a repeated name are dropped, so the JSON stays the
	// same as for an NPC that never had them.
	if len(p.Params) == 0 {
		p.Params = nil
	}
	if len(p.Skills) == 0 {
		p.Skills = nil
	}
	if len(p.Minions) == 0 {
		p.Minions = nil
	}
	return p, repeated
}

func parseNpcAI(e *mxElem, ai *data.NpcAI) {
	ai.Type = e.str("type", ai.Type)
	ai.AggroRange = e.intAttr("aggroRange", 0)
	ai.ClanHelpRange = e.intAttr("clanHelpRange", 0)
	ai.IsChaos = e.boolAttr("isChaos", false)
	ai.IsAggressive = e.boolAttr("isAggressive", true)
	for _, c := range e.children() {
		switch c.name {
		case "skill":
			ai.MinSkillChance = c.intAttr("minChance", 7)
			ai.MaxSkillChance = c.intAttr("maxChance", 15)
			ai.PrimarySkillID = c.intAttr("primaryId", 0)
			ai.ShortRangeSkillID = c.intAttr("shortRangeId", 0)
			ai.ShortRangeSkillChance = c.intAttr("shortRangeChance", 0)
			ai.LongRangeSkillID = c.intAttr("longRangeId", 0)
			ai.LongRangeSkillChance = c.intAttr("longRangeChance", 0)
		case "clanList":
			for _, k := range c.children() {
				switch k.name {
				case "clan":
					ai.Clans = append(ai.Clans, k.text)
				case "ignoreNpcId":
					ai.IgnoreNpcIDs = append(ai.IgnoreNpcIDs, k.textInt())
				default:
					k.unknown()
				}
			}
		default:
			c.unknown()
		}
	}
}

// parseNpcDrops reads <dropLists>: <drop> holds items and groups, <spoil>
// items (the Mobius DropType names).
func parseNpcDrops(e *mxElem) data.NpcDrops {
	var d data.NpcDrops
	for _, c := range e.children() {
		switch c.name {
		case "drop":
			for _, k := range c.children() {
				switch k.name {
				case "item":
					d.Items = append(d.Items, parseNpcDropItem(k))
				case "group":
					g := data.NpcDropGroup{Chance: k.reqFloat("chance")}
					for _, it := range k.children() {
						if it.name != "item" {
							it.unknown()
							continue
						}
						g.Items = append(g.Items, parseNpcDropItem(it))
					}
					d.Groups = append(d.Groups, g)
				default:
					k.unknown()
				}
			}
		case "spoil":
			for _, k := range c.children() {
				if k.name != "item" {
					k.unknown()
					continue
				}
				d.Spoil = append(d.Spoil, parseNpcDropItem(k))
			}
		default:
			c.unknown()
		}
	}
	return d
}

func parseNpcDropItem(e *mxElem) data.NpcDropItem {
	return data.NpcDropItem{
		ItemID: e.reqInt("id"),
		Min:    e.reqInt("min"),
		Max:    e.reqInt("max"),
		Chance: e.reqFloat("chance"),
	}
}

// readNpcTypes reads the NPC type list from the Mobius Java sources.
func readNpcTypes(o options) ([]data.MobiusClass, error) {
	return readMobiusClasses(o.src, npcTypeJavaDir, npcBaseJavaDir)
}
