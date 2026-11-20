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
	got := PatchValues{c.ClanID, c.Title, c.PowerGrade, c.PledgeType, c.Apprentice, c.Sponsor,
		c.LvlJoinedAcademy, c.WantsPeace, c.ClanJoinExpiry, c.ClanCreateExpiry}
	if got != full {
		t.Errorf("Apply of every field = %+v, want %+v", got, full)
	}

	c = Character{Name: "Kept", Title: "old", ClanID: 9, Level: 40}
	CharacterPatch{Fields: PatchPowerGrade | PatchSponsor, Values: full}.Apply(&c)
	want := Character{Name: "Kept", Title: "old", ClanID: 9, Level: 40, PowerGrade: 2, Sponsor: 4}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("Apply of two fields = %+v, want %+v", c, want)
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
