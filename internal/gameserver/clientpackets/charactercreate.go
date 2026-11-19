package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// CharacterCreate asks to create a character.
type CharacterCreate struct {
	Name      string
	Race      uint32
	Sex       uint32
	ClassID   uint32
	INT       uint32
	STR       uint32
	CON       uint32
	MEN       uint32
	DEX       uint32
	WIT       uint32
	HairStyle uint32
	HairColor uint32
	Face      uint32
}

// ParseCharacterCreate parses a CharacterCreate body.
func ParseCharacterCreate(data []byte) (CharacterCreate, error) {
	r := packets.NewReader(data)
	c := CharacterCreate{
		Name:      r.ReadString(),
		Race:      r.ReadUint32(),
		Sex:       r.ReadUint32(),
		ClassID:   r.ReadUint32(),
		INT:       r.ReadUint32(),
		STR:       r.ReadUint32(),
		CON:       r.ReadUint32(),
		MEN:       r.ReadUint32(),
		DEX:       r.ReadUint32(),
		WIT:       r.ReadUint32(),
		HairStyle: r.ReadUint32(),
		HairColor: r.ReadUint32(),
		Face:      r.ReadUint32(),
	}
	if err := r.Err(); err != nil {
		return CharacterCreate{}, fmt.Errorf("CharacterCreate: %w", err)
	}
	return c, nil
}
