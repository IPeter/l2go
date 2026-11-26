package scripting_test

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Monument of Heroes (Mobius ai/others/MonumentOfHeroes): the claim of
// an unclaimed hero, one weapon and one circlet for a hero.
func TestMonumentOfHeroes(t *testing.T) {
	h := scripttest.Load(t, "../../scripts", "npcs/monument_of_heroes/MonumentOfHeroes.lua")
	monument := &scripttest.Npc{ObjectID: 0x60000001, ID: 31690, Name: "Monument of Heroes"}
	noble := &scripttest.Player{ObjectID: 0x10000001, Name: "Noble", Level: 76, Noble: true}
	elected := &scripttest.Player{ObjectID: 0x10000002, Name: "Elected", Level: 78, Noble: true}
	h.World().SetUnclaimedHero(elected)

	page := func(p *scripttest.Player, event, want string) {
		t.Helper()
		reply, ok := h.Event("MonumentOfHeroes", event, p, monument)
		if !ok || reply.Kind != scripting.ReplyPage || reply.Content != want {
			t.Errorf("%s by %s = %+v, %v; want %s", event, p.Name, reply, ok, want)
		}
	}
	none := func(p *scripttest.Player, event string) {
		t.Helper()
		if reply, ok := h.Event("MonumentOfHeroes", event, p, monument); ok && reply.Kind != scripting.ReplyNone {
			t.Errorf("%s by %s = %+v, want nothing", event, p.Name, reply)
		}
	}

	// The claim.
	page(noble, "HeroClaim", "no_hero_status.htm")
	page(elected, "HeroClaim", "receive_hero.htm")
	page(noble, "HeroWeapon", "not_a_hero.htm")
	page(noble, "HeroCirclet", "not_a_hero.htm")
	none(elected, "HeroReceive")
	if !elected.Hero || !slices.Equal(h.World().HeroClaims(), []int32{elected.ObjectID}) {
		t.Fatalf("after HeroReceive: hero %v, claims %v", elected.Hero, h.World().HeroClaims())
	}
	page(elected, "HeroClaim", "already_hero_status.htm")

	// One weapon of the list.
	page(elected, "HeroWeapon", "weapon_list.htm")
	none(elected, "6614")
	if elected.Items[6614] != 1 {
		t.Errorf("Infinity Rod: %d", elected.Items[6614])
	}
	page(elected, "HeroWeapon", "already_have_weapon.htm")
	none(elected, "57") // not a hero weapon
	none(elected, "nonsense")
	if elected.Items[57] != 0 {
		t.Error("adena given for a weapon event")
	}

	// One circlet.
	none(elected, "HeroCirclet")
	if elected.Items[6842] != 1 {
		t.Errorf("circlet: %d", elected.Items[6842])
	}
	page(elected, "HeroCirclet", "already_have_circlet.htm")
	if elected.Items[6842] != 1 {
		t.Errorf("a second circlet was given: %d", elected.Items[6842])
	}
}
