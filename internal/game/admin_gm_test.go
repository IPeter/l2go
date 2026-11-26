package game

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// adminGMPages are short stand-ins for the admin pages the GM menu, the
// Effects page and the skill pages fill.
var adminGMPages = map[string]string{
	"admin/gm_menu.htm":             `<html><title>GM Menu</title><body><button value="Diet" action="bypass -h admin_diet"></body></html>`,
	"admin/effects_menu.htm":        `<html><title>Effects Panel</title><body></body></html>`,
	"admin/enchant.htm":             `<html><title>Enchant</title><body></body></html>`,
	"admin/charskills.htm":          `<html><title>Skills Editor</title><body>%name% %class% %level%</body></html>`,
	"admin/charskills_add.htm":      `<html><title>Skills</title><body>%racelist%</body></html>`,
	"admin/charskills_race.htm":     `<html><title>%racename% Classes</title><body>%classlist%</body></html>`,
	"admin/charskills_class.htm":    `<html><title>%classname% Skills</title><body>%skilllist%|%pages%</body></html>`,
	"admin/charskills_add_list.htm": `<html><title>Add Skill (%skillid%)</title><body>%classId%|%skilllist%|%pages%</body></html>`,
	"admin/getbuff_effects.htm":     `<html><title>Effects of %targetName%</title><body>%buffs%|%effectSize%</body></html>`,
}

// adminGMEnv starts a server with the skill data and the pages, and enters
// a GM, "Master", and a player, "Pupil" (a Human Mystic), at the same spot.
func adminGMEnv(t *testing.T) (e *testEnv, gs, ps *fakeSession, gm, pl *Player) {
	t.Helper()
	e = newLearnEnv(t)
	var dir string
	e.onLoop(func() { dir = e.srv.settings.HTMLDir })
	for name, content := range adminGMPages {
		writeDialogFile(t, filepath.Join(dir, filepath.FromSlash(name)), content)
	}
	gs, gm = e.enterLearner(learner{account: "gm", name: "Master", class: classHumanFighter, level: 20, access: storage.AccessLevelAdmin})
	ps, pl = e.enterLearner(learner{account: "acc", name: "Pupil", class: classHumanMystic, level: 10})
	gs.take(t)
	ps.take(t)
	return e, gs, ps, gm, pl
}

// adminRun runs a GM command line for p and returns what p got.
func adminRun(t *testing.T, e *testEnv, sess *fakeSession, p *Player, line string) []testclient.Packet {
	t.Helper()
	sess.take(t)
	e.onLoop(func() { e.srv.RunGMCommand(p, line) })
	return sess.take(t)
}

// sysSays returns the texts of Mobius sendSysMessage (general chat from
// "SYS") in ps.
func sysSays(ps []testclient.Packet) []string {
	var out []string
	for _, s := range only[testclient.CreatureSay](ps) {
		if s.Name == "SYS" {
			out = append(out, s.Text)
		}
	}
	return out
}

// adminTake lets the tick end (the coalesced UserInfo goes out) and returns
// what sess got.
func adminTake(t *testing.T, e *testEnv, sess *fakeSession) []testclient.Packet {
	t.Helper()
	e.advance(200 * time.Millisecond)
	return sess.take(t)
}

// gmLastUserInfo returns the last UserInfo of ps.
func gmLastUserInfo(t *testing.T, what string, ps []testclient.Packet) testclient.UserInfo {
	t.Helper()
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatalf("%s: no UserInfo in %q", what, testclient.Names(ps))
	}
	return uis[len(uis)-1]
}

func TestAdminGMCommandsAreRegistered(t *testing.T) {
	e := newEnv(t, nil)
	names := e.srv.GMCommands()
	for _, want := range []string{
		"gmliston", "gmlistoff", "silence", "diet", "tradeoff", "invul", "setinvul", "superhaste", "gmchat", "gmchat_menu",
		"enchant", "setew", "para", "unpara", "para_all", "unpara_all_menu", "bighead", "social_menu", "effect", "ave_abnormal",
		"earthquake", "atmosphere_menu", "play_sounds", "play_sound", "show_skills", "skill_list", "skill_index",
		"viewskilleffects", "ride_wyvern", "debug", "exceptions", "manor", "checkolympiad",
	} {
		if !slices.Contains(names, want) {
			t.Errorf("GM command %q is missing", want)
		}
	}
}

