package htmlutil

import (
	"slices"
	"strings"
	"testing"
)

// merchantPage is Mobius html/merchant/30001.htm with %objectId% replaced.
const merchantPage = `<html><body>Trader Lector:<br>
May I give you a piece of advice? The most expensive equipment is not always the best.<br>
<a action="bypass -h npc_1610612737_Buy 3000100">Buy Fighter equipment.</a><br>
<a action="bypass -h npc_1610612737_Buy 3000101">Buy Mystic equipment.</a><br>
<a action="bypass -h npc_1610612737_exc_multisell 003">Exchange equipment.</a><br>
<a action="bypass -h npc_1610612737_Wear 3000100">Wear Fighter equipment.</a><br>
<a action="bypass -h npc_1610612737_Sell">Sell.</a><br>
<a action="bypass npc_1610612737_TerritoryStatus">See the Lord and get the tax rate information.</a><br>
<a action="bypass Script">Quest</a>
</body></html>`

// chatPage has Chat and Link bypasses, and quotes in the link texts.
const chatPage = `<html><body>Gatekeeper Roxxy:<br>
Welcome to Talking Island Village.<br>
<a action="bypass -h npc_1610612740_Chat 1">"I want to go somewhere."</a><br>
<a action="bypass npc_1610612740_Link seven_signs/mammmerch_1a.htm">"Tell me about Ancient Adena."</a><br>
<a action="bypass npc_1610612740_showTeleports">Teleport</a><br>
<a action="bypass Script">Quest</a>
</body></html>`

// tutorialPage is Mobius quests/Q00255_Tutorial/tutorial_20.htm: a link
// action, and attributes that are not actions.
const tutorialPage = `<html><body>
<center><font color="LEVEL">[Penalty for Dying]</font></center><br>
From now on you'll be penalized for dying. You must avoid death in order to preserve your Exp.<br>
<a action="link TE00">Return</a>
</body></html>`

// adminPage has bypasses with parameters the client fills in from input
// fields.
const adminPage = `<html><body>
<edit var="exp_to_add" width=50><edit var="sp_to_add" width=50>
<button value="Add" action="bypass -h admin_add_exp_sp $exp_to_add $sp_to_add" width=60 height=15 back="L2UI_ct1.button_df" fore="L2UI_ct1.button_df">
<button value="Go" action="bypass -h npc_1610612741_teleport $dest now" width=60 height=15>
</body></html>`

// nonASCIIPage has letters before its action whose Unicode lower case is
// shorter in UTF-8: the capital I with a dot (U+0130) and the Kelvin sign
// (U+212A). Lower-casing them would shift the offsets.
const nonASCIIPage = "<html><body>\U000000DCnnepi \U00000130stv\U000000E1n, 3\U0000212A:<br>" +
	`<a action="bypass npc_7_Chat 0">Szia</a></body></html>`

