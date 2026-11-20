package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// friendSend handles a client packet of a session and waits for the loop.
func friendSend(e *testEnv, sess *fakeSession, op byte, body []byte) {
	e.srv.Handle(sess, op, body)
	e.sync()
}

func blockBody(typ int32, name string) []byte {
	w := packets.NewWriter(6 + 2*len(name))
	w.WriteInt32(typ)
	if typ == clientpackets.BlockAdd || typ == clientpackets.BlockRemove {
		w.WriteString(name)
	}
	return w.Bytes()
}

func friendMsgBody(message, receiver string) []byte {
	w := packets.NewWriter(4 + 2*(len(message)+len(receiver)))
	w.WriteString(message)
	w.WriteString(receiver)
	return w.Bytes()
}

// friendMsgs returns the system messages as "id" or "id:param,param".
func friendMsgs(ps []testclient.Packet) []string {
	var out []string
	for _, m := range only[testclient.SystemMessage](ps) {
		s := fmt.Sprint(m.ID)
		if texts := m.Texts(); len(texts) > 0 {
			s += ":" + strings.Join(texts, ",")
		}
		out = append(out, s)
	}
	return out
}

func checkFriendMsgs(t *testing.T, who string, ps []testclient.Packet, want ...string) {
	t.Helper()
	if got := friendMsgs(ps); !slices.Equal(got, want) {
		t.Errorf("%s got messages %q, want %q", who, got, want)
	}
}

// relationsOf reads the stored relations of a character once its pending
// writes are done.
func relationsOf(t *testing.T, e *testEnv, id int32) []storage.Relation {
	t.Helper()
	if err := e.srv.saver.wait(e.ctx, id); err != nil {
		t.Fatalf("wait: %v", err)
	}
	rels, err := e.store.Relations().Relations(e.ctx, id)
	if err != nil {
		t.Fatalf("Relations: %v", err)
	}
	return rels
}

// befriend makes Alice and Bob friends through the invite and the answer.
func befriend(t *testing.T, e *testEnv, sa, sb *fakeSession) {
	t.Helper()
	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("Bob"))
	friendSend(e, sb, clientpackets.OpRequestAnswerFriendInvite, int32Body(1))
	sa.take(t)
	sb.take(t)
}

