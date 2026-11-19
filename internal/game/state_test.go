package game

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/world"
)

func TestZones(t *testing.T) {
	var z Zones
	if z.Inside(ZonePeace) || z.Inside(NumZoneFlags) {
		t.Fatal("an empty set is inside a zone")
	}
	// Two overlapping peace zones: leaving one keeps the flag.
	z.Enter(ZonePeace)
	z.Enter(ZonePeace)
	z.Leave(ZonePeace)
	if !z.Inside(ZonePeace) {
		t.Error("left both peace zones after leaving one of two")
	}
	z.Leave(ZonePeace)
	z.Leave(ZonePeace) // one more than entered does nothing
	if z.Inside(ZonePeace) {
		t.Error("still in a peace zone after leaving both")
	}
	z.Enter(ZonePeace)
	if !z.Inside(ZonePeace) {
		t.Error("leaving too often broke the counter")
	}

	if !z.Add(11) || !z.Add(7) || z.Add(11) {
		t.Error("Add reports wrongly")
	}
	if !z.In(7) || z.In(8) {
		t.Error("In reports wrongly")
	}
	if !z.Remove(11) || z.Remove(11) {
		t.Error("Remove reports wrongly")
	}
	if got := z.IDs(); !slices.Equal(got, []int{7}) {
		t.Errorf("IDs() = %v, want [7]", got)
	}
}

func TestHTMLCache(t *testing.T) {
	const npc world.ObjectID = 0x60000001
	page := `<html><body>
<a action="bypass -h npc_%objectId%_Chat 1">Talk</a>
<a action="bypass -h npc_1610612737_Quest">Quest</a>
<edit var="amount"><a action="bypass -h npc_1610612737_buy $amount">Buy</a>
<a action="link default/30598-1.htm">More</a>
</body></html>`

	var c HTMLCache
	if _, ok := c.Validate("npc_1610612737_Quest"); ok {
		t.Fatal("an empty cache allowed an action")
	}
	c.Set(HTMLScopeNpc, npc, page)
	for _, tc := range []struct {
		action string
		ok     bool
	}{
		{"npc_1610612737_Quest", true},
		{"npc_1610612737_buy 100", true},   // a $ parameter allows any value
		{"link default/30598-1.htm", true}, // links are cached with their prefix
		{"npc_1610612737_Chat 2", false},
		{"npc_1610612737_Quest ", false},
		{"admin_teleport 0 0 0", false},
	} {
		origin, ok := c.Validate(tc.action)
		if ok != tc.ok || (ok && origin != npc) {
			t.Errorf("Validate(%q) = %v, %v; want %v from the NPC", tc.action, origin, ok, tc.ok)
		}
	}
	if c.LastOrigin() != npc || c.Origin(HTMLScopeNpc) != npc {
		t.Errorf("origins %v %v, want %v", c.LastOrigin(), c.Origin(HTMLScopeNpc), npc)
	}

	// Another scope keeps its own page; clearing one scope leaves the other.
	c.Set(HTMLScopeNpcQuest, 0, `<a action="bypass -h Quest Q00001_Letters 30048-02.htm">ok</a>`)
	if origin, ok := c.Validate("Quest Q00001_Letters 30048-02.htm"); !ok || origin != 0 {
		t.Errorf("quest page action: %v, %v", origin, ok)
	}
	c.Clear(HTMLScopeNpc)
	if _, ok := c.Validate("npc_1610612737_Quest"); ok {
		t.Error("a cleared scope still allows its actions")
	}
	if _, ok := c.Validate("Quest Q00001_Letters 30048-02.htm"); !ok {
		t.Error("clearing one scope cleared another")
	}
}
