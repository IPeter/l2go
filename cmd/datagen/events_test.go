package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/data"
)

func TestReadEvent(t *testing.T) {
	root, err := parseXMLTree([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<event name="L2 Day" active="23 03-28 03" dropPeriod="24 03-27 03" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<droplist>
		<add item="3875" min="1" max="2" chance="1.5%" />
		<add item="3876" min="1" max="1" chance="10%" minLevel="20" maxLevel="40" monsterIds="20001,20000,20001" />
	</droplist>
	<spawnlist>
		<add npc="31854" x="-84128" y="243258" z="-3735" heading="9707" />
		<add npc="31855" x="1" y="2" z="3" respawnTime="60" />
	</spawnlist>
	<messages>
		<add type="onEnd" text="bye" />
		<add type="onEnter" text="hello" />
	</messages>
	<destroyItemsOnEnd>
		<item id="3875" />
	</destroyItemsOnEnd>
</event>`), "events/L2Day/config.xml")
	if err != nil {
		t.Fatal(err)
	}
	e, err := readEvent(root, "L2Day")
	if err != nil {
		t.Fatal(err)
	}
	if e.Script != "L2Day" || e.Name != "L2 Day" || e.Active != "23 03-28 03" || e.DropPeriod != "24 03-27 03" {
		t.Errorf("header %+v", e)
	}
	want := []data.EventDrop{
		{ItemID: 3875, Min: 1, Max: 2, Chance: 1.5, MinLevel: 1, MaxLevel: 2147483647},
		{ItemID: 3876, Min: 1, Max: 1, Chance: 10, MinLevel: 20, MaxLevel: 40, MonsterIDs: []int{20000, 20001}},
	}
	if !slices.EqualFunc(e.Drops, want, func(a, b data.EventDrop) bool {
		return a.ItemID == b.ItemID && a.Min == b.Min && a.Max == b.Max && a.Chance == b.Chance &&
			a.MinLevel == b.MinLevel && a.MaxLevel == b.MaxLevel && slices.Equal(a.MonsterIDs, b.MonsterIDs)
	}) {
		t.Errorf("drops %+v", e.Drops)
	}
	spawns := []data.EventSpawn{{NpcID: 31854, X: -84128, Y: 243258, Z: -3735, Heading: 9707}, {NpcID: 31855, X: 1, Y: 2, Z: 3, RespawnTime: 60}}
	if !slices.Equal(e.Spawns, spawns) {
		t.Errorf("spawns %+v", e.Spawns)
	}
	if e.OnEnter != "hello" || e.OnEnd != "bye" || !slices.Equal(e.DestroyItemsOnEnd, []int{3875}) {
		t.Errorf("messages %q %q, destroy %v", e.OnEnter, e.OnEnd, e.DestroyItemsOnEnd)
	}
	if _, err := data.NewLongTimeEvents([]data.LongTimeEvent{e}); err != nil {
		t.Error(err)
	}
}

func TestReadEventErrors(t *testing.T) {
	for _, tc := range []struct{ xml, want string }{
		{`<list/>`, "want <event>"},
		{`<event name="x"/>`, "no active"},
		{`<event name="x" active="01 01-02 01" other="1"/>`, "unknown attribute"},
		{`<event name="x" active="01 01-02 01"><foo/></event>`, "unknown element"},
		{`<event name="x" active="01 01-02 01"><droplist><add item="1" min="1" max="1" chance="5"/></droplist></event>`, "percentage"},
		{`<event name="x" active="01 01-02 01"><droplist><add item="1" min="2" max="1" chance="5%"/></droplist></event>`, "min greater"},
		{`<event name="x" active="01 01-02 01"><droplist><add item="1" min="1" max="1" chance="101%"/></droplist></event>`, "chance"},
		{`<event name="x" active="01 01-02 01"><spawnlist><add npc="1" x="1" y="2"/></spawnlist></event>`, "no z"},
		{`<event name="x" active="01 01-02 01"><messages><add type="onStart" text="t"/></messages></event>`, "message type"},
	} {
		root, err := parseXMLTree([]byte(tc.xml), "config.xml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := readEvent(root, "X"); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", tc.xml, err, tc.want)
		}
	}
}

// The committed event data is what datagen makes from Mobius.
func TestCommittedEventsMatchMobius(t *testing.T) { committed(t, repoData, false, "events") }