// //invul makes the GM take no hit damage (damage over time still hurts, as
// in Mobius), and //invul again makes it mortal.
func TestAdminInvul(t *testing.T) {
	e, gs, _, gm, pl := adminGMEnv(t)
	if says := sysSays(adminRun(t, e, gs, gm, "invul")); !slices.Equal(says, []string{"You are now invulnerable."}) {
		t.Errorf("//invul: said %q", says)
	}
	var before, afterHit, afterDOT float64
	var invul bool
	// No attacker: a player's hits would take CP first.
	e.onLoop(func() {
		before = gm.CurHP
		gm.ReduceHP(50, nil)
		afterHit = gm.CurHP
		gm.skill.ReduceCurrentHPByDOT(10, nil, nil)
		afterDOT = gm.CurHP
		invul = gm.skill.IsInvul()
	})
	if !invul || afterHit != before || afterDOT != before-10 {
		t.Errorf("invulnerable GM: invul %v, HP %v, after a hit %v, after a DOT %v", invul, before, afterHit, afterDOT)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "invul")); !slices.Equal(says, []string{"You are now mortal."}) {
		t.Errorf("second //invul: said %q", says)
	}
	e.onLoop(func() {
		before = gm.CurHP
		gm.ReduceHP(50, nil)
		afterHit = gm.CurHP
	})
	if afterHit != before-50 {
		t.Errorf("mortal GM: HP %v after a hit of 50 from %v", afterHit, before)
	}
	// //setinvul works on the target player.
	if says := sysSays(adminRun(t, e, gs, gm, "setinvul")); !slices.Equal(says, []string{"You must target a player."}) {
		t.Errorf("//setinvul without a target: said %q", says)
	}
	e.onLoop(func() { gm.Target = pl })
	if says := sysSays(adminRun(t, e, gs, gm, "setinvul")); !slices.Equal(says, []string{"You made Pupil invulnerable."}) {
		t.Errorf("//setinvul: said %q", says)
	}
}

