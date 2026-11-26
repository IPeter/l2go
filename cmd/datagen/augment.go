package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The augment domain writes the augmentation data (Plan 14, M11d), loaded
// by data.LoadAugment: the options, the retail chances, and the tables
// Mobius keeps in Java, which come from input/augment.json and are checked
// against the Java sources (decision X6).
func init() { registerDomain("augment", genAugment) }

// Mobius sources of the augment domain, relative to the project folder.
const (
	augmentOptionsDir   = "dist/game/data/stats/augmentation/options"
	augmentChancesFile  = "dist/game/data/stats/augmentation/retailchances.xml"
	refinePacketFile    = javaRoot + "network/clientpackets/AbstractRefinePacket.java"
	confirmCancelFile   = javaRoot + "network/clientpackets/RequestConfirmCancelItem.java"
	refineCancelFile    = javaRoot + "network/clientpackets/RequestRefineCancel.java"
	augmentHTMLDir      = "dist/game/data/html"
	augmentRefineMarker = "common/augmentation_0" // the pages that open the windows
	augmentBypassMarker = "_Augment "             // the bypass of those pages
)

// augmentInput is the Java tables of augmentation, as input/augment.json
// writes them.
//
//go:embed input/augment.json
var augmentInput []byte

// augmentTables is the content of input/augment.json.
type augmentTables struct {
	Comment         string   `json:"comment"`
	Source          []string `json:"source"`
	LifeStoneLevels []int    `json:"lifeStoneLevels"`
	LifeStones      []struct {
		ItemID int `json:"itemId"`
		Grade  int `json:"grade"`
		Level  int `json:"level"`
	} `json:"lifeStones"`
	Gemstones  []data.AugmentGemstone  `json:"gemstones"`
	CancelFees []data.AugmentCancelFee `json:"cancelFees"`
}

// genAugment adds augment/refine.json, augment/options.json and
// augment/chances.json.
func genAugment(o options) error {
	refine, err := readAugmentRefine(o.src, augmentInput)
	if err != nil {
		return err
	}
	if refine.Npcs, err = readAugmentNpcs(o.src); err != nil {
		return err
	}
	opts, err := readAugmentOptions(o.src)
	if err != nil {
		return err
	}
	chances, err := readAugmentChances(o.src)
	if err != nil {
		return err
	}
	if _, err := data.NewAugment(data.AugmentData{Refine: refine, Options: opts, Chances: chances}); err != nil {
		return err
	}
	if err := o.out.addJSON(data.AugmentRefineFile, refine); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.AugmentOptionsFile, opts); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.AugmentChancesFile, chances); err != nil {
		return err
	}
	augments := 0
	for _, g := range chances {
		augments += len(g.Augments)
	}
	fmt.Fprintf(o.log, "augment: %d life stones, %d options, %d chance groups with %d augments, %d NPCs\n",
		len(refine.LifeStones), len(opts), len(chances), augments, len(refine.Npcs))
	return nil
}

