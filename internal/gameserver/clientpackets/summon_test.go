package clientpackets

import "testing"

// The opcodes are those of Mobius ClientPackets, the action IDs those of
// the Mobius Interlude RequestActionUse switch.
func TestSummonClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestChangePetName", OpRequestChangePetName, 0x89},
		{"RequestPetUseItem", OpRequestPetUseItem, 0x8a},
		{"RequestGiveItemToPet", OpRequestGiveItemToPet, 0x8b},
		{"RequestGetItemFromPet", OpRequestGetItemFromPet, 0x8c},
		{"RequestPetGetItem", OpRequestPetGetItem, 0x8f},
		{"Change Movement Mode (Pets)", ActionUsePetFollow, 15},
		{"Attack (Pets)", ActionUsePetAttack, 16},
		{"Stop (Pets)", ActionUsePetStop, 17},
		{"Unsummon Pet", ActionUseUnsummonPet, 19},
		{"Change Movement Mode (Servitors)", ActionUseServitorFollow, 21},
		{"Attack (Servitors)", ActionUseServitorAttack, 22},
		{"Stop (Servitors)", ActionUseServitorStop, 23},
		{"Mount/Dismount", ActionUseMountDismount, 38},
		{"Unsummon Servitor", ActionUseUnsummonServitor, 52},
		{"Move to target (Servitors)", ActionUseServitorMoveToTarget, 53},
		{"Move to target (Pets)", ActionUsePetMoveToTarget, 54},
		{"Sin Eater", ActionUseSinEaterBombasticBuster, 1001},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

// Every skill action is one of the switch cases that call useSkill, and
// names either a template parameter or a skill.
func TestSummonActionSkills(t *testing.T) {
	skillCases := []int32{32, 36, 39, 41, 42, 43, 44, 45, 46, 47, 48, 1000,
		1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017,
		1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039, 1040}
	if len(SummonActionSkills) != len(skillCases) {
		t.Errorf("%d skill actions, want %d", len(SummonActionSkills), len(skillCases))
	}
	for _, id := range skillCases {
		s, ok := SummonActionSkills[id]
		if !ok {
			t.Errorf("action %d missing", id)
			continue
		}
		if (s.Param == "") == (s.SkillID == 0) {
			t.Errorf("action %d: %+v, want a parameter or a skill", id, s)
		}
	}
	// Spot checks against the Mobius cases.
	want := map[int32]SummonActionSkill{
		45:   {Param: "HealMagic", OnOwner: true},
		41:   {SkillID: 4230, DoorOnly: true},
		1000: {SkillID: 4079, DoorOnly: true},
		1004: {Param: "Buff", OnOwner: true, Pet: true},
		1039: {SkillID: 5110},
	}
	for id, w := range want {
		if got := SummonActionSkills[id]; got != w {
			t.Errorf("action %d = %+v, want %+v", id, got, w)
		}
	}
}

func TestParseRequestChangePetName(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestChangePetName
	}{
		{"name", new(m3Body).s("Fang").b, RequestChangePetName{Name: "Fang"}},
		{"empty", new(m3Body).s("").b, RequestChangePetName{}},
		{"too long, not checked", new(m3Body).s("Fenrirwolf").b, RequestChangePetName{Name: "Fenrirwolf"}},
		{"trailing data", new(m3Body).s("Rex").d(7).b, RequestChangePetName{Name: "Rex"}},
	}
	for _, tt := range tests {
		got, err := ParseRequestChangePetName(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestChangePetName", new(m3Body).s("Fang").b, ParseRequestChangePetName)
}

func FuzzParseRequestChangePetName(f *testing.F) {
	m3Fuzz(f, ParseRequestChangePetName, -1, new(m3Body).s("Fang").b)
}

func TestParseRequestPetUseItem(t *testing.T) {
	// Mobius reads d object ID only; the client sends more.
	got, err := ParseRequestPetUseItem(new(m3Body).d(0x10000030).d(0).d(0).d(0).b)
	m3Parsed(t, "with the unread fields", got, err, RequestPetUseItem{ObjectID: 0x10000030})
	got, err = ParseRequestPetUseItem(new(m3Body).d(-1).b)
	m3Parsed(t, "negative", got, err, RequestPetUseItem{ObjectID: -1})
	m3Truncations(t, "RequestPetUseItem", new(m3Body).d(1).b, ParseRequestPetUseItem)
}

func FuzzParseRequestPetUseItem(f *testing.F) {
	m3Fuzz(f, ParseRequestPetUseItem, 4, new(m3Body).d(0x10000030).b)
}

func TestParseRequestGiveItemToPet(t *testing.T) {
	got, err := ParseRequestGiveItemToPet(new(m3Body).d(0x10000031).d(20).b)
	m3Parsed(t, "food", got, err, RequestGiveItemToPet{ObjectID: 0x10000031, Count: 20})
	got, err = ParseRequestGiveItemToPet(new(m3Body).d(1).d(-5).d(9).b)
	m3Parsed(t, "negative count, trailing data", got, err, RequestGiveItemToPet{ObjectID: 1, Count: -5})
	m3Truncations(t, "RequestGiveItemToPet", new(m3Body).d(1).d(2).b, ParseRequestGiveItemToPet)
}

func FuzzParseRequestGiveItemToPet(f *testing.F) {
	m3Fuzz(f, ParseRequestGiveItemToPet, 8, new(m3Body).d(0x10000031).d(20).b)
}

func TestParseRequestGetItemFromPet(t *testing.T) {
	got, err := ParseRequestGetItemFromPet(new(m3Body).d(0x10000032).d(3).d(0).b)
	m3Parsed(t, "item", got, err, RequestGetItemFromPet{ObjectID: 0x10000032, Count: 3})
	got, err = ParseRequestGetItemFromPet(new(m3Body).d(1).d(2).d(3).d(4).b)
	m3Parsed(t, "unknown word, trailing data", got, err, RequestGetItemFromPet{ObjectID: 1, Count: 2, Unknown: 3})
	m3Truncations(t, "RequestGetItemFromPet", new(m3Body).d(1).d(2).d(3).b, ParseRequestGetItemFromPet)
}

func FuzzParseRequestGetItemFromPet(f *testing.F) {
	m3Fuzz(f, ParseRequestGetItemFromPet, 12, new(m3Body).d(0x10000032).d(3).d(0).b)
}

func TestParseRequestPetGetItem(t *testing.T) {
	got, err := ParseRequestPetGetItem(new(m3Body).d(0x10000033).b)
	m3Parsed(t, "ground item", got, err, RequestPetGetItem{ObjectID: 0x10000033})
	m3Truncations(t, "RequestPetGetItem", new(m3Body).d(1).b, ParseRequestPetGetItem)
}

func FuzzParseRequestPetGetItem(f *testing.F) {
	m3Fuzz(f, ParseRequestPetGetItem, 4, new(m3Body).d(0x10000033).b)
}

// RequestActionUse with the summon actions: ctrl and shift come through.
func TestParseRequestActionUseSummon(t *testing.T) {
	got, err := ParseRequestActionUse(new(m3Body).d(ActionUsePetAttack).d(1).c(1).b)
	m3Parsed(t, "forced pet attack", got, err, RequestActionUse{ActionID: 16, CtrlPressed: true, ShiftPressed: true})
	got, err = ParseRequestActionUse(new(m3Body).d(ActionUseStriderWildStun).d(0).c(0).b)
	m3Parsed(t, "strider skill", got, err, RequestActionUse{ActionID: 1003})
}
