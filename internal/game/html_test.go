package game

import (
	"path/filepath"
	"testing"
)

func TestHTMLLoader(t *testing.T) {
	dir := t.TempDir()
	writeDialogFile(t, filepath.Join(dir, "default", "1.htm"), "<html>\r\n<body>\tA<!-- one\n-->B<!--two-->C<!-- open</body></html>\n")
	writeDialogFile(t, filepath.Join(dir, "page.html"), "<html><body>é</body></html>")
	writeDialogFile(t, filepath.Join(dir, "notes.txt"), "<html>not a page</html>")
	writeDialogFile(t, filepath.Join(dir, "bad.htm"), "<html>\xff</html>")
	l := newHTMLLoader(dir)

	for _, tc := range []struct {
		rel  string
		want string
		ok   bool
	}{
		// Comments go, then tabs and line feeds; carriage returns stay, and
		// so does an unclosed comment.
		{"default/1.htm", "<html>\r<body>ABC<!-- open</body></html>", true},
		{"default//1.htm", "<html>\r<body>ABC<!-- open</body></html>", true},
		{"page.html", "<html><body>é</body></html>", true},
		{"bad.htm", "<html>�</html>", true},
		{"notes.txt", "", false},
		{"missing.htm", "", false},
		{"../outside.htm", "", false},
		{"default/../page.html", "", false},
		{"/etc/passwd.htm", "", false},
		{"", "", false},
	} {
		got, ok := l.get(tc.rel)
		if got != tc.want || ok != tc.ok {
			t.Errorf("get(%q) = %q, %v; want %q, %v", tc.rel, got, ok, tc.want, tc.ok)
		}
	}

	// Pages are read once: a change on disk shows only in a new loader, and
	// so does a page added later.
	writeDialogFile(t, filepath.Join(dir, "page.html"), "<html>changed</html>")
	writeDialogFile(t, filepath.Join(dir, "missing.htm"), "<html>now here</html>")
	if got, _ := l.get("page.html"); got != "<html><body>é</body></html>" {
		t.Errorf("cached page = %q", got)
	}
	if l.exists("missing.htm") {
		t.Error("a missing page appeared without a new loader")
	}
	if got, _ := newHTMLLoader(dir).get("page.html"); got != "<html>changed</html>" {
		t.Errorf("new loader page = %q", got)
	}
}

func TestStripHTMLComments(t *testing.T) {
	for in, want := range map[string]string{
		"":                       "",
		"no comment":             "no comment",
		"a<!--x-->b":             "ab",
		"a<!--x-->b<!--y-->c":    "abc",
		"a<!--x--y-->b":          "ab",
		"a<!--<!--x-->-->b":      "a-->b",
		"a<!--never closed":      "a<!--never closed",
		"a<!--x-->b<!--unclosed": "ab<!--unclosed",
	} {
		if got := stripHTMLComments(in); got != want {
			t.Errorf("stripHTMLComments(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPagePlaceholders(t *testing.T) {
	pg := newPage(7, "no html tag %objectId% %objectId% %npcname%").replaceObjectID(7).replace("%npcname%", "Roxxy")
	if want := "<html><body>no html tag 7 7 Roxxy</body></html>"; pg.html != want || pg.npc != 7 {
		t.Errorf("page %q of %d, want %q of 7", pg.html, pg.npc, want)
	}
}
