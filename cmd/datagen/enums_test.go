package main

import (
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestGoConst(t *testing.T) {
	tests := []struct{ prefix, java, want string }{
		{"Stat", "MAX_HP", "StatMaxHP"},
		{"Stat", "STAT_STR", "StatSTR"},
		{"Stat", "STATUS_X", "StatStatusX"},
		{"Stat", "PVP_PHYSICAL_DMG", "StatPVPPhysicalDmg"},
		{"BodyPart", "LR_EAR", "BodyPartLREar"},
		{"BodyPart", "HAIR2", "BodyPartHair2"},
		{"Action", "SHOW_HTML", "ActionShowHTML"},
		{"ItemSkill", "ON_ENCHANT_4", "ItemSkillOnEnchant4"},
		{"Class", "DARK_AVENGER", "ClassDarkAvenger"},
		{"Material", "RUNE_XP", "MaterialRuneXP"},
		{"Weapon", "ETC", "WeaponEtc"},
	}
	for _, tt := range tests {
		if got := goConst(tt.prefix, tt.java); got != tt.want {
			t.Errorf("goConst(%q, %q) = %q, want %q", tt.prefix, tt.java, got, tt.want)
		}
	}
}

func TestParseJavaEnum(t *testing.T) {
	src := `package x;
public enum Kind implements Other
{
	// a comment, with commas; and a semicolon
	PLAIN,
	ARGS(1, "a,b)", Race.HUMAN) /* block, comment */ ,
	BODY(0x10) { @Override public int x() { return 1; } },
	LAST();

	private final int _v;
	Kind(int v) { _v = v; }
}`
	got, err := parseJavaEnum(stripJavaComments(src), "Kind")
	if err != nil {
		t.Fatal(err)
	}
	want := []javaConstant{
		{name: "PLAIN"},
		{name: "ARGS", args: []string{"1", `"a,b)"`, "Race.HUMAN"}},
		{name: "BODY", args: []string{"0x10"}},
		{name: "LAST"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseJavaEnum = %+v, want %+v", got, want)
	}

	// No semicolon and a trailing comma.
	got, err = parseJavaEnum("enum E { A, B, }", "E")
	if err != nil || len(got) != 2 || got[1].name != "B" {
		t.Errorf("trailing comma: %+v, %v", got, err)
	}

	for _, tt := range []struct{ src, want string }{
		{"enum Other { A }", "enum E not found"},
		{"enum E { A, , B }", "empty constant"},
		{"enum E { A, A }", "constant A declared twice"},
		{"enum E { A(1 }", "list is not closed"},
		{"enum E { a-b }", `cannot read enum constant`},
		{"enum E { 1A }", `cannot read enum constant "1A"`},
		{"enum E { A(1) B }", `cannot read enum constant "A(1) B"`},
		{"enum E { ; }", "has no constants"},
	} {
		_, err := parseJavaEnum(tt.src, "E")
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("parseJavaEnum(%q) = %v, want an error with %q", tt.src, err, tt.want)
		}
	}
}

func TestStripJavaComments(t *testing.T) {
	src := "a // x, y\nb /* c, d */ e \"// kept /* kept */\" '/' f /* open"
	want := "a  \nb   e \"// kept /* kept */\" '/' f "
	if got := stripJavaComments(src); got != want {
		t.Errorf("stripJavaComments = %q, want %q", got, want)
	}
}

func TestJavaLiterals(t *testing.T) {
	for in, want := range map[string]int{"0x0006": 6, "-100": -100, " 17 ": 17, "0x80000": 0x80000} {
		if got, err := javaInt(in); err != nil || got != want {
			t.Errorf("javaInt(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := javaInt("Inventory.PAPERDOLL_UNDER"); err == nil {
		t.Error("javaInt of a constant succeeded")
	}
	if s, err := javaString(`"rear;lear"`); err != nil || s != "rear;lear" {
		t.Errorf("javaString = %q, %v", s, err)
	}
	for _, bad := range []string{`rear`, `"open`, `"`} {
		if _, err := javaString(bad); err == nil {
			t.Errorf("javaString(%q) succeeded", bad)
		}
	}
	if b, err := javaBool(" true"); err != nil || !b {
		t.Errorf("javaBool = %v, %v", b, err)
	}
	if _, err := javaBool("yes"); err == nil {
		t.Error(`javaBool("yes") succeeded`)
	}
	if r, err := javaRef("Race.DARK_ELF", "Race"); err != nil || r != "DARK_ELF" {
		t.Errorf("javaRef = %q, %v", r, err)
	}
	for _, bad := range []string{"TraitType.SWORD", "Race.", "Race.A.B", "HUMAN"} {
		if _, err := javaRef(bad, "Race"); err == nil {
			t.Errorf("javaRef(%q, Race) succeeded", bad)
		}
	}
}

func TestReadJavaIntConstants(t *testing.T) {
	m := fstest.MapFS{"X.java": file(`class X {
	public static final int A = 3;
	// public static final int B = 4;
	public static final int C = -1;
	public static final long D = 5;
	private static final int E = 6;
}`)}
	got, err := readJavaIntConstants(m, "X.java")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]int{"A": 3, "C": -1}) {
		t.Errorf("readJavaIntConstants = %v", got)
	}
	if _, err := readJavaIntConstants(m, "Y.java"); err == nil {
		t.Error("reading a missing file succeeded")
	}
}

func TestEnumErrors(t *testing.T) {
	tests := []sampleCase{
		{"stat arguments", sampleEdit(statFile, `MAX_HP("maxHp")`, `MAX_HP("maxHp", 1)`), "Stat.MAX_HP has 2 arguments, want 1 (the XML name)"},
		{"stat XML name", sampleEdit(statFile, `MAX_HP("maxHp")`, `MAX_HP(maxHp)`), "Stat.MAX_HP: not a string literal"},
		{"stat XML name twice", sampleEdit(statFile, `MAX_MP("maxMp")`, `MAX_MP("maxHp")`), `enum Stat: name "maxHp" is used by MAX_HP and MAX_MP`},
		{"body part slot", sampleEdit(bodyPartFile, "Inventory.PAPERDOLL_CHEST", "Inventory.PAPERDOLL_BELT"), `BodyPart.CHEST: slot "Inventory.PAPERDOLL_BELT" is neither an int nor an Inventory constant`},
		{"body part name twice", sampleEdit(bodyPartFile, `"lear"`, `"rear"`), `name "rear" is used by BodyPart.R_EAR and BodyPart.L_EAR`},
		{"body part without names", sampleEdit(bodyPartFile, `WOLF(-100, -1, "wolf")`, `WOLF(-100, -1)`), "BodyPart.WOLF has 2 arguments"},
		{"no total slots", both(sampleEdit(inventoryFile, "PAPERDOLL_TOTALSLOTS", "PAPERDOLL_COUNT"),
			sampleEdit(bodyPartFile, "Inventory.PAPERDOLL_TOTALSLOTS", "17")), "PAPERDOLL_TOTALSLOTS not found"},
		{"weapon trait", sampleEdit(weaponTypeFile, "SWORD(TraitType.SWORD)", "SWORD(Trait.SWORD)"), `WeaponType.SWORD: not a TraitType constant: "Trait.SWORD"`},
		{"armor arguments", sampleEdit(armorTypeFile, "LIGHT,", "LIGHT(1),"), "ArmorType.LIGHT has 1 arguments, want 0"},
		{"crystal arguments", sampleEdit(crystalTypeFile, "D(1, 1458, 11, 90)", "D(1, 1458, 11)"), "CrystalType.D has 3 arguments, want 4"},
		{"crystal number", sampleEdit(crystalTypeFile, "D(1, 1458, 11, 90)", "D(1, CRYSTAL, 11, 90)"), "CrystalType.D: not an int literal"},
		{"no NONE race", sampleEdit(raceFile, "\tNONE;", "\tUNDEAD;"), "enum Race has no NONE constant"},
		{"duplicate Go name", sampleEdit(materialTypeFile, "FINE_STEEL,", "FINE_STEEL,\n\tFINESTEEL_,\n\tFINE__STEEL,"), "two constants map to the Go name MaterialFineSteel"},
		{"no enum", func(_ *testing.T, m fstest.MapFS) { m[actionTypeFile] = file("class ActionType {}") }, "enum ActionType not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := sampleFS(t)
			tt.change(t, m)
			_, err := generate(m, true, nil, io.Discard)
			if err == nil {
				t.Fatal("generate -enums succeeded")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}
