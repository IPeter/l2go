package storage

import (
	"reflect"
	"testing"
)

func TestCharacterPatchApply(t *testing.T) {
	full := PatchValues{ClanID: 1, Title: "t", PowerGrade: 2, PledgeType: UnitKnight1, Apprentice: 3, Sponsor: 4,
		LvlJoinedAcademy: 5, WantsPeace: true, ClanJoinExpiry: 6, ClanCreateExpiry: 7}
	var c Character
	CharacterPatch{Fields: PatchClanMembership, Values: full}.Apply(&c)
	got := PatchValues{ClanID: c.ClanID, Title: c.Title, PowerGrade: c.PowerGrade, PledgeType: c.PledgeType,
		Apprentice: c.Apprentice, Sponsor: c.Sponsor, LvlJoinedAcademy: c.LvlJoinedAcademy, WantsPeace: c.WantsPeace,
		ClanJoinExpiry: c.ClanJoinExpiry, ClanCreateExpiry: c.ClanCreateExpiry}
	if got != full {
		t.Errorf("Apply of every field = %+v, want %+v", got, full)
	}

	c = Character{Name: "Kept", Title: "old", ClanID: 9, Level: 40}
	CharacterPatch{Fields: PatchPowerGrade | PatchSponsor, Values: full}.Apply(&c)
	want := Character{Name: "Kept", Title: "old", ClanID: 9, Level: 40, PowerGrade: 2, Sponsor: 4}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("Apply of two fields = %+v, want %+v", c, want)
	}

	// The karma and PK count of a cursed weapon's offline end.
	c = Character{Karma: 9999999, PKKills: 14, ClanID: 9}
	CharacterPatch{Fields: PatchKarma | PatchPKKills, Values: PatchValues{Karma: 720, PKKills: 3}}.Apply(&c)
	if want := (Character{Karma: 720, PKKills: 3, ClanID: 9}); !reflect.DeepEqual(c, want) {
		t.Errorf("Apply of karma and PKs = %+v, want %+v", c, want)
	}
}

func TestBatchEmpty(t *testing.T) {
	if b := (Batch{}); !b.Empty() {
		t.Error("Batch{}.Empty() = false")
	}
	for name, b := range map[string]Batch{
		"saves":      {Saves: []CharacterSave{{}}},
		"items":      {Items: []OwnerItemChanges{{}}},
		"clans":      {Clans: []Clan{{}}},
		"deleted":    {DeletedClans: []int32{1}},
		"ally crest": {AllyCrests: []AllyCrest{{}}},
		"crests":     {Crests: []Crest{{}}},
		"dead crest": {DeletedCrests: []int32{1}},
		"added":      {Relations: RelationChanges{Added: FriendRelations(1, 2)}},
		"removed":    {Relations: RelationChanges{Removed: FriendRelations(1, 2)}},
		"patches":    {Patches: []CharacterPatch{{}}},
		"cw deleted": {DeletedCursedWeapons: []int32{8190}},
		"cw rows":    {CursedWeapons: []CursedWeapon{{ItemID: 8190}}},
	} {
		if b.Empty() {
			t.Errorf("%s: Empty() = true", name)
		}
	}
}

func TestFriendRelations(t *testing.T) {
	want := []Relation{{Owner: 1, Target: 2, Type: RelationFriend}, {Owner: 2, Target: 1, Type: RelationFriend}}
	if got := FriendRelations(1, 2); !reflect.DeepEqual(got, want) {
		t.Errorf("FriendRelations(1, 2) = %+v, want %+v", got, want)
	}
}
