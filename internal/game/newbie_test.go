package game

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// newbiePages are the pages of the newbie buffer, as in Mobius
// (html/default), the guide's page cut to its link.
var newbiePages = map[string]string{
	"default/SupportMagic.htm": "<html><body>\nGuide for Novices:<br>\n" +
		"<a action=\"bypass -h npc_%objectId%_SupportMagic\">Receive supplemental magic.</a>\n</body></html>",
	"default/SupportMagicNovice.htm": "<html><body>Newbie Guide:<br>\n" +
		"Only a <font color=\"LEVEL\">novice character of level 25 or below</font> can receive my support magic.<br>\n" +
		"Your novice character is the first one that you created and raised in this world.\n</body></html>",
	"default/SupportMagicHighLevel.htm": "<html><body>\n" +
		"Support magic is only available for characters who are <font color=\"LEVEL\">level 25 or below</font>.\n</body></html>",
	"default/SupportMagicLowLevel.htm": "<html><body>Newbie Guide:<br>\n" +
		"You do not yet require Support Magic. Come see me again when you reach level 8.\n</body></html>",
}

// classDuelist is a 3rd class (the Gladiator's).
const classDuelist = 88

// newNewbieEnv starts a server with the world data of the dialog tests, the
// skill data and the newbie buffer's pages, and spawns a Newbie Guide 50
// east of the human fighter start.
func newNewbieEnv(t *testing.T) (*testEnv, *Npc) {
	t.Helper()
	d := testGameData(t)
	dataDir := t.TempDir()
	for name, content := range newbiePages {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = dataDir
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: testWorldData(t),
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
	return e, e.spawnNpc(npcNewbieGuide, eastOf(tiStart, 50))
}

// enterNewbie stores a character of the class (a starting class, or a
// later one of the human fighter's line) at the level at the human fighter
// start, with the newbie flag as given, enters the world with it and drops
// the packets.
func (e *testEnv) enterNewbie(account string, class int, level int32, newbie bool) (*fakeSession, *Player) {
	e.t.Helper()
	start := class
	if class == classDuelist {
		start = classHumanFighter
	}
	c := e.createCharacter(account, "N"+account, start, storage.AccessLevelPlayer)
	// The flag is stored only at creation: create the character again.
	if err := e.store.Delete(e.ctx, c.ObjectID); err != nil {
		e.t.Fatalf("Delete: %v", err)
	}
	c.ClassID, c.BaseClassID = int32(class), int32(class)
	c.Level, c.Exp = level, e.srv.ExpForLevel(int(level))
	c.X, c.Y, c.Z = tiStart.X, tiStart.Y, tiStart.Z
	c.Newbie = newbie
	if err := e.store.Create(e.ctx, &c); err != nil {
		e.t.Fatalf("Create: %v", err)
	}
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.advance(time.Second)
	sess.take(e.t)
	return sess, p
}

// supportMagic opens the guide's support magic page for p and clicks its
// link; it returns the packets p got from the click.
func (e *testEnv) supportMagic(sess *fakeSession, p *Player, guide *Npc) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() {
		e.srv.sendPage(p, e.srv.pageFile(p, guide.ID(), "default/SupportMagic.htm").replaceObjectID(guide.ID()))
	})
	return e.bypass(sess, "npc_"+strconv.Itoa(int(guide.ID()))+"_SupportMagic")
}

// guideCasts returns the skills of the MagicSkillUse packets of ps the
// guide cast at target, in order.
func guideCasts(ps []testclient.Packet, guide *Npc, target int32) []int32 {
	var ids []int32
	for _, u := range only[testclient.MagicSkillUse](ps) {
		if u.CasterID == int32(guide.ID()) && u.TargetID == target {
			ids = append(ids, u.SkillID)
		}
	}
	return ids
}

// newbieBuffs are the beginner buffs.
var newbieBuffs = []int32{4322, 4323, 4324, 4325, 4326, 4327, 4328, 4329, 4330, 4331}

// buffsOn returns the beginner buffs p has, in newbieBuffs order.
func (e *testEnv) buffsOn(p *Player) []int32 {
	e.t.Helper()
	var ids []int32
	e.onLoop(func() {
		for _, id := range newbieBuffs {
			if p.Effects().IsAffectedBySkill(int(id)) {
				ids = append(ids, id)
			}
		}
	})
	return ids
}

