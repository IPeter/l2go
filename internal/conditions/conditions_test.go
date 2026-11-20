package conditions

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
)

// player is a Subject for the tests.
type player struct {
	race    data.Race
	level   int
	female  bool
	pk      int
	hero    bool
	flagged bool
	cats    []data.CategoryType
	clan    Clan
}

func (p player) Race() data.Race                     { return p.race }
func (p player) Level() int                          { return p.level }
func (p player) Female() bool                        { return p.female }
func (p player) PKKills() int                        { return p.pk }
func (p player) Hero() bool                          { return p.hero }
func (p player) PvPFlagged() bool                    { return p.flagged }
func (p player) InCategory(t data.CategoryType) bool { return slices.Contains(p.cats, t) }
func (p player) Clan() Clan                          { return p.clan }

func el(name string, attrs map[string]string, children ...data.XMLElement) data.XMLElement {
	return data.XMLElement{Name: name, Attrs: attrs, Children: children}
}

func pl(attrs map[string]string) data.XMLElement { return el("player", attrs) }

func cat(t *testing.T, name string) data.CategoryType {
	t.Helper()
	c, ok := data.ParseCategoryType(name)
	if !ok {
		t.Fatalf("no category %q", name)
	}
	return c
}

func TestConditions(t *testing.T) {
	wolf := cat(t, "WOLF")
	strider := cat(t, "STRIDER")
	human := player{race: data.RaceHuman, level: 20}
	tests := []struct {
		name string
		cond data.XMLElement
		p    player
		want bool
	}{
		{"race in list", pl(map[string]string{"races": "HUMAN,ELF"}), human, true},
		{"race not in list", pl(map[string]string{"races": "ELF,DARK_ELF"}), human, false},
		{"level reached", pl(map[string]string{"level": "20"}), human, true},
		{"level too low", pl(map[string]string{"level": "21"}), human, false},
		{"sex male", pl(map[string]string{"sex": "0"}), human, true},
		{"sex female wanted", pl(map[string]string{"sex": "1"}), human, false},
		{"sex female", pl(map[string]string{"sex": "1"}), player{female: true}, true},
		{"pk count at most", pl(map[string]string{"pkCount": "0"}), player{pk: 0}, true},
		{"pk count above", pl(map[string]string{"pkCount": "0"}), player{pk: 1}, false},
		{"hero wanted, not hero", pl(map[string]string{"isHero": "true"}), human, false},
		{"hero wanted, hero", pl(map[string]string{"isHero": "TRUE"}), player{hero: true}, true},
		{"not hero wanted", pl(map[string]string{"isHero": "false"}), human, true},
		{"pvp flag", pl(map[string]string{"isPvpFlagged": "false"}), human, true},
		{"pvp flagged", pl(map[string]string{"isPvpFlagged": "false"}), player{flagged: true}, false},
		{"category any", pl(map[string]string{"categoryType": "WOLF,STRIDER"}), player{cats: []data.CategoryType{strider}}, true},
		{"category none", pl(map[string]string{"categoryType": "WOLF"}), human, false},
		{"category one", pl(map[string]string{"categoryType": "WOLF"}), player{cats: []data.CategoryType{wolf}}, true},
		// Clan conditions as Mobius tests them for a player without a clan.
		{"pledge class, no clan", pl(map[string]string{"pledgeClass": "5"}), human, false},
		{"pledge class -1, no clan", pl(map[string]string{"pledgeClass": "-1"}), human, false},
		{"pledge class, leader", pl(map[string]string{"pledgeClass": "5"}), player{clan: Clan{ID: 1, Leader: true}}, true},
		{"pledge class -1, member", pl(map[string]string{"pledgeClass": "-1"}), player{clan: Clan{ID: 1, PledgeClass: 9}}, false},
		{"pledge class reached", pl(map[string]string{"pledgeClass": "5"}), player{clan: Clan{ID: 1, PledgeClass: 5}}, true},
		{"pledge class too low", pl(map[string]string{"pledgeClass": "5"}), player{clan: Clan{ID: 1, PledgeClass: 4}}, false},
		{"castle 0, no clan", pl(map[string]string{"castle": "0"}), human, true},
		{"castle -1, no clan", pl(map[string]string{"castle": "-1"}), human, false},
		{"castle 3, no clan", pl(map[string]string{"castle": "3"}), human, false},
		{"castle -1, clan without castle", pl(map[string]string{"castle": "-1"}), player{clan: Clan{ID: 1}}, false},
		{"castle -1, clan with castle", pl(map[string]string{"castle": "-1"}), player{clan: Clan{ID: 1, CastleID: 2}}, true},
		{"castle 2, clan with castle 2", pl(map[string]string{"castle": "2"}), player{clan: Clan{ID: 1, CastleID: 2}}, true},
		{"clan hall 0, no clan", pl(map[string]string{"clanHall": "0"}), human, true},
		{"clan hall -1, no clan", pl(map[string]string{"clanHall": "-1"}), human, false},
		{"clan hall -1, clan with hall", pl(map[string]string{"clanHall": "-1"}), player{clan: Clan{ID: 1, HideoutID: 22}}, true},
		{"clan hall list", pl(map[string]string{"clanHall": "21, 22"}), player{clan: Clan{ID: 1, HideoutID: 22}}, true},
		{"clan hall list, other hall", pl(map[string]string{"clanHall": "21,22"}), player{clan: Clan{ID: 1, HideoutID: 23}}, false},
		{"two attributes are and", pl(map[string]string{"races": "HUMAN", "level": "30"}), human, false},
		{"and true", el("and", nil, pl(map[string]string{"races": "HUMAN"}), pl(map[string]string{"level": "1"})), human, true},
		{"and false", el("and", nil, pl(map[string]string{"races": "HUMAN"}), pl(map[string]string{"castle": "-1"})), human, false},
		{"or true", el("or", nil, pl(map[string]string{"races": "ELF"}), pl(map[string]string{"level": "1"})), human, true},
		{"or false", el("or", nil, pl(map[string]string{"races": "ELF"}), pl(map[string]string{"level": "99"})), human, false},
		{"not", el("not", nil, pl(map[string]string{"races": "ELF"})), human, true},
		{"not not", el("not", nil, el("not", nil, pl(map[string]string{"races": "ELF"}))), human, false},
		{"hex level", pl(map[string]string{"level": "0x14"}), human, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Parse(&tt.cond)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := c.Test(tt.p); got != tt.want {
				t.Errorf("Test = %v, want %v (condition %+v)", got, tt.want, c)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		cond        data.XMLElement
		unsupported bool
	}{
		{"unknown race", pl(map[string]string{"races": "HUMAN, ELF"}), false},
		{"unknown category", pl(map[string]string{"categoryType": "NOPE"}), false},
		{"bad level", pl(map[string]string{"level": "x"}), false},
		{"empty player", pl(nil), false},
		{"empty and", el("and", nil), false},
		{"empty not", el("not", nil), false},
		{"unknown attribute", pl(map[string]string{"hp": "50"}), true},
		{"target", el("target", map[string]string{"level": "1"}), true},
		{"unknown element", el("foo", nil), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(&tt.cond)
			if err == nil {
				t.Fatal("Parse succeeded, want an error")
			}
			if got := errors.Is(err, ErrUnsupported); got != tt.unsupported {
				t.Errorf("errors.Is(err, ErrUnsupported) = %v, want %v (%v)", got, tt.unsupported, err)
			}
		})
	}
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true}, {"76", 76, true}, {"-1", -1, true}, {"+5", 5, true}, {"0x10", 16, true},
		{"#10", 16, true}, {"010", 8, true}, {"", 0, false}, {"--1", 0, false}, {"1_0", 0, false},
		{"2147483648", 0, false}, {"-2147483648", -2147483648, true},
	}
	for _, tt := range tests {
		got, err := decodeInt(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("decodeInt(%q) = %d, %v; want %d, ok %v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func TestParseItem(t *testing.T) {
	human := player{race: data.RaceHuman, level: 10}
	tests := []struct {
		name string
		elem *data.XMLElement
		want Result
	}{
		{"nil", nil, Result{OK: true}},
		{"empty", &data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "1518"}}, Result{OK: true}},
		{
			"passes",
			&data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "1518"}, Children: []data.XMLElement{pl(map[string]string{"races": "HUMAN"})}},
			Result{OK: true},
		},
		{
			"fails with msgId",
			&data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "1518"}, Children: []data.XMLElement{pl(map[string]string{"races": "ELF"})}},
			Result{MsgID: 1518},
		},
		{
			"fails with msgId and name",
			&data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "113", "addName": "1"}, Children: []data.XMLElement{pl(map[string]string{"level": "76"})}},
			Result{MsgID: 113, AddName: true},
		},
		{
			"addName needs a positive msgId",
			&data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "0", "addName": "1"}, Children: []data.XMLElement{pl(map[string]string{"level": "76"})}},
			Result{},
		},
		{
			"msg wins over msgId",
			&data.XMLElement{Name: "conditions", Attrs: map[string]string{"msg": "No.", "msgId": "113"}, Children: []data.XMLElement{pl(map[string]string{"level": "76"})}},
			Result{Msg: "No."},
		},
		{
			"first child only, value skipped",
			&data.XMLElement{Name: "conditions", Children: []data.XMLElement{
				el("value", nil), pl(map[string]string{"level": "1"}), pl(map[string]string{"level": "99"}),
			}},
			Result{OK: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it, err := ParseItem(tt.elem)
			if err != nil {
				t.Fatalf("ParseItem: %v", err)
			}
			if got := it.Check(human); got != tt.want {
				t.Errorf("Check = %+v, want %+v", got, tt.want)
			}
		})
	}
	if _, err := ParseItem(&data.XMLElement{Name: "stats"}); err == nil {
		t.Error("ParseItem(<stats>) succeeded, want an error")
	}
}

