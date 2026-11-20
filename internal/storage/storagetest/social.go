package storagetest

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// SocialStores are the stores of one database that a storage.Batch writes
// across (Plan 12, decisions T1–T6).
type SocialStores struct {
	Characters SaverStore
	Batches    storage.BatchWriter
	Clans      storage.ClanStore
	Crests     storage.CrestStore
	Relations  storage.RelationStore
}

// Clan IDs of the social tests. They come from the object ID space, apart
// from the sample characters and their items.
const (
	clanDragons int32 = 5000
	clanWolves  int32 = 4000
	clanRavens  int32 = 6000
)

// sampleClan returns a clan with every field set to a distinct value.
func sampleClan(id int32, name string) storage.Clan {
	c := storage.Clan{
		ID:                  id,
		Name:                name,
		Level:               5,
		Reputation:          1234,
		LeaderID:            id + 1,
		NewLeaderID:         id + 2,
		CrestID:             3,
		CrestLargeID:        4,
		AllyID:              id,
		AllyName:            "Alliance of " + name,
		AllyCrestID:         5,
		AllyPenalty:         storage.AllyPenalty{Expiry: 1_795_000_000_333, Type: storage.AllyPenaltyDismissClan},
		MemberPenaltyExpiry: 1_795_000_000_444,
		DissolvingExpiry:    1_795_000_000_555,
		CastleID:            6,
		HideoutID:           7,
		Units: []storage.ClanUnit{
			{Type: storage.UnitAcademy, Name: name + " Academy"},
			{Type: storage.UnitRoyalGuard1, Name: name + " Guard", LeaderID: id + 3},
		},
		Skills: []storage.ClanSkill{
			{SkillID: 391, Level: 2, Unit: storage.UnitRoyalGuard1},
			{SkillID: 370, Level: 1, Unit: storage.ClanSkillAllUnit},
		},
		Notice:       storage.ClanNotice{Text: "Welcome to " + name, Enabled: true},
		WarsDeclared: []int32{id + 200, id + 100},
	}
	for rank := int32(1); rank <= 9; rank++ {
		c.Ranks = append(c.Ranks, storage.ClanRank{Rank: rank, Privs: rank * 37})
	}
	return c
}

// bareClan returns a level 0 clan with no lists, as a new clan is.
func bareClan(id int32, name string, leader int32) storage.Clan {
	return storage.Clan{ID: id, Name: name, LeaderID: leader}
}

// mustWrite writes b, failing the test on an error.
func mustWrite(t *testing.T, s storage.BatchWriter, b storage.Batch) {
	t.Helper()
	if err := s.WriteBatch(t.Context(), b); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
}

