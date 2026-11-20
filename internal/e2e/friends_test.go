package e2e

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// sysMsgFriendLoggedIn is "$s1 (Friend) has logged in." (Mobius
// S1_FRIEND_HAS_LOGGED_IN).
const sysMsgFriendLoggedIn int32 = 503

// TestFriendsSeeEachOther runs Plan 12's friend demo (checklist item 4)
// through two test clients against the in-process stack: A invites B, B
// accepts, both windows list the other online; B logs out and in again and
// A gets both notices; friend messages arrive; B's list after the relog has
// A online.
func TestFriendsSeeEachOther(t *testing.T) {
	s := startStack(t)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	const password = "Secret-1"
	join := func(account, name string, create bool) (*testclient.GameConn, testclient.EnterWorldResult) {
		t.Helper()
		c := gameLogin(t, ctx, s, account, password)
		if create {
			if res, err := c.CreateCharacter(ctx, humanFighter(name)); err != nil || !res.Created {
				t.Fatalf("CreateCharacter %s: %+v, %v", name, res, err)
			}
		}
		if _, err := c.SelectCharacter(ctx, 0); err != nil {
			t.Fatalf("SelectCharacter %s: %v", name, err)
		}
		res, err := c.EnterWorld(ctx)
		if err != nil {
			t.Fatalf("EnterWorld %s: %v", name, err)
		}
		return c, res
	}
	a, _ := join("friendsa", "Amity", true)
	b, _ := join("friendsb", "Bonder", true)

	if err := a.InviteFriend(ctx, "Bonder"); err != nil {
		t.Fatalf("InviteFriend: %v", err)
	}
	if req, err := b.AwaitFriendInvite(ctx); err != nil || req.Requestor != "Amity" {
		t.Fatalf("B's invitation %+v, %v", req, err)
	}
	added, err := b.AcceptFriendInvite(ctx)
	if err != nil || added.Name != "Amity" || !added.Online {
		t.Fatalf("B's L2Friend %+v, %v", added, err)
	}
	aAdded := await(t, ctx, a, func(f testclient.L2Friend) bool { return f.Action == testclient.FriendAdded })
	if aAdded.Name != "Bonder" || !aAdded.Online {
		t.Errorf("A's L2Friend %+v", aAdded)
	}
	bID := aAdded.ObjectID

	if err := a.SendFriendMessage(ctx, "Bonder", "hello friend"); err != nil {
		t.Fatalf("SendFriendMessage: %v", err)
	}
	if m := await[testclient.L2FriendSay](t, ctx, b, nil); m.Sender != "Amity" || m.Message != "hello friend" {
		t.Errorf("B's friend message %+v", m)
	}

	// B logs out: A sees B go offline.
	if err := b.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if st := await[testclient.FriendStatus](t, ctx, a, nil); st.Online || st.Name != "Bonder" || st.ObjectID != bID {
		t.Errorf("A's FriendStatus at B's logout %+v", st)
	}

	// B comes back: A gets the log-in message and the status, and B's
	// FriendList has A online.
	b, res := join("friendsb", "Bonder", false)
	fl, ok := testclient.Find[testclient.FriendList](res.Packets)
	if !ok || len(fl.Friends) != 1 || fl.Friends[0].Name != "Amity" || !fl.Friends[0].Online {
		t.Errorf("B's FriendList after the relog %+v (%v)", fl, ok)
	}
	msg := await(t, ctx, a, func(m testclient.SystemMessage) bool { return m.ID == sysMsgFriendLoggedIn })
	if !slices.Equal(msg.Texts(), []string{"Bonder"}) {
		t.Errorf("A's log-in message %+v", msg)
	}
	if st := await[testclient.FriendStatus](t, ctx, a, nil); !st.Online || st.Name != "Bonder" {
		t.Errorf("A's FriendStatus at B's login %+v", st)
	}
	if err := b.SendFriendMessage(ctx, "Amity", "back again"); err != nil {
		t.Fatalf("SendFriendMessage: %v", err)
	}
	if m := await[testclient.L2FriendSay](t, ctx, a, nil); m.Sender != "Bonder" || m.Message != "back again" {
		t.Errorf("A's friend message %+v", m)
	}
}