// The GM menu toggles: //diet, //tradeoff, //silence (whispers refused),
// //gmliston and //gmlistoff with /gmlist, //gmchat.
func TestAdminGMMenuToggles(t *testing.T) {
	e, gs, ps, gm, pl := adminGMEnv(t)

	got := adminRun(t, e, gs, gm, "diet")
	adminPage(t, "//diet", got, "GM Menu")
	var diet bool
	e.onLoop(func() { diet = gm.skill.DietMode() })
	if says := sysSays(got); !diet || !slices.Equal(says, []string{"Diet mode on"}) {
		t.Errorf("//diet: diet %v, said %q", diet, says)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "diet on")); !slices.Equal(says, []string{"Diet mode on"}) {
		t.Errorf("//diet on: said %q", says)
	}
	// Like Mobius, "off" only counts as the second word; alone it toggles.
	if says := sysSays(adminRun(t, e, gs, gm, "diet off")); !slices.Equal(says, []string{"Diet mode off"}) {
		t.Errorf("//diet off: said %q", says)
	}
	// The page's button runs it too.
	e.onLoop(func() { e.srv.ShowAdminPage(gm, "gm_menu.htm") })
	if says := sysSays(e.bypass(gs, "admin_diet")); !slices.Equal(says, []string{"Diet mode on"}) {
		t.Errorf("Diet button: said %q", says)
	}

	for _, tc := range []struct{ line, want string }{
		{"tradeoff", "Trade refusal enabled"},
		{"tradeoff", "Trade refusal disabled"},
		{"tradeoff on", "Trade refusal enabled"},
		{"tradeoff off", "Trade refusal disabled"},
	} {
		if says := sysSays(adminRun(t, e, gs, gm, tc.line)); !slices.Equal(says, []string{tc.want}) {
			t.Errorf("//%s: said %q, want %q", tc.line, says, tc.want)
		}
	}

	// //silence: refusal mode, shown in EtcStatusUpdate; a whisper gets
	// "That player is not online.".
	got = adminRun(t, e, gs, gm, "silence")
	if etc := only[testclient.EtcStatusUpdate](got); len(etc) != 1 || !etc[0].ChatBlocked || !slices.Contains(sysMsgs(got), int32(adminSysMsgRefusalMode)) {
		t.Errorf("//silence: %+v, messages %v", etc, sysMsgs(got))
	}
	ps.take(t)
	chatSend(e, ps, serverpackets.ChatWhisper, "hello", "Master")
	if ids := sysMsgs(ps.take(t)); !slices.Equal(ids, []int32{int32(serverpackets.SysMsgPlayerNotOnline)}) {
		t.Errorf("whisper to a silent GM: messages %v", ids)
	}
	if says := only[testclient.CreatureSay](gs.take(t)); len(says) != 0 {
		t.Errorf("the silent GM got %+v", says)
	}
	got = adminRun(t, e, gs, gm, "silence")
	if etc := only[testclient.EtcStatusUpdate](got); len(etc) != 1 || etc[0].ChatBlocked || !slices.Contains(sysMsgs(got), int32(adminSysMsgAcceptanceMode)) {
		t.Errorf("second //silence: %+v, messages %v", etc, sysMsgs(got))
	}

	// /gmlist: hidden by default; a GM sees itself as "(invis)".
	if ids := sysMsgs(e.request(ps, opRequestGmList, nil)); !slices.Equal(ids, []int32{int32(adminSysMsgNoGMsVisible)}) {
		t.Errorf("/gmlist with a hidden GM: %v", ids)
	}
	got = e.request(gs, opRequestGmList, nil)
	if sms := only[testclient.SystemMessage](got); len(sms) != 2 || sms[1].ID != int32(adminSysMsgGMS1) || !slices.Equal(sms[1].Texts(), []string{"Master (invis)"}) {
		t.Errorf("/gmlist of the GM: %+v", sms)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "gmliston")); !slices.Equal(says, []string{"Registered into GM list."}) {
		t.Errorf("//gmliston: said %q", says)
	}
	got = e.request(ps, opRequestGmList, nil)
	if sms := only[testclient.SystemMessage](got); len(sms) != 2 || sms[0].ID != int32(adminSysMsgGMList) || !slices.Equal(sms[1].Texts(), []string{"Master"}) {
		t.Errorf("/gmlist with a listed GM: %+v", sms)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "gmlistoff")); !slices.Equal(says, []string{"Removed from GM list."}) {
		t.Errorf("//gmlistoff: said %q", says)
	}

	// //gmchat reaches the GMs only.
	ps.take(t)
	got = adminRun(t, e, gs, gm, "gmchat hello GMs")
	if says := only[testclient.CreatureSay](got); len(says) != 1 || says[0].Type != testclient.ChatAlliance || says[0].Name != "Master" || says[0].Text != "hello GMs" {
		t.Errorf("//gmchat: %+v", says)
	}
	if says := only[testclient.CreatureSay](ps.take(t)); len(says) != 0 {
		t.Errorf("a player got the GM chat %+v", says)
	}
	_ = pl

	// Buttons of systems that do not exist yet.
	if msgs := messages(adminRun(t, e, gs, gm, "ride_strider")); !slices.Equal(msgs, []string{"Mounts are not available yet."}) {
		t.Errorf("//ride_strider: %q", msgs)
	}
}