// Phase 4 items 1 and 2: invite, accept, the friend window, /friendlist,
// friend messages, the online and offline notices, and the removal.
func TestFriendInviteAcceptAndNotices(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	b := e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb := newSession("alice"), newSession("bob")
	e.enter(sa, 0)
	e.enter(sb, 0)
	sa.take(t)
	sb.take(t)

	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("bob"))
	pa, pb := sa.take(t), sb.take(t)
	checkFriendMsgs(t, "Alice", pa, "1983:You've requested bob to be on your Friends List.")
	if req, ok := testclient.Find[testclient.FriendAddRequest](pb); !ok || req.Requestor != "Alice" {
		t.Errorf("Bob got FriendAddRequest %+v (%v)", req, ok)
	}
	checkFriendMsgs(t, "Bob", pb, "168:Alice")

	friendSend(e, sb, clientpackets.OpRequestAnswerFriendInvite, int32Body(1))
	pa, pb = sa.take(t), sb.take(t)
	checkFriendMsgs(t, "Alice", pa, "525", "132:Bob")
	checkFriendMsgs(t, "Bob", pb, "479:Alice")
	if f, ok := testclient.Find[testclient.L2Friend](pa); !ok || f.Action != testclient.FriendAdded || f.Name != "Bob" || !f.Online || f.ObjectID != b.ObjectID {
		t.Errorf("Alice's L2Friend %+v", f)
	}
	if f, ok := testclient.Find[testclient.L2Friend](pb); !ok || f.Action != testclient.FriendAdded || f.Name != "Alice" || f.ObjectID != a.ObjectID {
		t.Errorf("Bob's L2Friend %+v", f)
	}
	want := []storage.Relation{{Owner: a.ObjectID, Target: b.ObjectID, Type: storage.RelationFriend}}
	if got := relationsOf(t, e, a.ObjectID); !slices.Equal(got, want) {
		t.Errorf("Alice's stored relations %+v, want %+v", got, want)
	}
	if got := relationsOf(t, e, b.ObjectID); len(got) != 1 || got[0].Target != a.ObjectID {
		t.Errorf("Bob's stored relations %+v", got)
	}

	// Inviting again: already a friend.
	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "484")
	sb.take(t)

	// /friendlist
	friendSend(e, sa, clientpackets.OpRequestFriendList, nil)
	checkFriendMsgs(t, "Alice", sa.take(t), "487", "488:Bob", "490")

	// Friend messages: 1-300 characters to a friend online.
	friendSend(e, sa, clientpackets.OpRequestSendFriendMsg, friendMsgBody("hi", "bob"))
	if m, ok := testclient.Find[testclient.L2FriendSay](sb.take(t)); !ok || m.Sender != "Alice" || m.Receiver != "bob" || m.Message != "hi" {
		t.Errorf("Bob's L2FriendSay %+v (%v)", m, ok)
	}
	friendSend(e, sa, clientpackets.OpRequestSendFriendMsg, friendMsgBody(strings.Repeat("x", 301), "Bob"))
	friendSend(e, sa, clientpackets.OpRequestSendFriendMsg, friendMsgBody("", "Bob"))
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob got %q for an empty and a too long message", testclient.Names(ps))
	}
	checkFriendMsgs(t, "Alice", sa.take(t))
	friendSend(e, sa, clientpackets.OpRequestSendFriendMsg, friendMsgBody("hi", "Nobody"))
	checkFriendMsgs(t, "Alice", sa.take(t), "145")

	// Bob logs out: Alice sees him go offline.
	e.srv.Logout(sb)
	e.sync()
	if st, ok := testclient.Find[testclient.FriendStatus](sa.take(t)); !ok || st.Online || st.Name != "Bob" || st.ObjectID != b.ObjectID {
		t.Errorf("Alice's FriendStatus at Bob's logout %+v (%v)", st, ok)
	}
	friendSend(e, sa, clientpackets.OpRequestFriendList, nil)
	checkFriendMsgs(t, "Alice", sa.take(t), "487", "489:Bob", "490")

	// Bob comes back: his list has Alice online, and Alice gets the log-in
	// message and the status.
	sb2 := newSession("bob")
	e.enter(sb2, 0)
	ps := sb2.take(t)
	fl, ok := testclient.Find[testclient.FriendList](ps)
	if wantFL := []testclient.Friend{{ObjectID: a.ObjectID, Name: "Alice", Online: true, OnlineObjectID: a.ObjectID}}; !ok || !slices.Equal(fl.Friends, wantFL) {
		t.Errorf("Bob's FriendList %+v, want %+v", fl.Friends, wantFL)
	}
	names := testclient.Names(ps)
	if i := slices.Index(names, "FriendList"); i < 1 || names[i-1] != "ExStorageMaxCount" {
		t.Errorf("FriendList does not follow ExStorageMaxCount: %q", names)
	}
	ps = sa.take(t)
	checkFriendMsgs(t, "Alice", ps, "503:Bob")
	if st, ok := testclient.Find[testclient.FriendStatus](ps); !ok || !st.Online || st.Name != "Bob" {
		t.Errorf("Alice's FriendStatus at Bob's login %+v (%v)", st, ok)
	}
	if got := testclient.Names(ps); slices.Index(got, "SystemMessage") > slices.Index(got, "FriendStatusPacket") {
		t.Errorf("Alice got the status before the log-in message: %q", got)
	}

	// Deleting a friend online removes both entries and both relations.
	friendSend(e, sa, clientpackets.OpRequestFriendDel, stringBody("Bob"))
	pa, pb = sa.take(t), sb2.take(t)
	checkFriendMsgs(t, "Alice", pa, "481:Bob")
	if f, ok := testclient.Find[testclient.L2Friend](pa); !ok || f.Action != testclient.FriendRemoved || f.Name != "Bob" {
		t.Errorf("Alice's L2Friend %+v", f)
	}
	if f, ok := testclient.Find[testclient.L2Friend](pb); !ok || f.Action != testclient.FriendRemoved || f.Name != "Alice" {
		t.Errorf("Bob's L2Friend %+v", f)
	}
	if got := relationsOf(t, e, a.ObjectID); len(got) != 0 {
		t.Errorf("Alice's relations after the removal %+v", got)
	}
	if got := relationsOf(t, e, b.ObjectID); len(got) != 0 {
		t.Errorf("Bob's relations after the removal %+v", got)
	}
	friendSend(e, sa, clientpackets.OpRequestFriendDel, stringBody("Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "171:Bob")
	// Bob no longer lists Alice: her messages do not reach him.
	friendSend(e, sa, clientpackets.OpRequestSendFriendMsg, friendMsgBody("hi", "Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "145")
}

// Removing an offline friend removes the friendship on both sides; only the
// remover is told (Mobius RequestFriendDel).
func TestFriendDeleteOffline(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	b := e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb := newSession("alice"), newSession("bob")
	e.enter(sa, 0)
	e.enter(sb, 0)
	befriend(t, e, sa, sb)
	e.srv.Logout(sb)
	e.sync()
	sa.take(t)

	friendSend(e, sa, clientpackets.OpRequestFriendDel, stringBody("BOB"))
	ps := sa.take(t)
	checkFriendMsgs(t, "Alice", ps, "481:BOB")
	if _, ok := testclient.Find[testclient.L2Friend](ps); ok {
		t.Error("Alice got L2Friend for an offline friend")
	}
	for _, id := range []int32{a.ObjectID, b.ObjectID} {
		if got := relationsOf(t, e, id); len(got) != 0 {
			t.Errorf("relations of %d after the removal: %+v", id, got)
		}
	}
	sb2 := newSession("bob")
	e.enter(sb2, 0)
	if fl, ok := testclient.Find[testclient.FriendList](sb2.take(t)); !ok || len(fl.Friends) != 0 {
		t.Errorf("Bob's FriendList %+v", fl)
	}
	if msgs := friendMsgs(sa.take(t)); slices.Contains(msgs, "503:Bob") {
		t.Error("Alice was told of Bob's login after the removal")
	}
}

// The invite checks in Mobius's order, a refusal, and an answer after the
// request expired.
func TestFriendInviteChecks(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("carol", "Carol", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb, sc := newSession("alice"), newSession("bob"), newSession("carol")
	e.enter(sa, 0)
	e.enter(sb, 0)
	e.enter(sc, 0)
	sa.take(t)
	sb.take(t)
	sc.take(t)

	for _, tc := range []struct {
		name string
		want []string
	}{
		{"Nobody", []string{"170"}},
		{"Alice", []string{"165"}},
	} {
		friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody(tc.name))
		checkFriendMsgs(t, "Alice inviting "+tc.name, sa.take(t), tc.want...)
	}

	// Bob blocks Alice: she is told she is on his list.
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Alice"))
	sa.take(t)
	sb.take(t)
	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "1983:You are in Bob's block list.")
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob got %q", testclient.Names(ps))
	}
	// And Bob may not invite Alice, whom he blocks.
	friendSend(e, sb, clientpackets.OpRequestFriendInvite, stringBody("alice"))
	checkFriendMsgs(t, "Bob", sb.take(t), "2057:Alice")
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRemove, "Alice"))
	sb.take(t)

	// Carol refuses all messages: nobody can invite her.
	friendSend(e, sc, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRefuseAll, ""))
	sc.take(t)
	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("Carol"))
	checkFriendMsgs(t, "Alice", sa.take(t), "1983:You are in Carol's block list.")
	friendSend(e, sc, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAcceptAll, ""))
	sc.take(t)

	// Bob invites Carol; while she has not answered she is busy for Alice.
	friendSend(e, sb, clientpackets.OpRequestFriendInvite, stringBody("Carol"))
	sb.take(t)
	sc.take(t)
	friendSend(e, sa, clientpackets.OpRequestFriendInvite, stringBody("carol"))
	checkFriendMsgs(t, "Alice", sa.take(t), "153:carol")
	// Carol declines.
	friendSend(e, sc, clientpackets.OpRequestAnswerFriendInvite, int32Body(0))
	checkFriendMsgs(t, "Bob", sb.take(t), "526")
	if ps := sc.take(t); len(ps) != 0 {
		t.Errorf("Carol got %q for declining", testclient.Names(ps))
	}

	// Bob invites Carol again; she answers after 15 s: nothing happens.
	friendSend(e, sb, clientpackets.OpRequestFriendInvite, stringBody("Carol"))
	sb.take(t)
	sc.take(t)
	e.advance(requestTimeout + time.Second)
	friendSend(e, sc, clientpackets.OpRequestAnswerFriendInvite, int32Body(1))
	ps := sb.take(t)
	checkFriendMsgs(t, "Bob, after an expired invite", ps)
	if _, ok := testclient.Find[testclient.L2Friend](ps); ok {
		t.Error("Bob got L2Friend for an expired invite")
	}
	var friends bool
	e.onLoop(func() {
		pb, _ := e.srv.PlayerByName("Bob")
		friends = pb.IsFriend(pb.ObjectID()) || len(pb.FriendIDs()) > 0
	})
	if friends {
		t.Error("an expired invite made friends")
	}
}