func TestTable(t *testing.T) {
	items := []data.Item{
		{ID: 1},
		{ID: 2, Conditions: &data.XMLElement{Name: "conditions", Attrs: map[string]string{"msgId": "1518"}, Children: []data.XMLElement{pl(map[string]string{"sex": "1"})}}},
		{ID: 3, Conditions: &data.XMLElement{Name: "conditions", Children: []data.XMLElement{pl(map[string]string{"bogus": "1"})}}},
	}
	tab, err := NewTable(items)
	if err == nil {
		t.Error("NewTable: no error for item 3")
	}
	if _, ok := tab[3]; ok {
		t.Error("item 3 is in the table")
	}
	male := player{}
	if r := tab.Check(1, male); !r.OK {
		t.Errorf("item 1: %+v, want OK", r)
	}
	if r := tab.Check(2, male); r.OK || r.MsgID != 1518 {
		t.Errorf("item 2, male: %+v, want message 1518", r)
	}
	if r := tab.Check(2, player{female: true}); !r.OK {
		t.Errorf("item 2, female: %+v, want OK", r)
	}
	if r := tab.Check(99, male); !r.OK {
		t.Errorf("unknown item: %+v, want OK", r)
	}
}

// TestRealItems parses the conditions of every item in the generated data,
// so a condition the data uses but this package lacks shows up here.
func TestRealItems(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "data", "items", "items.json"))
	if err != nil {
		t.Skipf("no item data: %v", err)
	}
	var items []data.Item
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	tab, err := NewTable(items)
	if err != nil {
		t.Fatalf("NewTable: %v", err)
	}
	if len(tab) < 400 {
		t.Errorf("%d items with conditions, want about 500", len(tab))
	}
	// Every item with conditions fails for some player, and the common
	// race list lets every playable race pass.
	for id, c := range tab {
		if c.Cond == nil {
			t.Errorf("item %d: no condition", id)
		}
	}
	for _, r := range []data.Race{data.RaceHuman, data.RaceElf, data.RaceDarkElf, data.RaceOrc, data.RaceDwarf} {
		if res := tab.Check(firstWithRaces(t, items), player{race: r}); !res.OK {
			t.Errorf("race %v: %+v, want OK", r, res)
		}
	}
}

func firstWithRaces(t *testing.T, items []data.Item) int {
	t.Helper()
	for _, it := range items {
		c := it.Conditions
		if c != nil && len(c.Children) == 1 && c.Children[0].Name == "player" && c.Children[0].Attrs["races"] != "" {
			return it.ID
		}
	}
	t.Fatal("no item with a race condition")
	return 0
}
