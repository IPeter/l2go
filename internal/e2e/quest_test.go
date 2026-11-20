package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// Letters of Love (Q001) and its NPCs and items, from the Mobius data.
const (
	q001       = "Q00001_LettersOfLove"
	npcDarin   = 30048
	npcBaulro  = 30033
	itemLetter = 687
	itemNeckla = 906
)

// Where the Q001 NPCs stand (spawns/TalkingIsland), with a spot next to
// each.
var (
	darinSpot  = testclient.Position{X: -84406, Y: 242793, Z: -3728}
	roxxyNear  = testclient.Position{X: -84078, Y: 244604, Z: -3728}
	baulroSpot = testclient.Position{X: -84699, Y: 245001, Z: -3720}
)

// TestQuestLettersOfLove walks Q001 end to end through the test client
// against the in-process stack with the shipped scripts and the Mobius
// pages: Darin's quest link, the accept page with its sound and quest log
// entry, Roxxy, Darin, Baulro and Darin again for the necklace, with a
// logout in the middle. The pages are not in git, so it needs make data.
func TestQuestLettersOfLove(t *testing.T) {
	pages := filepath.Join("..", "..", "data", "html", "scripts", "quests", q001)
	if _, err := os.Stat(pages); err != nil {
		t.Skipf("no Q001 pages in %s: make data copies them from the Mobius checkout", pages)
	}
	s := startStack(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	const account, password, name = "letters", "Secret-1", "Lover"
	c := gameLogin(t, ctx, s, account, password)
	res, err := c.CreateCharacter(ctx, humanFighter(name))
	if err != nil || !res.Created {
		t.Fatalf("CreateCharacter: %+v, %v", res, err)
	}
	// Level 2 (Darin wants it) and a GM, to teleport between the NPCs.
	s.updateCharacter(t, account, func(ch *storage.Character) {
		ch.Level, ch.Exp = 2, s.game.ExpForLevel(2)
		ch.AccessLevel = storage.AccessLevelAdmin
	})
	if _, err := c.SelectCharacter(ctx, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}

	teleport(t, ctx, c, darinSpot)
	page := questLink(t, ctx, c, npcDarin)
	page = follow(t, ctx, c, requireLink(t, page, "Script "+q001+" 30048-03.htm"))
	page = follow(t, ctx, c, requireLink(t, page, "Script "+q001+" 30048-04.htm"))

	// Accepting: QuestList with step 1, the quest mark, the accept sound,
	// the letter, and the page.
	bypass(t, ctx, c, requireLink(t, page, "Script "+q001+" 30048-06.htm"))
	_, before := until[testclient.NpcHtmlMessage](t, ctx, c, nil)
	var got []string
	for _, n := range testclient.Names(before) {
		if n == "QuestList" || n == "ExShowQuestMark" || n == "PlaySound" {
			got = append(got, n)
		}
	}
	if want := []string{"QuestList", "ExShowQuestMark", "QuestList", "PlaySound"}; !slices.Equal(got, want) {
		t.Errorf("accept packets %q, want %q", got, want)
	}
	if snd, ok := testclient.Find[testclient.PlaySound](before); !ok || snd.Sound != testclient.SoundQuestAccept {
		t.Errorf("accept sound %+v", snd)
	}
	checkQuests(t, ctx, c, []testclient.Quest{{ID: 1, State: 1}})

	teleport(t, ctx, c, roxxyNear)
	questLink(t, ctx, c, npcRoxxy)
	checkQuests(t, ctx, c, []testclient.Quest{{ID: 1, State: 2}})

	// Log out in the middle and come back: the step stays.
	if err := c.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	c = gameLogin(t, ctx, s, account, password)
	if _, err := c.SelectCharacter(ctx, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	enterRes, err := c.EnterWorld(ctx)
	if err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	if !slices.Equal(enterRes.Quests.Quests, []testclient.Quest{{ID: 1, State: 2}}) {
		t.Errorf("quest log after the relog %+v", enterRes.Quests.Quests)
	}

	teleport(t, ctx, c, darinSpot)
	questLink(t, ctx, c, npcDarin)
	checkQuests(t, ctx, c, []testclient.Quest{{ID: 1, State: 3}})
	teleport(t, ctx, c, baulroSpot)
	questLink(t, ctx, c, npcBaulro)
	checkQuests(t, ctx, c, []testclient.Quest{{ID: 1, State: 4}})
	teleport(t, ctx, c, darinSpot)
	bypassQuestLink(t, ctx, c, npcDarin)
	_, before = until[testclient.NpcHtmlMessage](t, ctx, c, nil)
	if snd := testclient.FindAll[testclient.PlaySound](before); len(snd) == 0 || snd[len(snd)-1].Sound != testclient.SoundQuestFinish {
		t.Errorf("finish sounds %+v", snd)
	}
	checkQuests(t, ctx, c, nil)

	if err := c.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	chars, err := s.chars.ListByAccount(ctx, account)
	if err != nil || len(chars) != 1 {
		t.Fatalf("characters: %v", err)
	}
	ch := chars[0]
	var necklace, letters int64
	for _, it := range ch.Items {
		switch it.ItemID {
		case itemNeckla:
			necklace += it.Count
		case itemLetter:
			letters += it.Count
		}
	}
	if necklace != 1 || letters != 0 {
		t.Errorf("stored necklaces %d letters %d", necklace, letters)
	}
	// A new character also starts the tutorial (Q00255), so look Q001 up.
	i := slices.IndexFunc(ch.Quests, func(q storage.Quest) bool { return q.Name == q001 })
	if i < 0 || ch.Quests[i].Vars["<state>"] != "Completed" {
		t.Errorf("stored quests %+v", ch.Quests)
	}
}

// gameLogin logs account in on the login server and the game server and
// returns the game connection at the character list.
func gameLogin(t *testing.T, ctx context.Context, s *stack, account, password string) *testclient.GameConn {
	t.Helper()
	keys, err := testclient.LoginAndSelect(ctx, s.loginAddr, account, password, serverID)
	if err != nil {
		t.Fatalf("logging in as %s: %v", account, err)
	}
	c, err := testclient.DialGame(ctx, s.gameAddr, testclient.InterludeProtocol)
	if err != nil {
		t.Fatalf("DialGame: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if _, err := c.AuthLogin(ctx, account, keys); err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
	return c
}

// teleport moves the GM to pos with //teleport.
func teleport(t *testing.T, ctx context.Context, c *testclient.GameConn, pos testclient.Position) {
	t.Helper()
	if err := c.GMCommand(ctx, fmt.Sprintf("teleport %d %d %d", pos.X, pos.Y, pos.Z)); err != nil {
		t.Fatalf("GMCommand: %v", err)
	}
	if _, err := c.FinishTeleport(ctx); err != nil {
		t.Fatalf("FinishTeleport: %v", err)
	}
}

// questLink talks to the NPC npcID and follows its quest link ("Script"),
// returning the page Q001 answers with. Where several quests talk at the
// NPC (Roxxy also starts Q00006_StepIntoTheFuture), the link opens Mobius's
// choose window, and Q001 is picked from it as the client does.
func questLink(t *testing.T, ctx context.Context, c *testclient.GameConn, npcID int32) testclient.NpcHtmlMessage {
	t.Helper()
	bypassQuestLink(t, ctx, c, npcID)
	page := await[testclient.NpcHtmlMessage](t, ctx, c, nil)
	for _, l := range pageLinks(page.HTML) {
		if strings.HasSuffix(l.bypass, "_Script "+q001) {
			return follow(t, ctx, c, l.bypass)
		}
	}
	return page
}

// bypassQuestLink talks to the NPC npcID and clicks its quest link.
func bypassQuestLink(t *testing.T, ctx context.Context, c *testclient.GameConn, npcID int32) {
	t.Helper()
	npc := findNpc(t, ctx, c, npcID)
	if _, err := c.Target(ctx, npc.ObjectID); err != nil {
		t.Fatalf("Target: %v", err)
	}
	page, err := c.Talk(ctx, npc.ObjectID)
	if err != nil {
		t.Fatalf("Talk: %v", err)
	}
	bypass(t, ctx, c, requireLink(t, page, "Script"))
}

// checkQuests asks for the quest log and compares it.
func checkQuests(t *testing.T, ctx context.Context, c *testclient.GameConn, want []testclient.Quest) {
	t.Helper()
	list, err := c.Quests(ctx)
	if err != nil {
		t.Fatalf("Quests: %v", err)
	}
	if !slices.Equal(list.Quests, want) {
		t.Errorf("quest log %+v, want %+v", list.Quests, want)
	}
}