// //superhaste 1-4 gives the GM Super Haste of that level, 0 takes it away.
func TestAdminSuperHaste(t *testing.T) {
	e, gs, _, gm, _ := adminGMEnv(t)
	level := func() int {
		var lv int
		e.onLoop(func() {
			if b := gm.Effects().BySkillID(adminSuperHaste); b != nil {
				lv = b.Skill().Level
			}
		})
		return lv
	}
	adminRun(t, e, gs, gm, "superhaste 2")
	if lv := level(); lv != 2 {
		t.Errorf("//superhaste 2: level %d", lv)
	}
	adminRun(t, e, gs, gm, "superhaste 4 60")
	if lv := level(); lv != 4 {
		t.Errorf("//superhaste 4 60: level %d", lv)
	}
	adminRun(t, e, gs, gm, "superhaste 0")
	if lv := level(); lv != 0 {
		t.Errorf("//superhaste 0: level %d", lv)
	}
	if msgs := messages(adminRun(t, e, gs, gm, "superhaste x")); !slices.Equal(msgs, []string{"Usage: //superhaste <Effect level (0-4)> <Time in seconds>"}) {
		t.Errorf("//superhaste x: %q", msgs)
	}
}

// //para paralyzes the target: the look, no walking, no attacking; //unpara
// frees it. //ave_abnormal and //bighead switch visual effects.
func TestAdminParalyzeAndVisuals(t *testing.T) {
	e, gs, ps, gm, pl := adminGMEnv(t)
	e.onLoop(func() { gm.Target = pl })
	ps.take(t)
	adminRun(t, e, gs, gm, "para")
	if ui := gmLastUserInfo(t, "//para", adminTake(t, e, ps)); uint32(ui.AbnormalEffects)&data.VisualParalyze.Mask() == 0 {
		t.Errorf("//para: abnormal mask %#x", ui.AbnormalEffects)
	}
	pos := pl.Position()
	got := e.request(ps, clientpackets.OpMoveToLocation, walkRequest(eastOf(pos, 300), pos, false))
	if len(only[testclient.ActionFailed](got)) != 1 || len(only[testclient.MoveToLocation](got)) != 0 {
		t.Errorf("a paralyzed player walks: %q", testclient.Names(got))
	}
	var out AttackOutcome
	e.onLoop(func() { out = e.srv.DoAttack(pl, gm) })
	if out != AttackNone {
		t.Errorf("a paralyzed player attacks: %v", out)
	}
	adminRun(t, e, gs, gm, "unpara_menu")
	var paralyzed bool
	e.onLoop(func() { paralyzed = adminParalyzed(pl) })
	if ui := gmLastUserInfo(t, "//unpara", adminTake(t, e, ps)); paralyzed || ui.AbnormalEffects != 0 {
		t.Errorf("//unpara: paralyzed %v, mask %#x", paralyzed, ui.AbnormalEffects)
	}

	got = adminRun(t, e, gs, gm, "ave_abnormal BIG_HEAD")
	if msgs := messages(got); !slices.Equal(msgs, []string{"Pupil affected by BIG_HEAD abnormal visual effect."}) {
		t.Errorf("//ave_abnormal: %q", msgs)
	}
	if ui := gmLastUserInfo(t, "//ave_abnormal", adminTake(t, e, ps)); uint32(ui.AbnormalEffects) != data.VisualBigHead.Mask() {
		t.Errorf("//ave_abnormal BIG_HEAD: mask %#x", ui.AbnormalEffects)
	}
	adminRun(t, e, gs, gm, "shrinkhead")
	if ui := gmLastUserInfo(t, "//shrinkhead", adminTake(t, e, ps)); ui.AbnormalEffects != 0 {
		t.Errorf("//shrinkhead: mask %#x", ui.AbnormalEffects)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "ave_abnormal")); !slices.Equal(says, []string{"Usage: //ave_abnormal <AbnormalVisualEffect> [radius]"}) {
		t.Errorf("//ave_abnormal without an effect: said %q", says)
	}
}