// Phase 4 items 3 and 4: the block list and message refusal, and blocks on
// whispers.
func TestBlockList(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	b := e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	c := e.createCharacter("carol", "Carol", classHumanFighter, storage.AccessLevelPlayer)
	dave := e.createCharacter("dave", "Dave", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("gm", "Gamemaster", classHumanFighter, storage.AccessLevelAdmin)
	sa, sb, sc := newSession("alice"), newSession("bob"), newSession("carol")
	e.enter(sa, 0)
	e.enter(sb, 0)
	e.enter(sc, 0)
	befriend(t, e, sa, sb)
	sc.take(t)

	// Alice blocks Carol, who is told.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "carol"))
	checkFriendMsgs(t, "Alice", sa.take(t), "617:Carol")
	checkFriendMsgs(t, "Carol", sc.take(t), "619:Alice")
	want := []storage.Relation{
		{Owner: a.ObjectID, Target: c.ObjectID, Type: storage.RelationBlock},
	}
	if got := relationsOf(t, e, a.ObjectID); !slices.Contains(got, want[0]) {
		t.Errorf("Alice's relations %+v, want the block %+v", got, want[0])
	}
	// Carol's whispers to Alice are refused; Bob's still arrive.
	chatSend(e, sc, serverpackets.ChatWhisper, "psst", "Alice")
	checkFriendMsgs(t, "Carol", sc.take(t), "176")
	if said := only[testclient.CreatureSay](sa.take(t)); len(said) != 0 {
		t.Errorf("Alice heard Carol: %+v", said)
	}
	chatSend(e, sb, serverpackets.ChatWhisper, "hello", "Alice")
	if said := only[testclient.CreatureSay](sa.take(t)); len(said) != 1 {
		t.Errorf("Alice heard %+v from Bob", said)
	}
	sb.take(t)

	// Again: already on the list. A friend, a GM, nobody and oneself cannot
	// be blocked.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Carol"))
	checkFriendMsgs(t, "Alice", sa.take(t), "1983:Already in ignore list.")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "484")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Alice"))
	checkFriendMsgs(t, "Alice", sa.take(t))
	// Offline characters are looked up in the store, off the loop.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Gamemaster"))
	sa.waitFor(t, "the GM answer", func(s *fakeSession) bool { return len(s.bodies) > 0 })
	checkFriendMsgs(t, "Alice", sa.take(t), "827")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Nobody"))
	sa.waitFor(t, "the unknown name answer", func(s *fakeSession) bool { return len(s.bodies) > 0 })
	checkFriendMsgs(t, "Alice", sa.take(t), "615")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "dave"))
	sa.waitFor(t, "Dave's block", func(s *fakeSession) bool { return len(s.bodies) > 0 })
	checkFriendMsgs(t, "Alice", sa.take(t), "617:Dave")

	// The list.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockShowList, ""))
	checkFriendMsgs(t, "Alice", sa.take(t), "613", "1983:1. Carol", "1983:2. Dave", "490")

	// Unblocking: one on the list, then one who is not.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRemove, "CAROL"))
	checkFriendMsgs(t, "Alice", sa.take(t), "618:Carol")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRemove, "Bob"))
	checkFriendMsgs(t, "Alice", sa.take(t), "144")
	chatSend(e, sc, serverpackets.ChatWhisper, "psst", "Alice")
	if said := only[testclient.CreatureSay](sa.take(t)); len(said) != 1 {
		t.Errorf("Alice heard %+v from Carol after unblocking her", said)
	}
	sc.take(t)
	wantRels := []storage.Relation{
		{Owner: a.ObjectID, Target: b.ObjectID, Type: storage.RelationFriend},
		{Owner: a.ObjectID, Target: dave.ObjectID, Type: storage.RelationBlock},
	}
	if got := relationsOf(t, e, a.ObjectID); !slices.Equal(got, wantRels) {
		t.Errorf("Alice's relations %+v, want %+v", got, wantRels)
	}

	// Refuse all: the status icon, and every whisper is refused, a friend's
	// too.
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRefuseAll, ""))
	ps := sa.take(t)
	checkFriendMsgs(t, "Alice", ps, "177")
	if st, ok := testclient.Find[testclient.EtcStatusUpdate](ps); !ok || !st.ChatBlocked {
		t.Errorf("EtcStatusUpdate %+v (%v), want the chat block icon", st, ok)
	}
	chatSend(e, sb, serverpackets.ChatWhisper, "hello", "Alice")
	checkFriendMsgs(t, "Bob", sb.take(t), "176")
	friendSend(e, sa, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAcceptAll, ""))
	ps = sa.take(t)
	checkFriendMsgs(t, "Alice", ps, "178")
	if st, ok := testclient.Find[testclient.EtcStatusUpdate](ps); !ok || st.ChatBlocked {
		t.Errorf("EtcStatusUpdate %+v (%v), want no chat block icon", st, ok)
	}

	// The block list comes back with the character.
	e.srv.Logout(sa)
	e.sync()
	sa2 := newSession("alice")
	pa := e.enter(sa2, 0)
	var blocked bool
	e.onLoop(func() {
		blocked = pa.HasBlocked(dave.ObjectID) && !pa.HasBlocked(c.ObjectID) && pa.IsFriend(b.ObjectID)
	})
	if !blocked {
		t.Error("Alice's lists did not come back with her")
	}
}