// readAugmentRefine reads the input tables and checks them against the
// Java sources they name.
func readAugmentRefine(src fs.FS, input []byte) (data.AugmentRefine, error) {
	var in augmentTables
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return data.AugmentRefine{}, fmt.Errorf("input/augment.json: %w", err)
	}
	want := []string{refinePacketFile, confirmCancelFile, refineCancelFile}
	if !slices.Equal(in.Source, want) {
		return data.AugmentRefine{}, fmt.Errorf("input/augment.json: source %q, want %q", in.Source, want)
	}
	java := make(map[string]string, len(want))
	for _, name := range want {
		b, err := fs.ReadFile(src, name)
		if err != nil {
			return data.AugmentRefine{}, fmt.Errorf("read %s: %w", name, err)
		}
		java[name] = stripJavaComments(string(b))
	}
	if err := checkLifeStones(java[refinePacketFile], in); err != nil {
		return data.AugmentRefine{}, fmt.Errorf("%s: %w", refinePacketFile, err)
	}
	if err := checkGemstones(java[refinePacketFile], in.Gemstones); err != nil {
		return data.AugmentRefine{}, fmt.Errorf("%s: %w", refinePacketFile, err)
	}
	for _, name := range []string{confirmCancelFile, refineCancelFile} {
		fees, err := javaCancelFees(java[name])
		if err != nil {
			return data.AugmentRefine{}, fmt.Errorf("%s: %w", name, err)
		}
		if !slices.Equal(fees, in.CancelFees) {
			return data.AugmentRefine{}, fmt.Errorf("%s: cancel fees %+v, input/augment.json has %+v", name, fees, in.CancelFees)
		}
	}

	out := data.AugmentRefine{Source: in.Source, Gemstones: in.Gemstones, CancelFees: in.CancelFees}
	for _, ls := range in.LifeStones {
		if ls.Level < 0 || ls.Level >= len(in.LifeStoneLevels) {
			return data.AugmentRefine{}, fmt.Errorf("input/augment.json: life stone %d: level %d has no player level", ls.ItemID, ls.Level)
		}
		out.LifeStones = append(out.LifeStones, data.LifeStone{ItemID: ls.ItemID, Grade: ls.Grade, Level: ls.Level, PlayerLevel: in.LifeStoneLevels[ls.Level]})
	}
	slices.SortFunc(out.LifeStones, func(a, b data.LifeStone) int { return a.ItemID - b.ItemID })
	return out, nil
}

var (
	lifeStonePutRe    = regexp.MustCompile(`_lifeStones\.put\(\s*(\d+)\s*,\s*new\s+LifeStone\(\s*(\w+)\s*,\s*(\d+)\s*\)\s*\)`)
	lifeStoneLevelRe  = regexp.MustCompile(`int\[\]\s+LEVELS\s*=\s*\{([^}]*)\}`)
	augmentIntConstRe = regexp.MustCompile(`\b([A-Z][A-Z0-9_]*)\s*=\s*(\d+)\s*;`)
)

// checkLifeStones compares the life stones and their level table with the
// static block of AbstractRefinePacket.
func checkLifeStones(java string, in augmentTables) error {
	consts := javaIntConsts(java)
	type stone struct{ grade, level int }
	got := map[int]stone{}
	for _, m := range lifeStonePutRe.FindAllStringSubmatch(java, -1) {
		id, _ := strconv.Atoi(m[1])
		grade, ok := consts[m[2]]
		if !ok {
			return fmt.Errorf("life stone %d: unknown grade %s", id, m[2])
		}
		level, _ := strconv.Atoi(m[3])
		if _, dup := got[id]; dup {
			return fmt.Errorf("life stone %d is put twice", id)
		}
		got[id] = stone{grade, level}
	}
	want := map[int]stone{}
	for _, ls := range in.LifeStones {
		want[ls.ItemID] = stone{ls.Grade, ls.Level}
	}
	if !maps.Equal(got, want) {
		return fmt.Errorf("life stones %v, input/augment.json has %v", got, want)
	}
	m := lifeStoneLevelRe.FindStringSubmatch(java)
	if m == nil {
		return fmt.Errorf("no LEVELS table")
	}
	var levels []int
	for _, f := range strings.Split(m[1], ",") {
		v, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return fmt.Errorf("LEVELS: %q is not an int", f)
		}
		levels = append(levels, v)
	}
	if !slices.Equal(levels, in.LifeStoneLevels) {
		return fmt.Errorf("LEVELS %v, input/augment.json has %v", levels, in.LifeStoneLevels)
	}
	return nil
}

