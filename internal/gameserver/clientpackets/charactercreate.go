package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// CharacterCreate asks to create a character. Race and the six stats are
// sent by the client but ignored by the server, as in Mobius: they come
// from the class template.
type CharacterCreate struct {
	Name      string
	Race      int32
	Sex       int32 // 0 male, anything else female
	ClassID   int32
	INT       int32
	STR       int32
	CON       int32
	MEN       int32
	DEX       int32
	WIT       int32
	HairStyle int32
	HairColor int32
	Face      int32
}

// Female reports whether the character is female. Mobius treats every
// non-zero value as female.
func (c CharacterCreate) Female() bool { return c.Sex != 0 }

// ParseCharacterCreate parses a CharacterCreate body.
func ParseCharacterCreate(data []byte) (CharacterCreate, error) {
	r := packets.NewReader(data)
	c := CharacterCreate{
		Name:      r.ReadString(),
		Race:      r.ReadInt32(),
		Sex:       r.ReadInt32(),
		ClassID:   r.ReadInt32(),
		INT:       r.ReadInt32(),
		STR:       r.ReadInt32(),
		CON:       r.ReadInt32(),
		MEN:       r.ReadInt32(),
		DEX:       r.ReadInt32(),
		WIT:       r.ReadInt32(),
		HairStyle: r.ReadInt32(),
		HairColor: r.ReadInt32(),
		Face:      r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return CharacterCreate{}, fmt.Errorf("CharacterCreate: %w", err)
	}
	return c, nil
}