func TestActions(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{"merchant", merchantPage, []string{
			"npc_1610612737_Buy 3000100",
			"npc_1610612737_Buy 3000101",
			"npc_1610612737_exc_multisell 003",
			"npc_1610612737_Wear 3000100",
			"npc_1610612737_Sell",
			"npc_1610612737_TerritoryStatus",
			"Script",
		}},
		{"chat and link bypasses", chatPage, []string{
			"npc_1610612740_Chat 1",
			"npc_1610612740_Link seven_signs/mammmerch_1a.htm",
			"npc_1610612740_showTeleports",
			"Script",
		}},
		{"link", tutorialPage, []string{"link TE00"}},
		{"parameters: cached up to the first $", adminPage, []string{
			"admin_add_exp_sp $",
			"npc_1610612741_teleport $",
		}},
		{"non-ASCII text before the action", nonASCIIPage, []string{"npc_7_Chat 0"}},
		{
			"bypasses before links",
			`<a action="link TE01">a</a><a action="bypass npc_9_Chat 1">b</a><a action="link TE02">c</a>`,
			[]string{"npc_9_Chat 1", "link TE01", "link TE02"},
		},
		{
			"markers in any case, text case kept",
			`<A ACTION="BYPASS -H npc_5_Chat 2">a</A><A Action="Link TE01">b</A>`,
			[]string{"npc_5_Chat 2", "link TE01"},
		},
		{
			"-h after the attribute is not the flag",
			`<a action="bypass npc_1_Chat 1">use -h here</a>`,
			[]string{"npc_1_Chat 1"},
		},
		{
			"-h drops everything before it",
			`<a action="bypass npc_1_x -h npc_1_y">a</a>`,
			[]string{"npc_1_y"},
		},
		{
			"trimmed",
			"<a action=\"bypass \t npc_1_Chat 3 \n\">a</a><a action=\"link  TE03 \">b</a>",
			[]string{"npc_1_Chat 3", "link TE03"},
		},
		{
			"empty bypasses left out",
			`<a action="bypass ">a</a><a action="bypass -h ">b</a><a action="bypass Script">c</a>`,
			[]string{"Script"},
		},
		{
			"empty links and links with .. left out, the next one found",
			`<a action="link ">a</a><a action="link ../../config/server.ini">b</a><a action="link TE01">c</a>`,
			[]string{"link TE01"},
		},
		{
			"no closing quote: nothing from there on",
			`<a action="bypass npc_1_Chat 1">ok</a><a action="bypass npc_1_Chat 2>broken</a>`,
			[]string{"npc_1_Chat 1"},
		},
		{
			"repeats kept",
			`<a action="bypass Script">Quest</a><br><a action="bypass Script">Quest</a>`,
			[]string{"Script", "Script"},
		},
		{
			"other quoting is not an action",
			`<a action='bypass npc_1_Chat 1'>a</a><a action = "bypass npc_1_Chat 2">b</a><a action="bypass-h x">c</a>`,
			nil,
		},
		{"no actions", "<html><body>Hello.</body></html>", nil},
		{"empty page", "", nil},
	}
	for _, tt := range tests {
		if got := Actions(tt.html); !slices.Equal(got, tt.want) {
			t.Errorf("%s: Actions() =\n%q\nwant\n%q", tt.name, got, tt.want)
		}
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		cached, action string
		want           bool
	}{
		{"npc_1610612737_Chat 1", "npc_1610612737_Chat 1", true},
		{"npc_1610612737_Chat 1", "npc_1610612737_Chat 10", false},
		{"npc_1610612737_Chat 1", "npc_1610612737_Chat", false},
		{"npc_1610612737_Chat 1", "npc_1610612738_Chat 1", false},
		{"Script", "script", false},
		{"link TE00", "link TE00", true},
		{"link TE00", "link TE01", false},
		{"admin_add_exp_sp $", "admin_add_exp_sp 100 20", true},
		{"admin_add_exp_sp $", "admin_add_exp_sp", true},
		{"admin_add_exp_sp $", "admin_add_exp_spree", true}, // a plain prefix, as in Mobius
		{"admin_add_exp_sp $", "admin_kill", false},
		{"$", "admin_kill", true}, // as in Mobius
		{"", "", false},
		{"", "Script", false},
	}
	for _, tt := range tests {
		if got := Matches(tt.cached, tt.action); got != tt.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", tt.cached, tt.action, got, tt.want)
		}
	}
}

// Every action of a page passes the check against itself, so a player can
// click everything a page offers.
func TestActionsMatchThemselves(t *testing.T) {
	for _, page := range []string{merchantPage, chatPage, tutorialPage, adminPage, nonASCIIPage} {
		for _, a := range Actions(page) {
			if !Matches(a, a) {
				t.Errorf("Matches(%q, %q) = false", a, a)
			}
		}
	}
}

func FuzzActions(f *testing.F) {
	for _, page := range []string{
		merchantPage, chatPage, tutorialPage, adminPage, nonASCIIPage,
		`="bypass "`, `="bypass -h "`, `="link "`, `="link .."`, `="BYPASS -H x $y"`, `="bypass`,
	} {
		f.Add(page)
	}
	f.Fuzz(func(t *testing.T, html string) {
		lower := asciiLower(html)
		if len(lower) != len(html) {
			t.Fatalf("asciiLower changed the length: %d, want %d", len(lower), len(html))
		}
		for _, b := range appendBypasses(nil, html, lower) {
			if b == "" || javaTrim(b) != b {
				t.Errorf("bypass %q is empty or not trimmed", b)
			}
			if i := strings.IndexByte(b, ParamChar); i >= 0 && i != len(b)-1 {
				t.Errorf("bypass %q goes on after its parameter", b)
			}
		}
		for _, l := range appendLinks(nil, html, lower) {
			path, ok := strings.CutPrefix(l, LinkPrefix)
			if !ok || path == "" || javaTrim(path) != path || strings.Contains(path, "..") {
				t.Errorf("bad link action %q", l)
			}
		}
		for _, a := range Actions(html) {
			if !Matches(a, a) {
				t.Errorf("Matches(%q, %q) = false", a, a)
			}
		}
	})
}
