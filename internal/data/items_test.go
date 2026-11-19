package data

import (
	"slices"
	"testing"
)

// Spot checks against dist/game/data/stats/items.
func TestRepositoryItems(t *testing.T) {
	d := loadRepository(t)
	if len(d.Items) != 9208 {
		t.Errorf("%d items, want 9208 (9215 with the 7 custom items left out)", len(d.Items))
	}

	// The API the game server used before data v2.
	item, ok := d.Item(2369)
	if !ok {
		t.Fatal("Item(2369) not found")
	}
	if item.Name != "Squire's Sword" || item.Type != ItemTypeWeapon || item.BodyPart != "rhand" {
		t.Errorf("Item(2369) = %q %q %q", item.Name, item.Type, item.BodyPart)
	}

	// 02300-02399.xml: Squire's Sword.
	sword, _ := d.ItemTemplate(2369)
	if sword.Weapon == nil || sword.Weapon.Type != WeaponSword || sword.Weapon.SoulShots != 1 || sword.Weapon.SpiritShots != 1 ||
		sword.Weapon.AttackRadius != 40 || sword.Weapon.AttackAngle != 120 || !sword.Weapon.AttackWeapon {
		t.Errorf("Squire's Sword weapon = %+v", sword.Weapon)
	}
	if sword.Type1 != 0 || sword.Type2 != 0 || sword.BodyPartMask != 0x80 || sword.Weight != 1600 || sword.Price != 138 ||
		sword.Grade != CrystalNone || sword.Material != MaterialSteel || sword.DefaultAction != ActionEquip {
		t.Errorf("Squire's Sword = %+v", sword)
	}
	if sword.Tradable || sword.Droppable || sword.Sellable || !sword.Destroyable || !sword.Depositable || sword.Stackable {
		t.Errorf("Squire's Sword flags: tradable %v droppable %v sellable %v destroyable %v depositable %v",
			sword.Tradable, sword.Droppable, sword.Sellable, sword.Destroyable, sword.Depositable)
	}
	wantStats := []ItemStat{
		{Stat: StatPowerAttack, Op: OpSet, Value: 6},
		{Stat: StatMagicAttack, Op: OpSet, Value: 5},
		{Stat: StatCriticalRate, Op: OpSet, Value: 8},
		{Stat: StatPowerAttackSpeed, Op: OpSet, Value: 379},
		{Stat: StatRandomDamage, Op: OpAdd, Value: 10},
		{Stat: StatPowerAttackRange, Op: OpSet, Value: 40},
		{Stat: StatPowerAttack, Op: OpEnchant},
		{Stat: StatMagicAttack, Op: OpEnchant},
	}
	if !slices.Equal(sword.Stats, wantStats) {
		t.Errorf("Squire's Sword stats = %+v, want %+v", sword.Stats, wantStats)
	}
	if v, ok := sword.StatValue(StatRandomDamage); !ok || v != 10 {
		t.Errorf("random damage = %v, %v; want 10", v, ok)
	}
	if v, ok := sword.StatValue(StatPowerAttackRange); !ok || v != 40 {
		t.Errorf("attack range = %v, %v; want 40", v, ok)
	}

	// 01100-01199.xml: Squire's Shirt, light armor with 33 P.Def.
	shirt, _ := d.ItemTemplate(1146)
	if shirt.Armor == nil || shirt.Armor.Type != ArmorLight || shirt.BodyPart != "chest" || shirt.BodyPartMask != 0x400 ||
		shirt.Type1 != 1 || shirt.Type2 != 1 {
		t.Errorf("Squire's Shirt = %+v, armor %+v", shirt, shirt.Armor)
	}
	if want := []ItemStat{{Stat: StatPowerDefence, Op: OpAdd, Value: 33}, {Stat: StatPowerDefence, Op: OpEnchant}}; !slices.Equal(shirt.Stats, want) {
		t.Errorf("Squire's Shirt stats = %+v, want %+v", shirt.Stats, want)
	}

	// 00100-00199.xml: Apprentice's Earring, a jewel with 11 M.Def.
	earring, _ := d.ItemTemplate(112)
	if earring.BodyPart != "rear;lear" || earring.BodyPartMask != 0x6 || earring.Type1 != 0 || earring.Type2 != 2 ||
		earring.Armor == nil || earring.Armor.Type != ArmorNone {
		t.Errorf("Apprentice's Earring = %+v", earring)
	}
	if want := []ItemStat{{Stat: StatMagicDefence, Op: OpAdd, Value: 11}, {Stat: StatMagicDefence, Op: OpEnchant}}; !slices.Equal(earring.Stats, want) {
		t.Errorf("Apprentice's Earring stats = %+v, want %+v", earring.Stats, want)
	}

	// Body part masks as ItemList sends them.
	for id, want := range map[int]int{
		2368: 0x4000, // Training Gloves: lrhand
		20:   0x0100, // Buckler: lhand
		1147: 0x0800, // Squire's Pants: legs
		5588: 0,      // Tutorial Guide: none
	} {
		if it, ok := d.ItemTemplate(id); !ok || it.BodyPartMask != want || it.Part().Mask() != want {
			t.Errorf("item %d: body part mask %#x, want %#x", id, it.BodyPartMask, want)
		}
	}

	// The Mobius rules datagen applies.
	if buckler, _ := d.ItemTemplate(20); buckler.Armor.Type != ArmorShield || buckler.Stats[len(buckler.Stats)-1] != (ItemStat{Stat: StatShieldDefence, Op: OpEnchant}) {
		t.Errorf("Buckler: armor %+v, stats %+v; want a shield enchanting sDef", buckler.Armor, buckler.Stats)
	}
	if adena, _ := d.ItemTemplate(57); adena.Type1 != 4 || adena.Type2 != 4 || !adena.Stackable {
		t.Errorf("Adena = %+v", adena)
	}
	if shot, _ := d.ItemTemplate(1835); shot.Etc == nil || shot.Etc.Type != EtcShot || shot.Etc.Handler != "SoulShots" {
		t.Errorf("Soulshot: No Grade = %+v", shot.Etc)
	}
	if blessed, _ := d.ItemTemplate(3948); !blessed.Etc.Blessed {
		t.Error("Blessed Spiritshot: D-Grade is not blessed")
	}
	hero, _ := d.ItemTemplate(6611)
	if !hero.Hero || hero.Conditions == nil || !slices.ContainsFunc(hero.Skills, func(s ItemSkill) bool { return s.Type == ItemSkillOnCriticalSkill }) {
		t.Errorf("Infinity Blade: hero %v, conditions %+v, skills %+v", hero.Hero, hero.Conditions, hero.Skills)
	}
	for _, it := range d.Items {
		if it.Weapon != nil && it.Weapon.Type == WeaponBow && it.ReuseDelay <= 0 {
			t.Errorf("bow %d (%s) has no reuse delay; Mobius gives bows 1500", it.ID, it.Name)
		}
		if it.DisplayID == 0 {
			t.Errorf("item %d has no display ID", it.ID)
		}
	}
	// Custom items are left out.
	for _, id := range []int{13531, 13534, 50000} {
		if _, ok := d.ItemTemplate(id); ok {
			t.Errorf("custom item %d is loaded", id)
		}
	}
	if potion, _ := d.ItemTemplate(733); potion.Name != "Endeavor Potion" {
		t.Errorf("item 733 = %q", potion.Name)
	}
}

