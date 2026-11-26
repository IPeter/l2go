package data

import "testing"

func TestCommittedClanHalls(t *testing.T) {
	c, err := LoadClanHalls("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Halls) != 44 || len(c.Guards) != 729 || len(c.Auctions) != 38 {
		t.Fatalf("%d halls, %d guards, %d auctions; want 44, 729, 38", len(c.Halls), len(c.Guards), len(c.Auctions))
	}
	siegable := 0
	for _, h := range c.Halls {
		if h.Siegable {
			siegable++
		}
	}
	if siegable != 6 {
		t.Errorf("%d contestable halls, want 6", siegable)
	}
	if h, ok := c.Hall(22); !ok || h.Name != "Moonstone Hall" || h.Lease != 500000 || h.Grade != 2 || h.Location != "Gludio" {
		t.Errorf("Hall(22) = %+v, %v", h, ok)
	}
	if h, ok := c.Hall(21); !ok || !h.Siegable || h.SiegeLength != 3600000 || h.Schedule != [5]int{14, 0, 0, 12, 0} {
		t.Errorf("Hall(21) = %+v, %v", h, ok)
	}
	if a, ok := c.Auction(36); !ok || a.StartingBid != 50000000 || a.ItemName != "The Golden Chamber" {
		t.Errorf("Auction(36) = %+v, %v", a, ok)
	}
	if _, ok := c.Auction(21); ok {
		t.Error("a contestable hall has an NPC auction")
	}
	if g := c.GuardsOf(34); len(g) == 0 || g[0].NpcID != 35408 {
		t.Errorf("GuardsOf(34) = %+v", g)
	}
}

func TestNewClanHallsChecks(t *testing.T) {
	_, err := NewClanHalls(ClanHallData{
		Halls:    []ClanHall{{ID: 22, Name: "A"}, {ID: 21, Name: "B", Siegable: true}},
		Guards:   []ClanHallGuard{{HallID: 99, NpcID: 1}},
		Auctions: []ClanHallAuction{{HallID: 22}, {HallID: 22}},
	})
	if err == nil {
		t.Fatal("bad data passed")
	}
}
