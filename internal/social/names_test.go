package social

import "testing"

func TestCheckClanName(t *testing.T) {
	anyName, err := CompileNameTemplate(".*")
	if err != nil {
		t.Fatal(err)
	}
	letters, err := CompileNameTemplate("[A-Za-z]+")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		tpl  bool // use the letters-only template
		want NameCheck
	}{
		{"Ab", false, NameOK},
		{"Clan1", false, NameOK},
		{"A", false, NameInvalid},
		{"", false, NameInvalid},
		{"Sixteen16Chars00", false, NameOK},
		{"Seventeen17Chars0", false, NameBadLength},
		{"Two Words", false, NameInvalid},
		{"Under_score", false, NameInvalid},
		{"Ünïcödé", false, NameOK},
		{"名前", false, NameOK},
		{"Ab😀", false, NameInvalid},
		{"Ab\xff", false, NameInvalid},
		{"Clan1", true, NameInvalid},
		{"Clan", true, NameOK},
		{"SeventeenLetterss", true, NameBadLength},
		{"SeventeenLetters1", true, NameInvalid},
	}
	for _, tt := range tests {
		tpl := anyName
		if tt.tpl {
			tpl = letters
		}
		if got := CheckClanName(tt.name, tpl); got != tt.want {
			t.Errorf("CheckClanName(%q, tpl %v) = %v, want %v", tt.name, tt.tpl, got, tt.want)
		}
	}
	if CheckClanName("Ab", nil) != NameOK {
		t.Error("nil template")
	}
}

func TestNameTemplateWhole(t *testing.T) {
	re, err := CompileNameTemplate("a|b")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("a") || re.MatchString("ab") {
		t.Error("template must match the whole name")
	}
	if _, err := CompileNameTemplate("(a"); err == nil {
		t.Error("bad template compiled")
	}
}

func TestCheckAllyName(t *testing.T) {
	tests := []struct {
		name string
		want NameCheck
	}{
		{"Ally", NameOK},
		{"A", NameBadLength},
		{"", NameInvalid},
		{"A!", NameInvalid},
		{"Sixteen16Chars00", NameOK},
		{"Seventeen17Chars0", NameBadLength},
		{"Two Words", NameInvalid},
	}
	for _, tt := range tests {
		if got := CheckAllyName(tt.name); got != tt.want {
			t.Errorf("CheckAllyName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
	if NameKey("MyClan") != NameKey("myCLAN") {
		t.Error("NameKey is case sensitive")
	}
}