// checkGemstones compares the gemstones with getGemStoneId and
// getGemStoneCount.
func checkGemstones(java string, gems []data.AugmentGemstone) error {
	consts := javaIntConsts(java)
	ids, err := javaSwitchReturns(java, "getGemStoneId", consts)
	if err != nil {
		return err
	}
	counts, err := javaSwitchReturns(java, "getGemStoneCount", consts)
	if err != nil {
		return err
	}
	want := map[string][2]int{}
	for _, g := range gems {
		want[g.Grade.String()] = [2]int{g.ItemID, g.Count}
	}
	got := map[string][2]int{}
	for grade, id := range ids {
		got[grade] = [2]int{id, counts[grade]}
	}
	if len(counts) != len(ids) || !maps.Equal(got, want) {
		return fmt.Errorf("gemstones (item, count) %v and counts %v, input/augment.json has %v", got, counts, want)
	}
	return nil
}

// javaIntConsts returns the int constants of a Java source (NAME = 123;).
func javaIntConsts(java string) map[string]int {
	consts := map[string]int{}
	for _, m := range augmentIntConstRe.FindAllStringSubmatch(java, -1) {
		v, err := strconv.Atoi(m[2])
		if err == nil {
			consts[m[1]] = v
		}
	}
	return consts
}

var javaCaseReturnRe = regexp.MustCompile(`case\s+(\w+)\s*:|default\s*:|return\s+(\w+)\s*;`)