// //social, //effect and //play_sound broadcast their packets.
func TestAdminSocialEffectSound(t *testing.T) {
	e, gs, ps, gm, pl := adminGMEnv(t)
	ps.take(t)
	got := adminRun(t, e, gs, gm, "social 3")
	if sa := only[testclient.SocialAction](got); len(sa) != 1 || sa[0].ObjectID != gm.ObjectID() || sa[0].ActionID != 3 {
		t.Errorf("//social 3: %+v", sa)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Master was affected by your request."}) {
		t.Errorf("//social 3: %q", msgs)
	}
	if sa := only[testclient.SocialAction](ps.take(t)); len(sa) != 1 {
		t.Errorf("the player saw %+v", sa)
	}
	if ids := sysMsgs(adminRun(t, e, gs, gm, "social 1")); !slices.Equal(ids, []int32{int32(adminSysMsgNothingHappened), int32(adminSysMsgNothingHappened)}) {
		t.Errorf("//social 1 on a player: %v", ids)
	}

	e.onLoop(func() { gm.Target = pl })
	got = adminRun(t, e, gs, gm, "effect 1177 2")
	if msu := only[testclient.MagicSkillUse](got); len(msu) != 1 || msu[0].CasterID != pl.ObjectID() || msu[0].TargetID != gm.ObjectID() || msu[0].SkillID != 1177 || msu[0].Level != 2 {
		t.Errorf("//effect: %+v", msu)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Pupil performs MSU 1177/2 by your request."}) {
		t.Errorf("//effect: %q", msgs)
	}

	got = adminRun(t, e, gs, gm, "play_sound ItemSound.quest_accept")
	if snd := only[testclient.PlaySound](got); len(snd) != 2 || snd[0].Sound != "ItemSound.quest_accept" || snd[0].Type != 1 {
		t.Errorf("//play_sound: %+v", snd)
	}
	if says := sysSays(got); !slices.Equal(says, []string{"Playing ItemSound.quest_accept."}) {
		t.Errorf("//play_sound: said %q", says)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "atmosphere sky")); !slices.Equal(says, []string{"Usage: //atmosphere <sky day|night|red> <duration>"}) {
		t.Errorf("//atmosphere sky: said %q", says)
	}
	ps.take(t)
	adminRun(t, e, gs, gm, "atmosphere sky night 0")
	if names := testclient.Names(ps.take(t)); len(names) != 1 {
		t.Errorf("//atmosphere sky night: the player got %q", names)
	}
}

// The skill pages: races, classes, a class's skills, a skill's levels; the
// level's link adds it.
func TestAdminSkillPages(t *testing.T) {
	e, gs, _, gm, pl := adminGMEnv(t)
	html := func(line string) string {
		t.Helper()
		return onePage(t, adminRun(t, e, gs, gm, line)).HTML
	}
	if h := html("skill_list"); !strings.Contains(h, `bypass admin_skill_race HUMAN">Human</a>`) || !strings.Contains(h, `admin_skill_race DARK_ELF">Dark Elf</a>`) {
		t.Errorf("//skill_list: %q", h)
	}
	if h := html("skill_race HUMAN"); !strings.Contains(h, "<title>Human Classes</title>") || !strings.Contains(h, "bypass admin_skill_class MAGE") {
		t.Errorf("//skill_race HUMAN: %q", h)
	}
	h := html("skill_class MAGE")
	if !strings.Contains(h, "bypass admin_skill_class MAGE 1") || !strings.Contains(h, "<td>1</td>") {
		t.Errorf("//skill_class MAGE: no pager in %q", h)
	}
	page := 0
	for !strings.Contains(h, "bypass admin_skill_add_list MAGE 1177") && page < 10 {
		page++
		h = html("skill_class MAGE " + strconv.Itoa(page))
	}
	if !strings.Contains(h, "bypass admin_skill_add_list MAGE 1177") {
		t.Errorf("//skill_class MAGE: no Wind Strike on the first pages")
	}
	if h := html("skill_add_list MAGE 1177"); !strings.Contains(h, "bypass admin_add_skill 1177 2") || !strings.Contains(h, "Add Skill (1177)") {
		t.Errorf("//skill_add_list MAGE 1177: %q", h)
	}
	e.onLoop(func() { gm.Target = pl })
	if h := html("show_skills"); !strings.Contains(h, "Pupil") {
		t.Errorf("//show_skills: %q", h)
	}
	adminRun(t, e, gs, gm, "add_skill 1177 2")
	var level int
	e.onLoop(func() { level = pl.SkillLevel(skillWindStrike) })
	if level != 2 {
		t.Errorf("Wind Strike %d after the page's link", level)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "remove_all_skills")); !slices.Equal(says, []string{"You have removed all skills from Pupil."}) {
		t.Errorf("//remove_all_skills: said %q", says)
	}
	var left int
	e.onLoop(func() { left = len(pl.Skills()) })
	if left != 0 {
		t.Errorf("%d skills left after //remove_all_skills", left)
	}
	if ids := sysMsgs(adminRun(t, e, gs, gm, "give_clan_skills")); !slices.Equal(ids, []int32{int32(adminSysMsgMustBeClanMember)}) {
		t.Errorf("//give_clan_skills: %v", ids)
	}

	// The effects of a buff.
	adminRun(t, e, gs, gm, "buff 1068 3")
	h = html("viewskilleffects " + strconv.Itoa(int(pl.ObjectID())) + " 1068 3")
	if !strings.Contains(h, "Effects of Pupil") || !strings.Contains(h, "Might Lv. 3 (") || strings.HasSuffix(h, "|0</body></html>") {
		t.Errorf("//viewskilleffects: %q", h)
	}
	if says := sysSays(adminRun(t, e, gs, gm, "viewskilleffects 1")); !slices.Contains(says, "Failed viewing skill effects: null") {
		t.Errorf("//viewskilleffects 1: said %q", says)
	}
}

