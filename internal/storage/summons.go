package storage

import "context"

// Summon state (Plan 14 M11f): what Mobius keeps in pets,
// character_summons, character_summon_skills_save and
// character_pet_skills_save. Pet items stay in the items collection with the
// player as their owner and the locations LocPet and LocPetEquip
// (PetInventory.getOwnerId).

// Item locations of a pet's inventory (Mobius PET and PET_EQUIP).
const (
	LocPet      ItemLocation = "PET"
	LocPetEquip ItemLocation = "PET_EQUIP"
)

// Pet is a pet's stored state (a Mobius pets row), keyed by the object ID
// of its collar (item_obj_id): the pet goes with the collar.
type Pet struct {
	// ControlID is the collar's object ID; OwnerID the character that last
	// stored the pet.
	ControlID, OwnerID int32
	// Name is the name its owner gave it, "" for none.
	Name  string
	Level int32
	// CurHP and CurMP are the HP and MP; a pet stored with less than 1 HP is
	// dead.
	CurHP, CurMP float64
	Exp          int64
	SP           int64
	// Fed is the meal.
	Fed int32
	// Restore is set when the pet comes back with its owner at login
	// (RestorePetOnReconnect: unsummoned by a logout).
	Restore bool
	// Effects are the effects kept over an unsummon (Mobius
	// character_pet_skills_save), in buff_index order.
	Effects []SavedSkill
}

// SavedServitor is a servitor that comes back with its owner at login
// (a Mobius character_summons row).
type SavedServitor struct {
	// SkillID is the skill that summoned it (summonSkillId).
	SkillID int32
	// CurHP and CurMP are the HP and MP, rounded.
	CurHP, CurMP int32
	// Time is the life time it had left, in milliseconds.
	Time int32
}

// ServitorEffects are the effects a servitor of one skill kept over an
// unsummon (Mobius character_summon_skills_save, keyed by owner, class index
// and summonSkillId), in buff_index order.
type ServitorEffects struct {
	ClassIndex    int32
	SummonSkillID int32
	Effects       []SavedSkill
}

// OwnerSummons is what a character keeps of its summons: the servitor to
// bring back, its servitors' saved effects, and the collar of the pet to
// bring back (Mobius CharSummonTable's servitors and pets maps), 0 for none.
type OwnerSummons struct {
	OwnerID    int32
	Servitor   *SavedServitor
	Effects    []ServitorEffects
	RestorePet int32
}

// SummonStore keeps the pets and the summon state of the characters. The
// game writes it in the background, never from the world loop.
type SummonStore interface {
	// LoadPet returns the pet of the collar, or ErrNotFound.
	LoadPet(ctx context.Context, controlID int32) (Pet, error)
	// SavePet stores the pet, replacing its stored state (Mobius storeMe's
	// INSERT or UPDATE).
	SavePet(ctx context.Context, p Pet) error
	// DeletePet deletes the pet of the collar; a missing one is no error
	// (Mobius destroyControlItem).
	DeletePet(ctx context.Context, controlID int32) error
	// LoadOwnerSummons returns a character's summon state, the zero state
	// with OwnerID set when it has none.
	LoadOwnerSummons(ctx context.Context, ownerID int32) (OwnerSummons, error)
	// SaveOwnerSummons replaces a character's summon state; the zero state
	// (no servitor, effects or pet) deletes it.
	SaveOwnerSummons(ctx context.Context, s OwnerSummons) error
}