// Every level of 8 to 25 of a newbie mage and fighter gets the buffs of its
// band from the guide (Mobius SupportMagic), cast at once in Mobius's order;
// a player standing by sees each MagicSkillUse.
func TestNewbieBufferBands(t *testing.T) {
	e, guide := newNewbieEnv(t)
	bySess, _ := e.enterNewbie("bystander", classHumanFighter, 40, false)

	// The buffs by level band: Wind Walk, Shield, then the mage's or the
	// fighter's four.
	bands := []struct {
		from, to      int32
		fighter, mage []int32
	}{
		{8, 10, []int32{4322}, []int32{4322}},
		{11, 11, []int32{4322, 4323}, []int32{4322, 4323}},
		{12, 12, []int32{4322, 4323, 4324}, []int32{4322, 4323, 4328}},
		{13, 13, []int32{4322, 4323, 4324, 4325}, []int32{4322, 4323, 4328, 4329}},
		{14, 14, []int32{4322, 4323, 4324, 4325, 4326}, []int32{4322, 4323, 4328, 4329, 4330}},
		{15, 19, []int32{4322, 4323, 4324, 4325, 4326, 4327}, []int32{4322, 4323, 4328, 4329, 4330, 4331}},
		{20, 20, []int32{4322, 4323, 4324, 4325, 4326}, []int32{4322, 4323, 4328, 4329, 4330}},
		{21, 21, []int32{4322, 4323, 4324, 4325}, []int32{4322, 4323, 4328, 4329}},
		{22, 22, []int32{4322, 4323, 4324}, []int32{4322, 4323, 4328}},
		{23, 23, []int32{4322, 4323}, []int32{4322, 4323}},
		{24, 24, []int32{4322}, []int32{4322}},
		{25, 25, nil, nil},
	}
	for _, b := range bands {
		for level := b.from; level <= b.to; level++ {
			for _, c := range []struct {
				name  string
				class int
				want  []int32
			}{
				{"fighter", classHumanFighter, b.fighter},
				{"mage", classHumanMystic, b.mage},
			} {
				sess, p := e.enterNewbie(fmt.Sprintf("%s%d", c.name, level), c.class, level, true)
				bySess.take(t)
				got := e.supportMagic(sess, p, guide)
				me := p.ObjectID()
				if casts := guideCasts(got, guide, me); !slices.Equal(casts, c.want) {
					t.Errorf("level %d %s: the guide cast %v, want %v", level, c.name, casts, c.want)
				}
				if pages := only[testclient.NpcHtmlMessage](got); len(pages) != 0 {
					t.Errorf("level %d %s: got the page %q", level, c.name, pages[0].HTML)
				}
				if seen := guideCasts(bySess.take(t), guide, me); !slices.Equal(seen, c.want) {
					t.Errorf("level %d %s: the bystander saw %v, want %v", level, c.name, seen, c.want)
				}
				want := slices.Clone(c.want)
				slices.Sort(want)
				if buffs := e.buffsOn(p); !slices.Equal(buffs, want) {
					t.Errorf("level %d %s: buffs %v, want %v", level, c.name, buffs, want)
				}
			}
		}
	}
}

// The buffs last an hour: the icons show 60:00.
func TestNewbieBufferIcons(t *testing.T) {
	e, guide := newNewbieEnv(t)
	sess, p := e.enterNewbie("mage", classHumanMystic, 15, true)
	e.supportMagic(sess, p, guide)
	e.advance(time.Second)
	icons := only[testclient.AbnormalStatusUpdate](sess.take(t))
	if len(icons) == 0 {
		t.Fatal("no AbnormalStatusUpdate")
	}
	last := icons[len(icons)-1]
	for _, id := range []int32{4322, 4323, 4328, 4329, 4330, 4331} {
		if !hasEffect(last, id, 3600) && !hasEffect(last, id, 3599) {
			t.Errorf("icons %+v, want skill %d with 3600 s", last, id)
		}
	}
}

// The refusals: a character that is not its account's newbie gets the
// novice page, one above level 25 the high level page, one below level 8 the
// low level page, all of the guide; a 3rd class one gets Mobius's message.
// Nobody gets a buff.
func TestNewbieBufferRefusals(t *testing.T) {
	e, guide := newNewbieEnv(t)
	gid := strconv.Itoa(int(guide.ID()))
	for _, c := range []struct {
		name   string
		class  int
		level  int32
		newbie bool
		page   string // the page, "" for the message
	}{
		{"second", classHumanFighter, 15, false, "default/SupportMagicNovice.htm"},
		{"secondlow", classHumanMystic, 3, false, "default/SupportMagicNovice.htm"},
		{"high", classHumanFighter, 26, true, "default/SupportMagicHighLevel.htm"},
		{"low", classHumanMystic, 7, true, "default/SupportMagicLowLevel.htm"},
		{"third", classDuelist, 20, true, ""},
		{"thirdhigh", classDuelist, 76, true, "default/SupportMagicHighLevel.htm"},
	} {
		sess, p := e.enterNewbie(c.name, c.class, c.level, c.newbie)
		got := e.supportMagic(sess, p, guide)
		if casts := guideCasts(got, guide, p.ObjectID()); len(casts) != 0 {
			t.Errorf("%s: the guide cast %v", c.name, casts)
		}
		if buffs := e.buffsOn(p); len(buffs) != 0 {
			t.Errorf("%s: buffs %v", c.name, buffs)
		}
		if c.page == "" {
			noPage(t, c.name, got)
			var texts []string
			for _, m := range only[testclient.SystemMessage](got) {
				texts = append(texts, m.Texts()...)
			}
			if !slices.Contains(texts, supportMagicThirdClassMessage) {
				t.Errorf("%s: messages %q, want the 3rd class one", c.name, texts)
			}
			continue
		}
		pg := onePage(t, got)
		// The page loader drops the line breaks.
		if want := strings.ReplaceAll(newbiePages[c.page], "\n", ""); pg.HTML != want || pg.ObjectID != int32(guide.ID()) {
			t.Errorf("%s: page of %d %q, want %s of the guide %s", c.name, pg.ObjectID, pg.HTML, c.page, gid)
		}
	}
}

// The newbie flag is the stored one: IsNewbie follows it (Mobius isNewbie),
// and AltNewCharAlwaysIsNewbie makes everyone a newbie.
func TestNewbieFlag(t *testing.T) {
	e, _ := newNewbieEnv(t)
	_, first := e.enterNewbie("first", classHumanFighter, 1, true)
	_, second := e.enterNewbie("second", classHumanFighter, 1, false)
	var a, b, always bool
	e.onLoop(func() {
		a, b = first.IsNewbie(), second.IsNewbie()
		e.srv.quest.settings.NewCharAlwaysNewbie = true
		always = second.IsNewbie()
		e.srv.quest.settings.NewCharAlwaysNewbie = false
	})
	if !a || b || !always {
		t.Errorf("IsNewbie: first %v, second %v, second with AltNewCharAlwaysIsNewbie %v; want true, false, true", a, b, always)
	}
}
