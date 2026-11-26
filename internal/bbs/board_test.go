package bbs

import (
	"slices"
	"strings"
	"testing"
)

// HtmlUtil.sendCBHtml: below 4,090 units one part, below 8,180 two, below
// 12,270 three, cut at 4,090 and 8,180; longer pages show the error.
func TestParts(t *testing.T) {
	page := func(n int) string { return strings.Repeat("a", n) }
	for _, tt := range []struct {
		n    int
		want [3]int // part lengths, -1 for MissingPart
	}{
		{0, [3]int{0, -1, -1}},
		{4089, [3]int{4089, -1, -1}},
		{4090, [3]int{4090, 0, -1}},
		{8179, [3]int{4090, 4089, -1}},
		{8180, [3]int{4090, 4090, 0}},
		{12269, [3]int{4090, 4090, 4089}},
	} {
		got := Parts(page(tt.n))
		for i, want := range tt.want {
			switch {
			case want < 0 && got[i] != MissingPart:
				t.Errorf("Parts(%d)[%d] = %d units, want %q", tt.n, i, len(got[i]), MissingPart)
			case want >= 0 && len(got[i]) != want:
				t.Errorf("Parts(%d)[%d] = %d units, want %d", tt.n, i, len(got[i]), want)
			}
		}
	}
	if got := Parts(page(12270)); got != [3]string{TooLongPage, MissingPart, MissingPart} {
		t.Errorf("Parts(12270) = %.40q", got)
	}
	// UTF-16 units, not bytes: 4,089 "é" fit in one part.
	if got := Parts(strings.Repeat("é", 4089)); got[1] != MissingPart {
		t.Error("4,089 two-byte characters were split")
	}
	// A surrogate pair across the cut goes whole into the next part.
	got := Parts(page(4089) + "😀" + "b")
	if got[0] != page(4089) || got[1] != "😀b" {
		t.Errorf("surrogate pair cut: %q / %q", got[0][4080:], got[1])
	}
	if Abbreviate("abcdef", 3) != "abc" || Abbreviate("ab", 3) != "ab" {
		t.Error("Abbreviate")
	}
}

func TestRoute(t *testing.T) {
	for cmd, want := range map[string]Board{
		"_bbshome":                     BoardHome,
		"_bbstop;region/region.html":   BoardHome,
		"_bbstopics":                   BoardMemo, // the longest command wins
		"_bbsgetfav":                   BoardFavorite,
		"bbs_add_fav":                  BoardFavorite,
		"_bbsdelfav_3":                 BoardFavorite,
		"_bbsloc;2":                    BoardRegion,
		"_bbsclan":                     BoardClan,
		"_bbsclan_clanhome;5;announce": BoardClan,
		"_BBSCLAN":                     BoardClan, // ignoring case
		"_bbsmemo":                     BoardMemo,
		"_bbsmail;view;1":              BoardMail,
		"_maillist_0_1_0_":             BoardMail,
		"_friendlist":                  BoardFriends,
		"_friend;select;5":             BoardFriends,
		"_block;del":                   BoardFriends,
		"_bbslink":                     BoardHomepage,
	} {
		if got, ok := Route(cmd); !ok || got != want {
			t.Errorf("Route(%q) = %v, %v; want %v", cmd, got, ok, want)
		}
	}
	// The custom board's and the drop search's commands are not registered,
	// nor the tab's "_bbsfriends" (Mobius has no such command either).
	for _, cmd := range []string{"_bbsteleport;Giran", "_bbsmultisell;1,a", "_bbs_search_item", "_bbsfriends", "_match", "bbs"} {
		if got, ok := Route(cmd); ok {
			t.Errorf("Route(%q) = %v, want no board", cmd, got)
		}
	}
}

func TestWriteTarget(t *testing.T) {
	for url, want := range map[string]Board{
		"Topic": BoardHome, "Post": BoardNone, "Region": BoardRegion, "Notice": BoardClan,
		"Search": BoardClan, "CMail": BoardClan, "Mail": BoardMail,
	} {
		if got, ok := WriteTarget(url); !ok || got != want {
			t.Errorf("WriteTarget(%q) = %v, %v; want %v", url, got, ok, want)
		}
	}
	if _, ok := WriteTarget("mail"); ok {
		t.Error("WriteTarget is case-sensitive in Mobius")
	}
	if got := NotImplementedPage("X"); got != "<html><body><br><br><center>The command: X is not implemented yet.</center><br><br></body></html>" {
		t.Errorf("NotImplementedPage = %q", got)
	}
}

func TestJavaHelpers(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want []string
	}{
		{"", []string{""}},
		{"Bob", []string{"Bob"}},
		{"Bob;Ann", []string{"Bob", "Ann"}},
		{"Bob;;Ann;;", []string{"Bob", "", "Ann"}},
		{";;", []string{}},
		{";Bob", []string{"", "Bob"}},
	} {
		if got := SplitJava(tt.in, ";"); !slices.Equal(got, tt.want) {
			t.Errorf("SplitJava(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := Tokens("_bbsmail;;view;3", ';'); !slices.Equal(got, []string{"_bbsmail", "view", "3"}) {
		t.Errorf("Tokens = %q", got)
	}
	if !IsNumeric("012") || IsNumeric("") || IsNumeric("-1") || IsNumeric("1a") {
		t.Error("IsNumeric")
	}
	if JavaTrim(" \t x \r\n") != "x" {
		t.Error("JavaTrim")
	}
}
