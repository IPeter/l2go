package data

import (
	"slices"
	"strings"
	"testing"
)

func TestStatEnum(t *testing.T) {
	if NumStats != 126 {
		t.Errorf("NumStats = %d, want the 126 stats of Stat.java", NumStats)
	}
	if StatPowerAttack.String() != "pAtk" || StatPowerAttack.JavaName() != "POWER_ATTACK" {
		t.Errorf("StatPowerAttack = %q / %q", StatPowerAttack, StatPowerAttack.JavaName())
	}
	if StatSTR.String() != "STR" || StatMaxHP.String() != "maxHp" || StatRandomDamage != NumStats-1 {
		t.Errorf("StatSTR %q, StatMaxHP %q, StatRandomDamage %d", StatSTR, StatMaxHP, StatRandomDamage)
	}
	for s := range Stat(NumStats) {
		text, err := s.MarshalText()
		if err != nil {
			t.Fatalf("%d: %v", s, err)
		}
		var back Stat
		if err := back.UnmarshalText(text); err != nil || back != s {
			t.Errorf("stat %q: round trip gave %v, %v", text, back, err)
		}
		if p, ok := ParseStat(string(text)); !ok || p != s {
			t.Errorf("ParseStat(%q) = %v, %v", text, p, ok)
		}
	}
	bad := Stat(200)
	if bad.Valid() || bad.String() != "stat(200)" || bad.JavaName() != "stat(200)" {
		t.Errorf("Stat(200): valid %v, %q, %q", bad.Valid(), bad, bad.JavaName())
	}
	if _, err := bad.MarshalText(); err == nil {
		t.Error("MarshalText of an invalid stat succeeded")
	}
	var s Stat
	if err := s.UnmarshalText([]byte("POWER_ATTACK")); err == nil || !strings.Contains(err.Error(), `unknown stat "POWER_ATTACK"`) {
		t.Errorf("UnmarshalText of a Java name: %v; the data uses the XML names", err)
	}
}

func TestBodyPartEnum(t *testing.T) {
	for name, want := range map[string]BodyPart{
		"rhand": BodyPartRHand, "lrhand": BodyPartLRHand, "rear;lear": BodyPartLREar,
		"shirt": BodyPartUnderwear, "rbracelet": BodyPartREar, "onepiece": BodyPartFullArmor, "dhair": BodyPartHairall,
	} {
		if got, ok := BodyPartByName(name); !ok || got != want {
			t.Errorf("BodyPartByName(%q) = %v, %v; want %v", name, got, ok, want)
		}
	}
	if _, ok := BodyPartByName("lbracelet"); ok {
		t.Error(`BodyPartByName("lbracelet") found; Interlude has no such body part`)
	}
	if _, ok := ParseBodyPart("onepiece"); ok {
		t.Error(`ParseBodyPart("onepiece") found; only BodyPartByName takes the other names`)
	}
	for part, want := range map[BodyPart]int{
		BodyPartNone: 0, BodyPartUnderwear: 0x1, BodyPartLREar: 0x6, BodyPartLRFinger: 0x30, BodyPartRHand: 0x80,
		BodyPartLHand: 0x100, BodyPartChest: 0x400, BodyPartLRHand: 0x4000, BodyPartFullArmor: 0x8000,
		BodyPartHair: 0x10000, BodyPartHairall: 0x80000, BodyPartWolf: -100, BodyPart(99): 0,
	} {
		if got := part.Mask(); got != want {
			t.Errorf("%v.Mask() = %#x, want %#x", part, got, want)
		}
	}
	for part, want := range map[BodyPart]int{
		BodyPartRHand: 7, BodyPartChest: 10, BodyPartHair2: 16, BodyPartLRHand: -1, BodyPartNone: PaperdollSlots, BodyPart(99): -1,
	} {
		if got := part.PaperdollSlot(); got != want {
			t.Errorf("%v.PaperdollSlot() = %d, want %d", part, got, want)
		}
	}
	if PaperdollSlots != 17 {
		t.Errorf("PaperdollSlots = %d, want 17", PaperdollSlots)
	}
	if !slices.Equal(BodyPartFullArmor.Names(), []string{"fullarmor", "onepiece"}) || BodyPartRHand.JavaName() != "R_HAND" ||
		BodyPart(99).Names() != nil || BodyPart(99).JavaName() != "body part(99)" {
		t.Errorf("names: %v %q", BodyPartFullArmor.Names(), BodyPartRHand.JavaName())
	}
}