func TestItemLookups(t *testing.T) {
	d, err := LoadFS(validTables().fsys(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{-1, 0, 11, 99999999} {
		if _, ok := d.ItemTemplate(id); ok {
			t.Errorf("ItemTemplate(%d) found", id)
		}
		if _, ok := d.Item(id); ok {
			t.Errorf("Item(%d) found", id)
		}
	}
	p1, _ := d.ItemTemplate(2369)
	p2, _ := d.ItemTemplate(2369)
	if p1 != p2 || p1.DisplayID != 2369 {
		t.Errorf("ItemTemplate(2369) = %p, %p, display ID %d", p1, p2, p1.DisplayID)
	}
	c, _ := d.Item(2369)
	c.Name = "changed"
	if p1.Name != "Squire's Sword" {
		t.Error("changing the copy from Item changed the data")
	}
}

func TestItemMethods(t *testing.T) {
	tb := validTables()
	dagger, tunic, guide := &tb.items[0], &tb.items[1], &tb.items[3]
	if dagger.Part() != BodyPartRHand || guide.Part() != BodyPartNone || (&Item{BodyPart: "tail"}).Part() != BodyPartNone {
		t.Errorf("Part: %v, %v", dagger.Part(), guide.Part())
	}
	arrow := &Item{Type: ItemTypeEtc, BodyPart: "lhand"}
	if !dagger.Equipable() || !tunic.Equipable() || guide.Equipable() || arrow.Equipable() {
		t.Error("Equipable: want the weapon and the armor only")
	}
	if dagger.Mask() != 1<<WeaponDagger || tunic.Mask() != 1<<(int(ArmorMagic)+NumWeaponTypes) || guide.Mask() != 0 {
		t.Errorf("Mask: %#x, %#x, %#x", dagger.Mask(), tunic.Mask(), guide.Mask())
	}
	if v, ok := tunic.StatValue(StatPowerDefence); !ok || v != 17 {
		t.Errorf("StatValue(pDef) = %v, %v", v, ok)
	}
	if _, ok := tunic.StatValue(StatPowerAttack); ok {
		t.Error("StatValue(pAtk) found on the tunic")
	}
	enchantOnly := &Item{Stats: []ItemStat{{Stat: StatPowerAttack, Op: OpEnchant}}}
	if _, ok := enchantOnly.StatValue(StatPowerAttack); ok {
		t.Error("StatValue found an enchant function")
	}
	for _, tt := range []struct {
		grade CrystalType
		count int
		want  bool
	}{{CrystalD, 5, true}, {CrystalNone, 5, false}, {CrystalS, 0, false}} {
		if got := (&Item{Grade: tt.grade, CrystalCount: tt.count}).Crystallizable(); got != tt.want {
			t.Errorf("Crystallizable(%s, %d) = %v", tt.grade, tt.count, got)
		}
	}
	if !WeaponBow.Ranged() || WeaponSword.Ranged() || WeaponSword.Mask() != 1 || ArmorShield.Mask() != 1<<(5+NumWeaponTypes) {
		t.Error("weapon and armor type helpers")
	}
}

func TestStatOp(t *testing.T) {
	for op, want := range map[StatOp]int{
		OpAdd: 30, OpDiv: 20, OpEnchant: 0, OpEnchantHP: 40, OpMul: 20, OpSet: 0, OpShare: 30, OpSub: 30,
	} {
		if got := op.DefaultOrder(); got != want {
			t.Errorf("%s.DefaultOrder() = %d, want %d", op, got, want)
		}
		text, err := op.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var back StatOp
		if err := back.UnmarshalText(text); err != nil || back != op {
			t.Errorf("%s: round trip gave %v, %v", op, back, err)
		}
		if p, ok := ParseStatOp(op.String()); !ok || p != op {
			t.Errorf("ParseStatOp(%q) = %v, %v", op, p, ok)
		}
	}
	bad := StatOp(99)
	if bad.Valid() || bad.DefaultOrder() != 0 || bad.String() != "stat function(99)" {
		t.Errorf("StatOp(99): valid %v, order %d, name %q", bad.Valid(), bad.DefaultOrder(), bad.String())
	}
	if _, err := bad.MarshalText(); err == nil {
		t.Error("MarshalText of an invalid StatOp succeeded")
	}
	var op StatOp
	if err := op.UnmarshalText([]byte("pow")); err == nil {
		t.Error(`UnmarshalText("pow") succeeded`)
	}
}