// javaSwitchReturns reads the switch of a method that returns a value per
// case label (case C: case B: { return GEMSTONE_D; }), resolving constants.
// The default branch is left out.
func javaSwitchReturns(java, method string, consts map[string]int) (map[string]int, error) {
	body, err := javaMethodBody(java, method)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	var pending []string
	for _, m := range javaCaseReturnRe.FindAllStringSubmatch(body, -1) {
		switch {
		case m[1] != "":
			pending = append(pending, m[1])
		case m[2] != "":
			v, ok := consts[m[2]]
			if !ok {
				n, err := strconv.Atoi(m[2])
				if err != nil {
					return nil, fmt.Errorf("%s: unknown value %s", method, m[2])
				}
				v = n
			}
			for _, label := range pending {
				out[label] = v
			}
			pending = nil
		default:
			pending = nil
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no cases", method)
	}
	return out, nil
}

var javaCancelFeeRe = regexp.MustCompile(`case\s+(\w+)\s*:|getCrystalCount\(\)\s*<\s*(\d+)|price\s*=\s*(\d+)\s*;`)

// javaCancelFees reads the fee switch of RequestConfirmCancelItem or
// RequestRefineCancel: per grade label, "price = F" after a crystal count
// test "< N" is a fee below N, one after no test is the open-ended fee.
func javaCancelFees(java string) ([]data.AugmentCancelFee, error) {
	var fees []data.AugmentCancelFee
	var grade data.CrystalType
	below := 0
	for _, m := range javaCancelFeeRe.FindAllStringSubmatch(java, -1) {
		switch {
		case m[1] != "":
			g, ok := data.ParseCrystalType(m[1])
			if !ok {
				return nil, fmt.Errorf("unknown grade %s", m[1])
			}
			grade, below = g, 0
		case m[2] != "":
			below, _ = strconv.Atoi(m[2])
		case m[3] != "":
			fee, _ := strconv.ParseInt(m[3], 10, 64)
			if fee == 0 {
				continue // int price = 0;
			}
			if grade == data.CrystalNone {
				return nil, fmt.Errorf("a price outside a grade case")
			}
			fees = append(fees, data.AugmentCancelFee{Grade: grade, Below: below, Fee: fee})
			below = 0
		}
	}
	if len(fees) == 0 {
		return nil, fmt.Errorf("no cancel fees")
	}
	return fees, nil
}

// readAugmentNpcs returns the NPC templates whose dialogs open the
// augmentation windows: the pages <npcId>.htm and <npcId>-<n>.htm that
// link the augmentation pages or send the Augment bypass.
func readAugmentNpcs(src fs.FS) ([]int, error) {
	seen := map[int]bool{}
	err := fs.WalkDir(src, augmentHTMLDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".htm") {
			return err
		}
		base := strings.TrimSuffix(path.Base(p), ".htm")
		idText, _, _ := strings.Cut(base, "-")
		id, convErr := strconv.Atoi(idText)
		if convErr != nil {
			return nil
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte(augmentRefineMarker)) || bytes.Contains(b, []byte(augmentBypassMarker)) {
			seen[id] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("augmentation NPCs: %w", err)
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("augmentation NPCs: no page links the augmentation pages")
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

// readAugmentOptions reads the option files (Mobius OptionData), sorted by
// ID.
func readAugmentOptions(src fs.FS) ([]data.AugmentOption, error) {
	entries, err := fs.ReadDir(src, augmentOptionsDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", augmentOptionsDir, err)
	}
	var opts []data.AugmentOption
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		root, err := readXMLTree(src, path.Join(augmentOptionsDir, e.Name()))
		if err != nil {
			return nil, err
		}
		if root.name != "list" {
			return nil, root.errorf("want <list>")
		}
		if err := root.checkChildren("option"); err != nil {
			return nil, err
		}
		for _, n := range root.children {
			o, err := readAugmentOption(n)
			if err != nil {
				return nil, err
			}
			opts = append(opts, o)
		}
	}
	slices.SortStableFunc(opts, func(a, b data.AugmentOption) int { return a.ID - b.ID })
	for i := 1; i < len(opts); i++ {
		if opts[i].ID == opts[i-1].ID {
			return nil, fmt.Errorf("option %d is defined twice", opts[i].ID)
		}
	}
	return opts, nil
}

// readAugmentOption reads one <option>.
func readAugmentOption(n *xmlNode) (data.AugmentOption, error) {
	if err := n.checkAttrs("id", "name"); err != nil {
		return data.AugmentOption{}, err
	}
	id, err := n.intAttr("id")
	if err != nil {
		return data.AugmentOption{}, err
	}
	o := data.AugmentOption{ID: id}
	o.Name, _ = n.attr("name")
	if err := n.checkChildren("stats", "active_skill", "passive_skill", "attack_skill", "magic_skill", "critical_skill"); err != nil {
		return o, err
	}
	for _, c := range n.children {
		switch c.name {
		case "stats":
			if len(c.attrs) > 0 {
				return o, c.errorf("option %d: unexpected attributes", id)
			}
			for _, f := range c.children {
				fn, err := readAugmentFunc(f, id)
				if err != nil {
					return o, err
				}
				o.Funcs = append(o.Funcs, fn)
			}
		case "active_skill", "passive_skill":
			ref, _, err := readOptionSkill(c, false)
			if err != nil {
				return o, err
			}
			dst := &o.ActiveSkill
			if c.name == "passive_skill" {
				dst = &o.PassiveSkill
			}
			if *dst != nil {
				return o, c.errorf("option %d: a second <%s>", id, c.name)
			}
			*dst = &ref
		default:
			ref, chance, err := readOptionSkill(c, true)
			if err != nil {
				return o, err
			}
			o.Triggers = append(o.Triggers, data.AugmentTrigger{Type: strings.TrimSuffix(c.name, "_skill"), ID: ref.ID, Level: ref.Level, Chance: chance})
		}
	}
	return o, nil
}

// readAugmentFunc reads a stat function of <stats> as OptionData.parseFuncs
// does: stat, val and an optional order (a negative one takes the
// operation's default).
func readAugmentFunc(n *xmlNode, id int) (data.AugmentFunc, error) {
	op, ok := data.ParseStatOp(n.name)
	if !ok {
		return data.AugmentFunc{}, n.errorf("option %d: unknown stat function", id)
	}
	if err := n.checkAttrs("stat", "val", "order"); err != nil {
		return data.AugmentFunc{}, err
	}
	if len(n.children) > 0 {
		return data.AugmentFunc{}, n.errorf("option %d: unexpected child elements", id)
	}
	name, _ := n.attr("stat")
	stat, ok := data.ParseStat(name)
	if !ok {
		return data.AugmentFunc{}, n.errorf("option %d: unknown stat %q", id, name)
	}
	v, err := n.floatAttr("val")
	if err != nil {
		return data.AugmentFunc{}, err
	}
	f := data.AugmentFunc{Stat: stat, Op: op, Order: op.DefaultOrder(), Value: v}
	if _, has := n.attr("order"); has {
		order, err := n.intAttr("order")
		if err != nil {
			return data.AugmentFunc{}, err
		}
		if order >= 0 {
			f.Order = order
		}
	}
	return f, nil
}

// readOptionSkill reads the id and level of a skill element, and the chance
// of a trigger.
func readOptionSkill(n *xmlNode, trigger bool) (data.SkillRef, float64, error) {
	allowed := []string{"id", "level"}
	if trigger {
		allowed = append(allowed, "chance")
	}
	if err := n.checkAttrs(allowed...); err != nil {
		return data.SkillRef{}, 0, err
	}
	if len(n.children) > 0 {
		return data.SkillRef{}, 0, n.errorf("unexpected child elements")
	}
	id, err := n.intAttr("id")
	if err != nil {
		return data.SkillRef{}, 0, err
	}
	level, err := n.intAttr("level")
	if err != nil {
		return data.SkillRef{}, 0, err
	}
	var chance float64
	if trigger {
		if chance, err = n.floatAttr("chance"); err != nil {
			return data.SkillRef{}, 0, err
		}
	}
	return data.SkillRef{ID: id, Level: level}, chance, nil
}

// readAugmentChances reads retailchances.xml in file order, one group per
// <category>, with the chances as Java floats.
func readAugmentChances(src fs.FS) ([]data.AugmentChanceGroup, error) {
	root, err := readXMLTree(src, augmentChancesFile)
	if err != nil {
		return nil, err
	}
	if root.name != "list" {
		return nil, root.errorf("want <list>")
	}
	if err := root.checkChildren("weapon"); err != nil {
		return nil, err
	}
	var groups []data.AugmentChanceGroup
	for _, w := range root.children {
		if err := w.checkAttrs("type"); err != nil {
			return nil, err
		}
		weapon, _ := w.attr("type")
		if err := w.checkChildren("stone"); err != nil {
			return nil, err
		}
		for _, s := range w.children {
			if err := s.checkAttrs("id"); err != nil {
				return nil, err
			}
			stone, err := s.intAttr("id")
			if err != nil {
				return nil, err
			}
			if err := s.checkChildren("variation"); err != nil {
				return nil, err
			}
			for _, v := range s.children {
				if err := v.checkAttrs("id"); err != nil {
					return nil, err
				}
				variation, err := v.intAttr("id")
				if err != nil {
					return nil, err
				}
				if err := v.checkChildren("category"); err != nil {
					return nil, err
				}
				for _, c := range v.children {
					g, err := readChanceCategory(c, weapon, stone, variation)
					if err != nil {
						return nil, err
					}
					groups = append(groups, g)
				}
			}
		}
	}
	return groups, nil
}

// readChanceCategory reads one <category> with its <augment> elements.
func readChanceCategory(c *xmlNode, weapon string, stone, variation int) (data.AugmentChanceGroup, error) {
	if err := c.checkAttrs("probability"); err != nil {
		return data.AugmentChanceGroup{}, err
	}
	prob, err := c.intAttr("probability")
	if err != nil {
		return data.AugmentChanceGroup{}, err
	}
	if err := c.checkChildren("augment"); err != nil {
		return data.AugmentChanceGroup{}, err
	}
	g := data.AugmentChanceGroup{Weapon: weapon, Stone: stone, Variation: variation, Category: prob}
	for _, a := range c.children {
		if err := a.checkAttrs("id", "chance"); err != nil {
			return g, err
		}
		id, err := a.intAttr("id")
		if err != nil {
			return g, err
		}
		text, _ := a.attr("chance")
		chance, err := strconv.ParseFloat(strings.TrimSpace(text), 32)
		if err != nil {
			return g, a.errorf("chance %q is not a float", text)
		}
		g.Augments = append(g.Augments, data.AugmentChance{ID: id, Chance: float32(chance)})
	}
	return g, nil
}
