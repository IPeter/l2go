package gameserver

import (
	"context"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/testclient"
)

// clanGame is the fake game with the clan rule of character deletion: a
// character named "Leader" leads its clan, any other with a clan is a
// member.
type clanGame struct{ *fakeGame }

func (g clanGame) CharacterClanDeletion(ctx context.Context, objectID, clanID int32) (game.ClanDeletion, error) {
	if clanID == 0 {
		return game.ClanDeletionAllowed, nil
	}
	ch, err := g.chars.Get(ctx, objectID)
	if err != nil {
		return 0, err
	}
	if ch.Name == "Leader" {
		return game.ClanDeletionLeader, nil
	}
	return game.ClanDeletionMember, nil
}

// Clan members and leaders cannot be deleted (Mobius CharacterDelete with
// markToDeleteChar's answers 1 and 2): CharDeleteFail with the reason, then
// the list; a character without a clan is deleted as before.
func TestCharacterDeleteClanMembers(t *testing.T) {
	e := newEnv(t, envOptions{server: func(s *Server) { s.game = clanGame{s.game.(*fakeGame)} }})
	c, list := withCharacters(t, e, "player1", "Leader", "Member", "Free")
	for _, name := range []string{"Leader", "Member"} {
		ch, ok := stored(t, e, "player1", name)
		if !ok {
			t.Fatalf("no %s", name)
		}
		ch.ClanID = 0x10000100
		if err := e.chars.Update(testCtx(t), &ch); err != nil {
			t.Fatal(err)
		}
	}
	slot := func(name string) int32 {
		i := slices.IndexFunc(list.Characters, func(ch testclient.CharInfo) bool { return ch.Name == name })
		if i < 0 {
			t.Fatalf("%s is not listed", name)
		}
		return int32(i)
	}
	for _, tc := range []struct {
		name   string
		reason testclient.DeleteFailReason
	}{{"Leader", testclient.DeleteFailClanLeader}, {"Member", testclient.DeleteFailClanMember}} {
		res := deleteSlot(t, c, slot(tc.name))
		if res.Deleted || !res.Failed || res.Reason != tc.reason {
			t.Errorf("%s: %+v, want CharDeleteFail %v", tc.name, res, tc.reason)
		}
		if ch, _ := stored(t, e, "player1", tc.name); ch.Deleting() {
			t.Errorf("%s was marked for deletion", tc.name)
		}
		list = res.Characters
	}
	if res := deleteSlot(t, c, slot("Free")); !res.Deleted {
		t.Errorf("Free: %+v, want CharDeleteSuccess", res)
	}
}