// //setew sets the enchant of the worn weapon: the item, its P. Atk, the
// InventoryUpdate and the UserInfo follow.
func TestAdminEnchant(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin})
	sword := e.give(gm, itemLongSword, 1)
	e.onLoop(func() { e.srv.useEquippableItem(gm, sword) })
	e.advance(time.Second)
	sess.take(t)
	var before, after int32
	e.onLoop(func() { before = gm.stats.View().PAtk })
	got := adminRun(t, e.testEnv, sess, gm, "setew 7")
	e.advance(inventoryUpdateDelay)
	got = append(got, sess.take(t)...)
	var enchant int32
	e.onLoop(func() {
		after = gm.stats.View().PAtk
		if w := gm.inv.Paperdoll(storage.PaperdollRHand); w != nil {
			enchant = w.Enchant()
		}
	})
	if enchant != 7 || after <= before {
		t.Errorf("//setew 7: enchant %d, P. Atk %d from %d", enchant, after, before)
	}
	enchanted := func(c testclient.ItemChange) bool {
		return c.ItemID == itemLongSword && c.Enchant == 7 && c.Equipped
	}
	iu := only[testclient.InventoryUpdate](got)
	if len(iu) != 1 || !slices.ContainsFunc(iu[0].Items, enchanted) {
		t.Errorf("//setew 7: InventoryUpdate %+v", iu)
	}
	if says := sysSays(got); !slices.Equal(says, []string{"Changed enchantment of Gm's Long Sword from 0 to 7."}) {
		t.Errorf("//setew 7: said %q", says)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Admin has changed the enchantment of your Long Sword from 0 to 7."}) {
		t.Errorf("//setew 7: messages %q", msgs)
	}
	gmLastUserInfo(t, "//setew 7", got)
	for line, want := range map[string]string{
		"setew":       "Please specify a new enchant value.",
		"setew x":     "Please specify a valid new enchant value.",
		"setew 70000": "You must set the enchant level to be between 0-65535.",
	} {
		if says := sysSays(adminRun(t, e.testEnv, sess, gm, line)); !slices.Equal(says, []string{want}) {
			t.Errorf("//%s: said %q, want %q", line, says, want)
		}
	}
}

// //diet lifts the weight penalty, and the penalty comes back without it.
func TestAdminDiet(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin})
	var maxLoad, now int64
	e.onLoop(func() { maxLoad, now = int64(p.stats.WeightLimit()), p.inv.Weight() })
	e.give(p, itemMithrilOre, (maxLoad*70/100-now)/2)
	sess.take(t)
	for _, want := range []int32{0, 2} {
		got := adminRun(t, e.testEnv, sess, p, "diet")
		if etc := only[testclient.EtcStatusUpdate](got); len(etc) != 1 || etc[0].WeightPenalty != want {
			t.Errorf("//diet: EtcStatusUpdate %+v, want level %d", etc, want)
		}
	}
}