func TestItemEnums(t *testing.T) {
	if w, ok := ParseWeaponType("DUALFIST"); !ok || w != WeaponDualfist || w.String() != "DUALFIST" {
		t.Errorf("ParseWeaponType(DUALFIST) = %v, %v", w, ok)
	}
	if _, ok := ParseWeaponType("dualfist"); ok {
		t.Error("ParseWeaponType is not case sensitive")
	}
	if NumWeaponTypes != 14 || NumArmorTypes != 6 || !EtcShot.Valid() || EtcShot.String() != "SHOT" {
		t.Errorf("enum sizes: %d weapon types, %d armor types", NumWeaponTypes, NumArmorTypes)
	}
	if CrystalS.Level() != 5 || CrystalD.CrystalItemID() != 1458 || CrystalA.EnchantBonusWeapon() != 145 ||
		CrystalB.EnchantBonusArmor() != 11 || CrystalNone.Level() != 0 || CrystalType(9).Level() != 0 {
		t.Error("crystal type values differ from CrystalType.java")
	}
	if MaterialSteel != 0 || ActionNone.String() != "NONE" || ItemSkillNormal != 0 || ItemSkillOnMagicSkill.String() != "ON_MAGIC_SKILL" {
		t.Error("item enums differ from the Mobius sources")
	}
}

func TestClassEnums(t *testing.T) {
	if NumPlayerClasses != 89 || len(PlayerClasses()) != 89 || !slices.IsSorted(PlayerClasses()) {
		t.Errorf("%d classes, list of %d, sorted %v", NumPlayerClasses, len(PlayerClasses()), slices.IsSorted(PlayerClasses()))
	}
	if NumRaces != 23 || RaceDarkElf != 2 || RaceDwarf != 4 {
		t.Errorf("races: %d, Dark Elf %d, Dwarf %d", NumRaces, RaceDarkElf, RaceDwarf)
	}
	if r, ok := ParseRace("DARK_ELF"); !ok || r != RaceDarkElf {
		t.Errorf("ParseRace(DARK_ELF) = %v, %v", r, ok)
	}
	tests := []struct {
		class          PlayerClass
		parent         PlayerClass
		hasParent      bool
		level          int
		race           Race
		mage, summoner bool
	}{
		{ClassFighter, 0, false, 0, RaceHuman, false, false},
		{ClassWarrior, ClassFighter, true, 1, RaceHuman, false, false},
		{ClassWarlock, ClassWizard, true, 2, RaceHuman, true, true},
		{ClassDuelist, ClassGladiator, true, 3, RaceHuman, false, false},
		{ClassBladedancer, ClassPalusKnight, true, 2, RaceDarkElf, false, false},
		{ClassElementalMaster, ClassElementalSummoner, true, 3, RaceElf, true, true},
		{ClassOrcMage, 0, false, 0, RaceOrc, true, false},
		{ClassMaestro, ClassWarsmith, true, 3, RaceDwarf, false, false},
	}
	for _, tt := range tests {
		parent, ok := tt.class.Parent()
		if parent != tt.parent || ok != tt.hasParent || tt.class.Level() != tt.level || tt.class.Race() != tt.race ||
			tt.class.IsMage() != tt.mage || tt.class.IsSummoner() != tt.summoner || !tt.class.Valid() {
			t.Errorf("%v: parent %v %v, level %d, race %v, mage %v, summoner %v", tt.class, parent, ok,
				tt.class.Level(), tt.class.Race(), tt.class.IsMage(), tt.class.IsSummoner())
		}
	}
	if ClassFighter != 0 || ClassDuelist != 88 || ClassMaestro != 118 || ClassDwarvenFighter.String() != "DWARVEN_FIGHTER" {
		t.Error("class IDs differ from PlayerClass.java")
	}
	for _, bad := range []PlayerClass{-1, 58, 87, 119} {
		if bad.Valid() || bad.IsMage() || bad.Race() != RaceNone || bad.Level() != 0 {
			t.Errorf("PlayerClass(%d) is valid", bad)
		}
		if _, ok := bad.Parent(); ok {
			t.Errorf("PlayerClass(%d) has a parent", bad)
		}
	}
	if PlayerClass(58).String() != "PlayerClass(58)" {
		t.Errorf("PlayerClass(58) = %q", PlayerClass(58).String())
	}
}