// checkClan fails the test if the stored clan differs from want.
func checkClan(t *testing.T, s storage.ClanStore, want storage.Clan) {
	t.Helper()
	got, err := s.GetClan(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("GetClan(%d): %v", want.ID, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("stored clan %s = %+v, want %+v", want.Name, got, want)
	}
}

// checkNoClan fails the test unless clan id is missing.
func checkNoClan(t *testing.T, s storage.ClanStore, id int32) {
	t.Helper()
	if c, err := s.GetClan(t.Context(), id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("GetClan(%d) = %+v, %v; want ErrNotFound", id, c, err)
	}
}

// checkRelations fails the test unless owner has exactly want, in order.
func checkRelations(t *testing.T, s storage.RelationStore, owner int32, want ...storage.Relation) {
	t.Helper()
	got, err := s.Relations(t.Context(), owner)
	if err != nil {
		t.Fatalf("Relations(%d): %v", owner, err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Relations(%d) = %+v, want %+v", owner, got, want)
	}
}

// checkRelationsTo fails the test unless the relations to target are
// exactly want, in order.
func checkRelationsTo(t *testing.T, s storage.RelationStore, target int32, want ...storage.Relation) {
	t.Helper()
	got, err := s.RelationsTo(t.Context(), target)
	if err != nil {
		t.Fatalf("RelationsTo(%d): %v", target, err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("RelationsTo(%d) = %+v, want %+v", target, got, want)
	}
}

// checkClanItems fails the test unless the clan owns exactly want, in their
// stored form.
func checkClanItems(t *testing.T, s storage.ClanStore, clanID int32, want []storage.Item) {
	t.Helper()
	got, err := s.ClanItems(t.Context(), clanID)
	if err != nil {
		t.Fatalf("ClanItems(%d): %v", clanID, err)
	}
	if want := StoredItems(want); !slices.Equal(got, want) {
		t.Errorf("ClanItems(%d) = %+v, want %+v", clanID, got, want)
	}
}

// memberOf returns what ClanMembers reads of c.
func memberOf(c storage.Character) storage.ClanMember {
	return storage.ClanMember{
		ObjectID:         c.ObjectID,
		ClanID:           c.ClanID,
		Name:             c.Name,
		Level:            c.Level,
		ClassID:          c.ClassID,
		Race:             c.Race,
		Sex:              c.Sex,
		Title:            c.Title,
		PowerGrade:       c.PowerGrade,
		PledgeType:       c.PledgeType,
		Apprentice:       c.Apprentice,
		Sponsor:          c.Sponsor,
		LvlJoinedAcademy: c.LvlJoinedAcademy,
		WantsPeace:       c.WantsPeace,
	}
}

// withItem returns items with it in place of the item with its object ID,
// or added.
func withItem(items []storage.Item, it storage.Item) []storage.Item {
	return append(without(items, it.ObjectID), it)
}

// trade returns the joint save of a trade between the sample characters a
// and b (decision T5): a gives b 1,000 of its adena, which b adds to its own
// stack, and b gives a its shadow weapon. It also returns a and b as they
// are stored after it.
func trade(a, b storage.Character) (batch storage.Batch, wantA, wantB storage.Character) {
	adenaA, adenaB, weapon := a.Items[2], b.Items[2], b.Items[4]
	adenaA.Count -= 1000
	adenaB.Count += 1000
	batch = storage.Batch{Saves: []storage.CharacterSave{
		{Character: a, Changes: storage.ItemChanges{Upserts: []storage.Item{adenaA, weapon}}},
		{Character: b, Changes: storage.ItemChanges{Upserts: []storage.Item{adenaB}, Left: []int32{weapon.ObjectID}}},
	}}
	wantA, wantB = a, b
	wantA.Items = withItem(withItem(a.Items, adenaA), weapon)
	wantB.Items = withItem(without(b.Items, weapon.ObjectID), adenaB)
	return batch, wantA, wantB
}

// TestSocialStores runs the conformance tests of the clan, crest and
// relation stores and of the joint save (storage.BatchWriter). newStores
// must return the stores of an empty database for each call, and a function
// that makes the next write fail with the given error after it did all its
// work, just before it commits, as for TestFailedWrites.
func TestSocialStores(t *testing.T, newStores func(t *testing.T) (s SocialStores, failNext func(error))) {
	t.Run("ClanRoundTrip", func(t *testing.T) {
		s, _ := newStores(t)
		dragons := sampleClan(clanDragons, "Dragons")
		wolves := bareClan(clanWolves, "Wolves", 100)
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{dragons, wolves}})
		checkClan(t, s.Clans, dragons)
		checkClan(t, s.Clans, wolves)
		list, err := s.Clans.LoadClans(t.Context())
		if err != nil {
			t.Fatalf("LoadClans: %v", err)
		}
		if want := []storage.Clan{wolves, dragons}; !reflect.DeepEqual(list, want) {
			t.Errorf("LoadClans = %+v, want %+v in ID order", list, want)
		}
		checkNoClan(t, s.Clans, clanRavens)
	})

	t.Run("NoClans", func(t *testing.T) {
		s, _ := newStores(t)
		if list, err := s.Clans.LoadClans(t.Context()); err != nil || list != nil {
			t.Errorf("LoadClans on an empty store = %+v, %v; want nil", list, err)
		}
		if err := s.Batches.WriteBatch(t.Context(), storage.Batch{}); err != nil {
			t.Errorf("WriteBatch of an empty batch: %v", err)
		}
	})

	t.Run("ClanReplace", func(t *testing.T) {
		s, _ := newStores(t)
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{sampleClan(clanDragons, "Dragons")}})
		renamed := sampleClan(clanDragons, "Drakes")
		renamed.Level, renamed.Reputation, renamed.Ranks = 6, 0, renamed.Ranks[:5]
		renamed.Skills, renamed.WarsDeclared, renamed.Notice = nil, nil, storage.ClanNotice{}
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{renamed}})
		checkClan(t, s.Clans, renamed)
		// The old name is free.
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{bareClan(clanWolves, "dragons", 100)}})
	})

	t.Run("ClanNameTaken", func(t *testing.T) {
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		mustCreate(t, s.Characters, a)
		dragons := sampleClan(clanDragons, "Dragons")
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{dragons}})

		// Another clan with the name, ignoring case: nothing of the batch
		// is written.
		err := s.Batches.WriteBatch(t.Context(), storage.Batch{
			Saves: []storage.CharacterSave{{Character: changedCharacter(a)}},
			Clans: []storage.Clan{bareClan(clanWolves, "DRAGONS", 100)},
		})
		if !errors.Is(err, storage.ErrExists) {
			t.Fatalf("WriteBatch with a taken clan name: error = %v, want ErrExists", err)
		}
		checkStored(t, s.Characters, a)
		checkNoClan(t, s.Clans, clanWolves)
		checkClan(t, s.Clans, dragons)

		// Two new clans with one name.
		err = s.Batches.WriteBatch(t.Context(), storage.Batch{Clans: []storage.Clan{
			bareClan(clanWolves, "Ravens", 100), bareClan(clanRavens, "ravens", 200),
		}})
		if !errors.Is(err, storage.ErrExists) {
			t.Fatalf("WriteBatch with two clans of one name: error = %v, want ErrExists", err)
		}
		checkNoClan(t, s.Clans, clanWolves)
		checkNoClan(t, s.Clans, clanRavens)

		// A clan renamed first frees its name for a later one.
		renamed := dragons
		renamed.Name = "Drakes"
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{renamed, bareClan(clanWolves, "Dragons", 100)}})
		checkClan(t, s.Clans, renamed)
		checkClan(t, s.Clans, bareClan(clanWolves, "Dragons", 100))
	})

	t.Run("DeleteClan", func(t *testing.T) {
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		mustCreate(t, s.Characters, a)
		dragons := sampleClan(clanDragons, "Dragons")
		wolves := sampleClan(clanWolves, "Wolves")
		wolves.WarsDeclared = []int32{clanRavens, clanDragons}
		ravens := bareClan(clanRavens, "Ravens", 300)
		ravens.WarsDeclared = []int32{clanDragons}
		stash := []storage.Item{
			{ObjectID: 5001, ItemID: itemAdena, Count: 100, Loc: storage.LocClanWarehouse, Mana: storage.NoMana},
			{ObjectID: 5002, ItemID: itemBoneArrow, Count: 7, Loc: storage.LocClanWarehouse, Mana: storage.NoMana},
		}
		mustWrite(t, s.Batches, storage.Batch{
			Clans: []storage.Clan{dragons, wolves, ravens},
			Items: []storage.OwnerItemChanges{{Owner: clanDragons, Upserts: stash}},
		})
		checkClanItems(t, s.Clans, clanDragons, stash)

		mustWrite(t, s.Batches, storage.Batch{DeletedClans: []int32{clanDragons, 999}})
		checkNoClan(t, s.Clans, clanDragons)
		checkClanItems(t, s.Clans, clanDragons, nil)
		wolves.WarsDeclared = []int32{clanRavens}
		checkClan(t, s.Clans, wolves)
		ravens.WarsDeclared = nil
		checkClan(t, s.Clans, ravens)
		checkStored(t, s.Characters, a)
		// The name and the item IDs are free.
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{bareClan(clanDragons, "DRAGONS", 100)}})
		checkItemIDsFree(t, s.Characters, 900, 5001, 5002)
	})

	t.Run("AllyCrest", func(t *testing.T) {
		s, _ := newStores(t)
		dragons := sampleClan(clanDragons, "Dragons") // the alliance leader
		wolves := sampleClan(clanWolves, "Wolves")
		wolves.AllyID, wolves.AllyName = clanDragons, dragons.AllyName
		ravens := bareClan(clanRavens, "Ravens", 300) // in no alliance
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{dragons, wolves, ravens}})
		mustWrite(t, s.Batches, storage.Batch{AllyCrests: []storage.AllyCrest{{AllyID: clanDragons, CrestID: 77}, {AllyID: 0, CrestID: 78}}})
		dragons.AllyCrestID, wolves.AllyCrestID = 77, 77
		checkClan(t, s.Clans, dragons)
		checkClan(t, s.Clans, wolves)
		checkClan(t, s.Clans, ravens)
	})

	t.Run("ClanMembers", func(t *testing.T) {
		s, _ := newStores(t)
		var chars []storage.Character
		for i, clan := range []int32{clanDragons, clanWolves, clanDragons, 0, clanWolves} {
			c := sampleCharacter(int32(100*(5-i)), "alice", "Member"+string(rune('A'+i)), baseTime) //nolint:gosec // small test loop index
			c.ClanID = clan
			chars = append(chars, c)
		}
		chars[2].Title, chars[2].WantsPeace, chars[2].PledgeType = "", false, storage.UnitMain
		mustCreate(t, s.Characters, chars...)
		// Object IDs: A 500, B 400, C 300, D 200 (no clan), E 100.
		for clanID, want := range map[int32][]storage.Character{
			clanDragons: {chars[2], chars[0]},
			clanWolves:  {chars[4], chars[1]},
			0:           {chars[4], chars[1], chars[2], chars[0]},
			clanRavens:  nil,
		} {
			got, err := s.Clans.ClanMembers(t.Context(), clanID)
			if err != nil {
				t.Fatalf("ClanMembers(%d): %v", clanID, err)
			}
			var members []storage.ClanMember
			for _, c := range want {
				members = append(members, memberOf(c))
			}
			if !slices.Equal(got, members) {
				t.Errorf("ClanMembers(%d) = %+v, want %+v", clanID, got, members)
			}
		}

		// A dismissal patch takes a member out.
		mustWrite(t, s.Batches, storage.Batch{Patches: []storage.CharacterPatch{
			{ObjectID: 500, Fields: storage.PatchClanMembership, Values: storage.PatchValues{ClanJoinExpiry: 1_795_086_400_000}},
		}})
		got, err := s.Clans.ClanMembers(t.Context(), clanDragons)
		if err != nil {
			t.Fatalf("ClanMembers: %v", err)
		}
		if want := []storage.ClanMember{memberOf(chars[2])}; !slices.Equal(got, want) {
			t.Errorf("ClanMembers after a dismissal = %+v, want %+v", got, want)
		}
	})

	t.Run("CreateClanWithLeader", func(t *testing.T) {
		// Decision T6: a new clan and its leader are saved jointly.
		s, _ := newStores(t)
		leader := sampleCharacter(100, "alice", "Fili", baseTime)
		leader.ClanID, leader.PowerGrade, leader.PledgeType = 0, 0, storage.UnitMain
		mustCreate(t, s.Characters, leader)
		clan := bareClan(clanDragons, "Dragons", leader.ObjectID)
		leader.ClanID, leader.PowerGrade = clanDragons, 6
		mustWrite(t, s.Batches, storage.Batch{
			Saves: []storage.CharacterSave{{Character: leader}},
			Clans: []storage.Clan{clan},
		})
		checkClan(t, s.Clans, clan)
		checkStored(t, s.Characters, leader)
		got, err := s.Clans.ClanMembers(t.Context(), clanDragons)
		if err != nil || !slices.Equal(got, []storage.ClanMember{memberOf(leader)}) {
			t.Errorf("ClanMembers = %+v, %v; want the leader", got, err)
		}
	})

	t.Run("ClanWarehouse", func(t *testing.T) {
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		mustCreate(t, s.Characters, a)
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{bareClan(clanDragons, "Dragons", 100)}})

		// a deposits 500 adena, a new stack in the warehouse, and its whole
		// shadow weapon.
		adena, weapon := a.Items[2], a.Items[4]
		adena.Count -= 500
		stack := storage.Item{ObjectID: 150, ItemID: itemAdena, Count: 500, Loc: storage.LocClanWarehouse, Mana: storage.NoMana}
		stored := weapon
		stored.Loc = storage.LocClanWarehouse
		mustWrite(t, s.Batches, storage.Batch{
			Saves: []storage.CharacterSave{{Character: a, Changes: storage.ItemChanges{Upserts: []storage.Item{adena}, Left: []int32{weapon.ObjectID}}}},
			Items: []storage.OwnerItemChanges{{Owner: clanDragons, Upserts: []storage.Item{stack, stored}}},
		})
		wantA := a
		wantA.Items = withItem(without(a.Items, weapon.ObjectID), adena)
		checkStored(t, s.Characters, wantA)
		checkClanItems(t, s.Clans, clanDragons, []storage.Item{stack, stored})

		// The leader takes the weapon back.
		mustWrite(t, s.Batches, storage.Batch{
			Saves: []storage.CharacterSave{{Character: a, Changes: storage.ItemChanges{Upserts: []storage.Item{weapon}}}},
			Items: []storage.OwnerItemChanges{{Owner: clanDragons, Left: []int32{weapon.ObjectID}}},
		})
		wantA.Items = withItem(wantA.Items, weapon)
		checkStored(t, s.Characters, wantA)
		checkClanItems(t, s.Clans, clanDragons, []storage.Item{stack})
	})

	t.Run("JointTrade", func(t *testing.T) {
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		b := sampleCharacter(200, "bob", "Kili", baseTime)
		mustCreate(t, s.Characters, a, b)
		batch, wantA, wantB := trade(a, b)
		mustWrite(t, s.Batches, batch)
		checkStored(t, s.Characters, wantA)
		checkStored(t, s.Characters, wantB)
	})

	t.Run("JointFailure", func(t *testing.T) {
		// An injected failure inside a joint save leaves both inventories,
		// and everything else of the batch, unchanged.
		s, failNext := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		b := sampleCharacter(200, "bob", "Kili", baseTime)
		c := sampleCharacter(300, "carol", "Oin", baseTime)
		mustCreate(t, s.Characters, a, b, c)
		batch, wantA, wantB := trade(a, b)
		batch.Clans = []storage.Clan{sampleClan(clanDragons, "Dragons")}
		batch.Items = []storage.OwnerItemChanges{{Owner: clanDragons, Upserts: []storage.Item{
			{ObjectID: 5001, ItemID: itemAdena, Count: 1, Loc: storage.LocClanWarehouse, Mana: storage.NoMana},
		}}}
		batch.Crests = []storage.Crest{{ID: 1, Type: storage.CrestPledge, Data: []byte{1, 2, 3}}}
		batch.Relations.Added = storage.FriendRelations(100, 200)
		batch.Patches = []storage.CharacterPatch{{ObjectID: 300, Fields: storage.PatchClanID | storage.PatchTitle,
			Values: storage.PatchValues{ClanID: clanDragons, Title: "Patched"}}}

		failNext(errInjected)
		if err := s.Batches.WriteBatch(t.Context(), batch); !errors.Is(err, errInjected) {
			t.Fatalf("WriteBatch error = %v, want the injected failure", err)
		}
		checkStored(t, s.Characters, a)
		checkStored(t, s.Characters, b)
		checkStored(t, s.Characters, c)
		checkNoClan(t, s.Clans, clanDragons)
		checkClanItems(t, s.Clans, clanDragons, nil)
		checkRelations(t, s.Relations, 100)
		if crests, err := s.Crests.LoadCrests(t.Context()); err != nil || crests != nil {
			t.Errorf("LoadCrests after a failed batch = %+v, %v; want nil", crests, err)
		}

		// The retry writes everything.
		mustWrite(t, s.Batches, batch)
		checkStored(t, s.Characters, wantA)
		checkStored(t, s.Characters, wantB)
		wantC := c
		batch.Patches[0].Apply(&wantC)
		checkStored(t, s.Characters, wantC)
		checkClan(t, s.Clans, batch.Clans[0])
		checkClanItems(t, s.Clans, clanDragons, batch.Items[0].Upserts)
		checkRelations(t, s.Relations, 100, storage.Relation{Owner: 100, Target: 200, Type: storage.RelationFriend})
	})

	t.Run("JointNotFound", func(t *testing.T) {
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		mustCreate(t, s.Characters, a)
		ghost := sampleCharacter(200, "bob", "Ghost", baseTime)
		batch, _, _ := trade(a, ghost)
		batch.Clans = []storage.Clan{bareClan(clanDragons, "Dragons", 100)}
		if err := s.Batches.WriteBatch(t.Context(), batch); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("WriteBatch with a missing character: error = %v, want ErrNotFound", err)
		}
		checkStored(t, s.Characters, a)
		checkNoClan(t, s.Clans, clanDragons)
	})

	t.Run("Patches", func(t *testing.T) {
		// Decision T6: a patch comes after the save of its character.
		s, _ := newStores(t)
		a := sampleCharacter(100, "alice", "Fili", baseTime)
		b := sampleCharacter(200, "bob", "Kili", baseTime)
		mustCreate(t, s.Characters, a, b)
		saved := changedCharacter(a)
		saved.Items, saved.Quests = a.Items, a.Quests // a save does not use them
		saved.ClanID, saved.Title, saved.PowerGrade = clanDragons, "Saved", 5
		dismiss := storage.CharacterPatch{ObjectID: 100, Fields: storage.PatchClanMembership,
			Values: storage.PatchValues{ClanJoinExpiry: 1_795_086_400_000}}
		promote := storage.CharacterPatch{ObjectID: 200, Fields: storage.PatchPowerGrade | storage.PatchPledgeType | storage.PatchWantsPeace,
			Values: storage.PatchValues{PowerGrade: 2, PledgeType: storage.UnitKnight3, Title: "not written"}}
		mustWrite(t, s.Batches, storage.Batch{
			Saves: []storage.CharacterSave{{Character: saved}},
			Patches: []storage.CharacterPatch{
				dismiss,
				{ObjectID: 999, Fields: storage.PatchClanMembership}, // no such character
				promote,
				{ObjectID: 200, Values: storage.PatchValues{ClanID: 1}}, // no fields
			},
		})
		wantA, wantB := saved, b
		dismiss.Apply(&wantA)
		promote.Apply(&wantB)
		if wantA.ClanID != 0 || wantA.Title != "" || wantA.ClanJoinExpiry != 1_795_086_400_000 || wantB.Title != b.Title || wantB.WantsPeace {
			t.Fatalf("CharacterPatch.Apply: %+v, %+v", wantA, wantB)
		}
		checkStored(t, s.Characters, wantA)
		checkStored(t, s.Characters, wantB)
		if _, err := s.Characters.Get(t.Context(), 999); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(999) after a patch = %v, want ErrNotFound", err)
		}

		// Patches set and clear every field.
		set := storage.CharacterPatch{ObjectID: 200, Fields: storage.PatchClanMembership, Values: storage.PatchValues{
			ClanID: clanWolves, Title: "Knight", PowerGrade: 3, PledgeType: storage.UnitRoyalGuard2, Apprentice: 7, Sponsor: 8,
			LvlJoinedAcademy: 19, WantsPeace: true, ClanJoinExpiry: 9, ClanCreateExpiry: 10,
		}}
		mustWrite(t, s.Batches, storage.Batch{Patches: []storage.CharacterPatch{set}})
		set.Apply(&wantB)
		checkStored(t, s.Characters, wantB)
		cleared := storage.CharacterPatch{ObjectID: 200, Fields: storage.PatchClanMembership}
		mustWrite(t, s.Batches, storage.Batch{Patches: []storage.CharacterPatch{cleared}})
		cleared.Apply(&wantB)
		checkStored(t, s.Characters, wantB)
	})

	t.Run("Relations", func(t *testing.T) {
		s, _ := newStores(t)
		friend := func(o, tg int32) storage.Relation {
			return storage.Relation{Owner: o, Target: tg, Type: storage.RelationFriend}
		}
		block := func(o, tg int32) storage.Relation {
			return storage.Relation{Owner: o, Target: tg, Type: storage.RelationBlock}
		}
		mustWrite(t, s.Batches, storage.Batch{Relations: storage.RelationChanges{
			Added: append(storage.FriendRelations(100, 300), block(100, 200), friend(100, 200), block(400, 300), friend(100, 300)),
		}})
		checkRelations(t, s.Relations, 100, friend(100, 200), friend(100, 300), block(100, 200))
		checkRelations(t, s.Relations, 300, friend(300, 100))
		checkRelations(t, s.Relations, 200)
		checkRelationsTo(t, s.Relations, 300, friend(100, 300), block(400, 300))
		checkRelationsTo(t, s.Relations, 200, friend(100, 200), block(100, 200))

		// Removing a friend removes both directions; adding again what is
		// stored and removing what is not changes nothing.
		mustWrite(t, s.Batches, storage.Batch{Relations: storage.RelationChanges{
			Removed: append(storage.FriendRelations(100, 300), block(500, 100)),
			Added:   []storage.Relation{block(100, 200)},
		}})
		checkRelations(t, s.Relations, 100, friend(100, 200), block(100, 200))
		checkRelations(t, s.Relations, 300)
		checkRelationsTo(t, s.Relations, 300, block(400, 300))

		// Removed and added in one batch: the add comes last.
		mustWrite(t, s.Batches, storage.Batch{Relations: storage.RelationChanges{
			Removed: []storage.Relation{friend(100, 200)},
			Added:   []storage.Relation{friend(100, 200)},
		}})
		checkRelations(t, s.Relations, 100, friend(100, 200), block(100, 200))
	})

	t.Run("DeleteRemovesRelations", func(t *testing.T) {
		s, _ := newStores(t)
		mustCreate(t, s.Characters,
			sampleCharacter(100, "alice", "Fili", baseTime),
			sampleCharacter(200, "bob", "Kili", baseTime),
			sampleCharacter(300, "carol", "Oin", baseTime))
		block := func(o, tg int32) storage.Relation {
			return storage.Relation{Owner: o, Target: tg, Type: storage.RelationBlock}
		}
		mustWrite(t, s.Batches, storage.Batch{Relations: storage.RelationChanges{
			Added: append(storage.FriendRelations(100, 200), block(300, 100), block(200, 300), block(100, 300)),
		}})
		if err := s.Characters.Delete(t.Context(), 100); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		checkRelations(t, s.Relations, 100)
		checkRelationsTo(t, s.Relations, 100)
		checkRelations(t, s.Relations, 200, block(200, 300))
		checkRelations(t, s.Relations, 300)
		checkRelationsTo(t, s.Relations, 300, block(200, 300))
	})

	t.Run("Crests", func(t *testing.T) {
		s, _ := newStores(t)
		for want := int32(1); want <= 3; want++ {
			id, err := s.Crests.NextCrestID(t.Context())
			if err != nil {
				t.Fatalf("NextCrestID: %v", err)
			}
			if id != want {
				t.Errorf("NextCrestID = %d, want %d", id, want)
			}
		}
		large := bytes.Repeat([]byte{0xAB}, 2176)
		pledge := storage.Crest{ID: 1, Type: storage.CrestPledge, Data: []byte{1, 2, 3, 4}}
		mustWrite(t, s.Batches, storage.Batch{Crests: []storage.Crest{
			{ID: 2, Type: storage.CrestPledgeLarge, Data: large}, pledge,
		}})
		pledge.Data[0] = 99 // must not reach the store
		got, err := s.Crests.LoadCrests(t.Context())
		if err != nil {
			t.Fatalf("LoadCrests: %v", err)
		}
		want := []storage.Crest{{ID: 1, Type: storage.CrestPledge, Data: []byte{1, 2, 3, 4}}, {ID: 2, Type: storage.CrestPledgeLarge, Data: large}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LoadCrests = %+v, want %+v", got, want)
		}

		// A new crest replaces the clan's old one (decision T3).
		clan := bareClan(clanDragons, "Dragons", 100)
		clan.CrestID = 3
		ally := storage.Crest{ID: 3, Type: storage.CrestAlly, Data: []byte{5, 6}}
		mustWrite(t, s.Batches, storage.Batch{DeletedCrests: []int32{1}, Crests: []storage.Crest{ally}, Clans: []storage.Clan{clan}})
		got, err = s.Crests.LoadCrests(t.Context())
		if err != nil {
			t.Fatalf("LoadCrests: %v", err)
		}
		want = []storage.Crest{{ID: 2, Type: storage.CrestPledgeLarge, Data: large}, ally}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LoadCrests = %+v, want %+v", got, want)
		}
		got[1].Data[0] = 99
		if again, err := s.Crests.LoadCrests(t.Context()); err != nil || !reflect.DeepEqual(again, want) {
			t.Errorf("stored crests changed through a caller's slice: %+v, %v", again, err)
		}
		checkClan(t, s.Clans, clan)
	})

	t.Run("CrestIDsConcurrent", func(t *testing.T) {
		s, _ := newStores(t)
		const workers, calls = 8, 10
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seen = make(map[int32]bool)
		)
		for range workers {
			wg.Go(func() {
				for range calls {
					id, err := s.Crests.NextCrestID(t.Context())
					if err != nil {
						t.Errorf("NextCrestID: %v", err)
						return
					}
					mu.Lock()
					if seen[id] {
						t.Errorf("crest ID %d handed out twice", id)
					}
					seen[id] = true
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		for id := int32(1); id <= workers*calls; id++ {
			if !seen[id] {
				t.Errorf("crest ID %d was not handed out; want 1 to %d", id, workers*calls)
				break
			}
		}
	})

	t.Run("ClanResultsAreCopies", func(t *testing.T) {
		s, _ := newStores(t)
		clan := sampleClan(clanDragons, "Dragons")
		want := sampleClan(clanDragons, "Dragons")
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{clan}})
		clan.Ranks[0].Privs = 1
		clan.WarsDeclared[0] = 1
		got, err := s.Clans.GetClan(t.Context(), clanDragons)
		if err != nil {
			t.Fatalf("GetClan: %v", err)
		}
		got.Units[0].Name = "changed"
		list, err := s.Clans.LoadClans(t.Context())
		if err != nil {
			t.Fatalf("LoadClans: %v", err)
		}
		list[0].Skills[0].Level = 9
		checkClan(t, s.Clans, want)
	})
}
