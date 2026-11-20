package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The GM menu (//admin7), the Effects page (//admin3) and the skill pages of
// the admin panel: Mobius AdminAdmin's gmliston, gmlistoff, silence, diet and
// tradeoff, AdminInvul, AdminSuperHaste, AdminGmChat, AdminEnchant,
// AdminEffects, the page commands of AdminSkill and AdminBuffs, and
// RequestGmList (/gmlist). The buttons of systems the server does not have
// yet (mounts, petitions, events, clans, sieges, the Olympiad, ...) answer
// that they are not available yet.
func init() { registerFeature("admin_gm", setupAdminGM) }

const (
	// adminSysMsgNotClanLeader is "$s1 is not a clan leader." (Mobius
	// S1_IS_NOT_A_CLAN_LEADER).
	adminSysMsgNotClanLeader serverpackets.SystemMessageID = 9
	// adminSysMsgNothingHappened is "Nothing happened." (Mobius
	// NOTHING_HAPPENED).
	adminSysMsgNothingHappened serverpackets.SystemMessageID = 61
	// adminSysMsgRefusalMode is "Message refusal mode." (Mobius
	// MESSAGE_REFUSAL_MODE).
	adminSysMsgRefusalMode serverpackets.SystemMessageID = 177
	// adminSysMsgAcceptanceMode is "Message acceptance mode." (Mobius
	// MESSAGE_ACCEPTANCE_MODE).
	adminSysMsgAcceptanceMode serverpackets.SystemMessageID = 178
	// adminSysMsgMustBeClanMember is "The target must be a clan member."
	// (Mobius THE_TARGET_MUST_BE_A_CLAN_MEMBER).
	adminSysMsgMustBeClanMember serverpackets.SystemMessageID = 234
	// adminSysMsgNoGMsVisible is "There are no GMs currently visible in the
	// public list ..." (Mobius THERE_ARE_NO_GMS_CURRENTLY_VISIBLE_...).
	adminSysMsgNoGMsVisible serverpackets.SystemMessageID = 702
	// adminSysMsgGMList is "======<GM List>======" (Mobius GM_LIST).
	adminSysMsgGMList serverpackets.SystemMessageID = 703
	// adminSysMsgGMS1 is "GM : $s1" (Mobius GM_S1).
	adminSysMsgGMS1 serverpackets.SystemMessageID = 704

	// opRequestGmList is the opcode of RequestGmList, the client's /gmlist.
	opRequestGmList = 0x81
	// adminSuperHaste is the skill of //superhaste (Mobius SUPER_HASTE_ID).
	adminSuperHaste = 7029
	// adminPageLimit is the number of rows on a page of the skill pages
	// (Mobius AdminSkill and AdminBuffs PAGE_LIMIT).
	adminPageLimit = 7
	// adminFontRed and adminFontEnd mark an effect not in use (Mobius
	// AdminBuffs FONT_RED1 and FONT_RED2).
	adminFontRed = `<font color="FF0000">`
	adminFontEnd = `</font>`
)

// Server packets of the Effects page, which nothing else sends (Mobius
// ServerPackets).
const (
	opSunRise      = 0x1c
	opSunSet       = 0x1d
	opEarthquake   = 0xc4
	opEventTrigger = 0xcf
	opExServer     = 0xfe
	opExRedSky     = 0x40
)

// playerAdmin is the GM state of a player (Player.admin): the toggles of
// the GM menu, which last until the player leaves.
type playerAdmin struct {
	invul        bool // //invul, Mobius _isInvul
	diet         bool // //diet: no weight penalty
	silence      bool // //silence: whispers refused
	tradeRefusal bool // //tradeoff: trade requests refused (when trade exists)
	gmListed     bool // //gmliston: shown in /gmlist (Mobius GMStartupAutoList is off)
	// taken are the GM's own skills while it has another player's
	// (//get_skills), given back by //reset_skills.
	taken []*skills.Skill
}

// creatureAdmin is the GM state of a creature (skillBase.admin): the
// paralysis of //para and the abnormal visual effects GMs started.
type creatureAdmin struct {
	paralyzed bool   // Mobius _isParalyzed
	aves      uint64 // the started effects, a bit per AbnormalVisualEffect
}

// visual returns the normal visual mask of the started effects; the special
// and event ones are not shown by Interlude.
func (c *creatureAdmin) visual() uint32 {
	var m uint32
	for ave := range data.AbnormalVisualEffect(data.NumAbnormalVisualEffects) {
		if c.aves&(1<<ave) != 0 && !ave.IsSpecial() && !ave.IsEvent() {
			m |= ave.Mask()
		}
	}
	return m
}

func (a *skillPlayer) adminVisual() uint32 {
	if a == nil {
		return 0
	}
	return a.admin.visual()
}

func (a *skillNpc) adminVisual() uint32 {
	if a == nil {
		return 0
	}
	return a.admin.visual()
}

// adminParalyzed reports a creature paralyzed by //para.
func adminParalyzed(c Creature) bool {
	switch v := c.(type) {
	case *Player:
		return v != nil && v.skill != nil && v.skill.admin.paralyzed
	case *Npc:
		return v != nil && v.skill != nil && v.skill.admin.paralyzed
	}
	return false
}

// adminNotAvailable are the buttons of the admin pages whose systems do not
// exist yet, with what they answer.
var adminNotAvailable = map[string]string{
	// GM menu.
	"ride_strider":   "Mounts are not available yet.",
	"ride_wyvern":    "Mounts are not available yet.",
	"unride":         "Mounts are not available yet.",
	"unride_strider": "Mounts are not available yet.",
	"unride_wyvern":  "Mounts are not available yet.",
	"view_petitions": "Petitions are not available yet.",
	"event_menu":     "Events are not available yet.",
	"debug":          "Debugging is not available yet.",
	"exceptions":     "The exception list is not available yet.",
	"snoop":          "Snooping is not available yet.",
	// Effects page.
	"mobmenu": "Mob groups are not available yet.",
	// Game page.
	"siege":          "Sieges are not available yet.",
	"cw_info_menu":   "Cursed weapons are not available yet.",
	"cw_add":         "Cursed weapons are not available yet.",
	"cw_remove":      "Cursed weapons are not available yet.",
	"mammon_find":    "The Seven Signs are not available yet.",
	"mammon_respawn": "The Seven Signs are not available yet.",
	"manor":          "The manor is not available yet.",
	"checkolympiad":  "The Olympiad is not available yet.",
	"saveolymp":      "The Olympiad is not available yet.",
	"givehero":       "The Olympiad is not available yet.",
}

// adminEnchantSlots are the commands of the enchant page (Mobius
// AdminEnchant) and the paperdoll slot each sets.
var adminEnchantSlots = map[string]storage.PaperdollSlot{
	"seteh": storage.PaperdollHead,
	"setec": storage.PaperdollChest,
	"seteg": storage.PaperdollGloves,
	"seteb": storage.PaperdollFeet,
	"setel": storage.PaperdollLegs,
	"setew": storage.PaperdollRHand,
	"setes": storage.PaperdollLHand,
	"setle": storage.PaperdollLEar,
	"setre": storage.PaperdollREar,
	"setlf": storage.PaperdollLFinger,
	"setrf": storage.PaperdollRFinger,
	"seten": storage.PaperdollNeck,
	"setun": storage.PaperdollUnder,
	"setba": storage.PaperdollCloak,
}

// setupAdminGM registers the commands and RequestGmList.
func setupAdminGM(s *Server) error {
	errs := []error{s.Register(opRequestGmList, func(p *Player, _ []byte) { s.SendGMList(p) })}
	add := func(name string, fn GMCommand) {
		errs = append(errs, s.RegisterGMCommand(name, storage.AccessLevelAdmin, fn))
	}

	// The GM menu (AdminAdmin, AdminInvul, AdminSuperHaste, AdminGmChat).
	add("gmliston", s.gmListOn)
	add("gmlistoff", s.gmListOff)
	add("silence", s.gmSilence)
	add("diet", s.gmDiet)
	add("tradeoff", s.gmTradeOff)
	add("invul", s.gmInvul)
	add("setinvul", s.gmSetInvul)
	add("superhaste", s.gmSuperHaste)
	add("speed", s.gmSuperHaste)
	add("superhaste_menu", s.gmAdminMenuPage)
	add("speed_menu", s.gmAdminMenuPage)
	add("gmchat", s.gmChat(false))
	add("gmchat_menu", s.gmChat(true))
	add("enchant", s.gmEnchantPage)
	for name, slot := range adminEnchantSlots {
		add(name, s.gmSetEnchant(slot))
	}

	// The Effects page (AdminEffects).
	for _, name := range []string{"para", "para_menu", "unpara", "unpara_menu"} {
		add(name, s.gmPara(name, !strings.HasPrefix(name, "un")))
	}
	for _, name := range []string{"para_all", "para_all_menu", "unpara_all", "unpara_all_menu"} {
		add(name, s.gmParaAll(name, !strings.HasPrefix(name, "un")))
	}
	add("bighead", s.gmBigHead(true))
	add("shrinkhead", s.gmBigHead(false))
	add("social", s.gmSocial(false))
	add("social_menu", s.gmSocial(true))
	add("effect", s.gmEffect(false))
	add("effect_menu", s.gmEffect(true))
	for _, name := range []string{"ave_abnormal", "ave_special", "ave_event"} {
		add(name, s.gmAbnormal(name))
	}
	add("earthquake", s.gmEarthquake(false))
	add("earthquake_menu", s.gmEarthquake(true))
	add("atmosphere", s.gmAtmosphere(false))
	add("atmosphere_menu", s.gmAtmosphere(true))
	add("play_sounds", s.gmPlaySounds)
	add("play_sound", s.gmPlaySound)
	add("event_trigger", s.gmEventTrigger)

	// The skill pages (AdminSkill).
	add("show_skills", s.gmShowSkills)
	add("remove_skills", s.gmRemoveSkills)
	add("skill_list", s.gmSkillList)
	add("skill_race", s.gmSkillRace)
	add("skill_class", s.gmSkillClass)
	add("skill_add_list", s.gmSkillAddList)
	add("skill_index", s.gmSkillIndex)
	add("get_skills", s.gmGetSkills)
	add("reset_skills", s.gmResetSkills)
	add("give_all_skills_fs", s.gmGiveAllFS)
	add("remove_all_skills", s.gmRemoveAllSkills)
	add("give_clan_skills", s.gmGiveClanSkills)
	add("give_all_clan_skills", s.gmGiveClanSkills)
	add("add_clan_skill", s.gmAddClanSkill)
	add("setskill", s.gmSetSkill)

	// The buff pages (AdminBuffs).
	add("viewskilleffects", s.gmViewSkillEffects(false))
	add("viewskilleffects_ps", s.gmViewSkillEffects(true))
	add("viewblockedeffects", s.gmViewBlockedEffects)
	add("areacancel", s.gmAreaCancel)
	add("switch_gm_buffs", func(p *Player, _ []string) { gmSysMessage(p, "There is nothing to switch.") })

	for name, answer := range adminNotAvailable {
		add(name, func(p *Player, _ []string) { p.SendMessage(answer) })
	}
	return errors.Join(errs...)
}

// gmAdminMenuPage shows the GM menu (Mobius //superhaste_menu).
func (s *Server) gmAdminMenuPage(p *Player, _ []string) { s.ShowAdminPage(p, "gm_menu.htm") }

// The GM menu.

// gmListOn is //gmliston (Mobius AdminAdmin): the GM shows in /gmlist.
func (s *Server) gmListOn(p *Player, _ []string) {
	p.admin.gmListed = true
	gmSysMessage(p, "Registered into GM list.")
	s.ShowAdminPage(p, "gm_menu.htm")
}

// gmListOff is //gmlistoff: the GM shows in /gmlist to GMs only, as
// "(invis)".
func (s *Server) gmListOff(p *Player, _ []string) {
	p.admin.gmListed = false
	gmSysMessage(p, "Removed from GM list.")
	s.ShowAdminPage(p, "gm_menu.htm")
}

// SendGMList answers /gmlist (Mobius RequestGmList and
// AdminData.sendListToPlayer): the listed GMs, and to a GM the hidden ones
// too with "(invis)"; or that no GM is visible. Loop-only.
func (s *Server) SendGMList(p *Player) {
	var names []string
	for v := range s.Players() {
		switch {
		case !v.IsGM() || !v.InGame():
		case v.admin.gmListed:
			names = append(names, v.Name())
		case p.IsGM():
			names = append(names, v.Name()+" (invis)")
		}
	}
	if len(names) == 0 {
		p.SendSystemMessage(adminSysMsgNoGMsVisible)
		return
	}
	p.SendSystemMessage(adminSysMsgGMList)
	for _, name := range names {
		p.SendSystemMessage(adminSysMsgGMS1, serverpackets.TextParam(name))
	}
}

// gmSilence is //silence (Mobius AdminAdmin): message refusal mode on or
// off, shown in EtcStatusUpdate; whispers to the GM then answer "That player
// is not online.".
func (s *Server) gmSilence(p *Player, _ []string) {
	p.admin.silence = !p.admin.silence
	p.Send(etcStatusPacket(p)) // Mobius setSilenceMode
	if p.admin.silence {
		p.SendSystemMessage(adminSysMsgRefusalMode)
	} else {
		p.SendSystemMessage(adminSysMsgAcceptanceMode)
	}
	s.ShowAdminPage(p, "gm_menu.htm")
}

// gmDiet is //diet [on|off] (Mobius AdminAdmin): no weight penalty while it
// is on. Like Mobius, "on" as the first word switches it on, "off" only as
// the second; otherwise it toggles. The weight penalty is computed again.
func (s *Server) gmDiet(p *Player, args []string) {
	set := func(on bool) {
		p.admin.diet = on
		if on {
			gmSysMessage(p, "Diet mode on")
		} else {
			gmSysMessage(p, "Diet mode off")
		}
	}
	switch {
	case len(args) > 0 && strings.EqualFold(args[0], "on"):
		set(true)
	case len(args) > 1 && strings.EqualFold(args[1], "off"):
		set(false)
	case len(args) < 2: // Mobius's NoSuchElementException: a toggle
		set(!p.admin.diet)
	}
	s.refreshOverloaded(p)
	s.ShowAdminPage(p, "gm_menu.htm")
}

// gmTradeOff is //tradeoff [on|off] (Mobius AdminAdmin): trade requests are
// refused while it is on. Trade does not exist yet; the flag is kept.
func (s *Server) gmTradeOff(p *Player, args []string) {
	set := func(on bool) {
		p.admin.tradeRefusal = on
		if on {
			gmSysMessage(p, "Trade refusal enabled")
		} else {
			gmSysMessage(p, "Trade refusal disabled")
		}
	}
	mode := strings.Join(args, " ")
	switch {
	case len(args) == 0:
		set(!p.admin.tradeRefusal)
	case strings.EqualFold(mode, "on"):
		set(true)
	case strings.EqualFold(mode, "off"):
		set(false)
	}
	s.ShowAdminPage(p, "gm_menu.htm")
}

// TradeRefusal reports whether the player refuses trade requests (//tradeoff,
// Mobius getTradeRefusal). Loop-only.
func (p *Player) TradeRefusal() bool { return p.admin.tradeRefusal }

// gmInvul is //invul (Mobius AdminInvul): the GM becomes invulnerable or
// mortal again. An invulnerable player takes no damage (damage over time
// aside), negative skills do not land on it and monsters forget it.
func (s *Server) gmInvul(p *Player, _ []string) { s.toggleInvul(p, p) }

// gmSetInvul is //setinvul: the same for the GM's target player.
func (s *Server) gmSetInvul(p *Player, _ []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		gmSysMessage(p, "You must target a player.")
		return
	}
	s.toggleInvul(t, p)
}

// toggleInvul is AdminInvul.toggleInvul.
func (s *Server) toggleInvul(t, gm *Player) {
	invul := !t.skill.IsInvul()
	t.admin.invul = invul
	word := "mortal."
	if invul {
		word = "invulnerable."
	}
	gmSysMessage(t, "You are now "+word)
	if gm != t {
		gmSysMessage(gm, "You made "+t.Name()+" "+word)
	}
}

// gmSuperHaste is //superhaste <0-4> [seconds] and //speed (Mobius
// AdminSuperHaste): the GM's Super Haste (skill 7029) ends, and levels 1 to
// 4 start that level, for the given time or the skill's own.
func (s *Server) gmSuperHaste(p *Player, args []string) {
	usage := func() { p.SendMessage("Usage: //superhaste <Effect level (0-4)> <Time in seconds>") }
	if len(args) == 0 {
		usage()
		return
	}
	val, ok := javaParseInt(args[0])
	if !ok {
		usage()
		return
	}
	ft := skills.FinishNormal
	if val == 0 && p.Effects().IsAffectedBySkill(adminSuperHaste) {
		ft = skills.FinishRemoved
	}
	p.Effects().StopSkill(ft, adminSuperHaste)
	if val < 1 || val > 4 {
		return
	}
	seconds := 0
	if len(args) > 1 {
		if seconds, ok = javaParseInt(args[1]); !ok {
			usage()
			return
		}
	}
	sk, found := s.Skill(adminSuperHaste, val)
	if !found {
		usage()
		return
	}
	c := s.skillCreatureOf(p)
	sk.ApplyEffects(s.skills.env, c, c, false, false, true, seconds)
}

// gmChat is //gmchat <text> and //gmchat_menu <text> (Mobius AdminGmChat):
// the text goes to every GM in the alliance channel with the GM's name; the
// menu version shows the GM menu again.
func (s *Server) gmChat(menu bool) GMCommand {
	return func(p *Player, args []string) {
		if len(args) > 0 {
			say := serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{
				Type:       serverpackets.ChatAlliance,
				SenderName: p.Name(),
				Text:       strings.Join(args, " "),
			})
			for v := range s.Players() {
				if v.IsGM() && v.InGame() {
					v.Send(say)
				}
			}
		}
		if menu {
			s.ShowAdminPage(p, "gm_menu.htm")
		}
	}
}

// The enchant page.

// gmEnchantPage is //enchant (Mobius AdminEnchant): the enchant page.
func (s *Server) gmEnchantPage(p *Player, args []string) {
	if len(args) == 0 {
		s.ShowAdminPage(p, "enchant.htm")
	}
}

// gmSetEnchant is //seteh <level> and the other slots (Mobius
// AdminEnchant): the item worn in the slot by the GM's target player (or
// the GM) gets the enchant level, 0 to 65535; then the enchant page shows.
func (s *Server) gmSetEnchant(slot storage.PaperdollSlot) GMCommand {
	return func(p *Player, args []string) {
		if len(args) == 0 {
			gmSysMessage(p, "Please specify a new enchant value.")
		} else if v, ok := javaParseInt(strings.Join(args, " ")); !ok {
			gmSysMessage(p, "Please specify a valid new enchant value.")
		} else if v < 0 || v > 65535 {
			gmSysMessage(p, "You must set the enchant level to be between 0-65535.")
		} else {
			s.adminSetEnchant(p, int32(v), slot) //nolint:gosec // 0-65535
		}
		s.ShowAdminPage(p, "enchant.htm")
	}
}

// adminSetEnchant is AdminEnchant.setEnchant: the item comes off, gets the
// level and goes on again (its stats follow), the InventoryUpdate and the
// UserInfo and CharInfo go out, and both are told. Mobius caps the level for
// players below GM level with the enchant data (OverEnchantProtection),
// which the server does not have yet.
func (s *Server) adminSetEnchant(gm *Player, level int32, slot storage.PaperdollSlot) {
	t := gm
	if gm.Target != nil {
		tp, ok := gm.Target.(*Player)
		if !ok || tp == nil {
			gm.SendSystemMessage(gmSysMsgInvalidTarget)
			return
		}
		t = tp
	}
	it := t.inv.Paperdoll(slot)
	if it == nil || it.Slot() != slot {
		return
	}
	cur := it.Enchant()
	t.inv.UnequipSlot(slot)
	it.SetEnchant(level)
	t.inv.Equip(it, inventory.Wearer{GM: t.IsGM()})
	s.itemsChanged(t, false, nil)
	t.BroadcastUserInfo()
	name := it.Template().Name
	gmSysMessage(gm, fmt.Sprintf("Changed enchantment of %s's %s from %d to %d.", t.Name(), name, cur, level))
	t.SendMessage(fmt.Sprintf("Admin has changed the enchantment of your %s from %d to %d.", name, cur, level))
}

// The Effects page.

// adminEffectsPage is AdminEffects.showMainPage: the page of the command's
// kind.
func (s *Server) adminEffectsPage(p *Player, cmd string) {
	page := "effects_menu"
	switch {
	case strings.Contains(cmd, "ave_abnormal"):
		page = "ave_abnormal"
	case strings.Contains(cmd, "ave_special"):
		page = "ave_special"
	case strings.Contains(cmd, "ave_event"):
		page = "ave_event"
	case strings.Contains(cmd, "social"):
		page = "social"
	}
	s.ShowAdminPage(p, page+".htm")
}

// hasAdminVisual reports whether the creature shows the visual effect.
func hasAdminVisual(b *skillBase, ave data.AbnormalVisualEffect) bool {
	if b.admin.aves&(1<<ave) != 0 {
		return true
	}
	m := ave.Mask()
	return m != 0 && !ave.IsSpecial() && !ave.IsEvent() && b.el.Visual().Normal&m != 0
}

// setAdminVisual starts or stops a visual effect and shows it (Mobius
// start/stopAbnormalVisualEffect(true, ...)).
func setAdminVisual(b *skillBase, ave data.AbnormalVisualEffect, on bool) {
	if on {
		b.admin.aves |= 1 << ave
	} else {
		b.admin.aves &^= 1 << ave
	}
	b.UpdateAbnormalEffect()
}

// adminParalyze paralyzes c with the visual effect (Mobius
// startAbnormalVisualEffect, setParalyzed(true), startParalyze): its swing
// and cast stop, it stops where it is, and it can neither move, attack nor
// cast until //unpara.
func (s *Server) adminParalyze(c Creature, ave data.AbnormalVisualEffect) {
	b := s.skillBaseOf(c)
	if b == nil {
		return
	}
	setAdminVisual(b, ave, true)
	b.admin.paralyzed = true
	s.AbortAttack(c)
	b.AbortCast()
	s.stopCreature(c)
}

// adminUnparalyze ends //para (Mobius stopAbnormalVisualEffect,
// setParalyzed(false)).
func (s *Server) adminUnparalyze(c Creature, ave data.AbnormalVisualEffect) {
	b := s.skillBaseOf(c)
	if b == nil {
		return
	}
	setAdminVisual(b, ave, false)
	b.admin.paralyzed = false
}

// gmPara is //para [type] and //unpara [type] (Mobius AdminEffects): the
// GM's target creature is paralyzed (or freed), with the paralysis look for
// type 1 (the default) and petrification otherwise. The menu versions show
// the Effects page.
func (s *Server) gmPara(name string, on bool) GMCommand {
	return func(p *Player, args []string) {
		typ := "1"
		if len(args) > 0 {
			typ = args[0]
		}
		ave := data.VisualParalyze
		if typ != "1" {
			ave = data.VisualFleshStone
		}
		if c, ok := p.Target.(Creature); ok && c != nil {
			if on {
				s.adminParalyze(c, ave)
			} else {
				s.adminUnparalyze(c, ave)
			}
		}
		if strings.Contains(name, "menu") {
			s.adminEffectsPage(p, name)
		}
	}
}

// gmParaAll is //para_all and //unpara_all (Mobius AdminEffects): every
// player around the GM below GM level is paralyzed, or every player around
// freed. (Mobius's menu versions fall into //para by mistake and paralyze
// only the target; here they do what their buttons say.)
func (s *Server) gmParaAll(name string, on bool) GMCommand {
	return func(p *Player, _ []string) {
		var around []*Player
		for v := range s.world.VisiblePlayers(p) {
			if vp, ok := v.(*Player); ok {
				around = append(around, vp)
			}
		}
		for _, v := range around {
			switch {
			case !on:
				s.adminUnparalyze(v, data.VisualParalyze)
			case !v.IsGM():
				s.adminParalyze(v, data.VisualParalyze)
			}
		}
		if strings.Contains(name, "menu") {
			s.adminEffectsPage(p, name)
		}
	}
}

// gmBigHead is //bighead and //shrinkhead (Mobius AdminEffects): the GM's
// target creature gets a big head, or its normal one back.
func (s *Server) gmBigHead(on bool) GMCommand {
	return func(p *Player, _ []string) {
		if c, ok := p.Target.(Creature); ok && c != nil {
			if b := s.skillBaseOf(c); b != nil {
				setAdminVisual(b, data.VisualBigHead, on)
			}
		}
	}
}

// objectName returns the name of a world object, "" for one without.
func objectName(o world.Object) string {
	if n, ok := o.(interface{ Name() string }); ok {
		return n.Name()
	}
	return ""
}

// gmSocial is //social <id> [player name|radius] and //social_menu (Mobius
// AdminEffects): the GM's target (or the GM), the named player or every
// creature within the radius plays the social action; the menu version
// shows the social page.
func (s *Server) gmSocial(menu bool) GMCommand {
	return func(p *Player, args []string) {
		switch len(args) {
		case 2:
			social, ok := javaParseInt(args[0])
			if !ok {
				break
			}
			if t, found := s.PlayerByName(args[1]); found {
				if s.performSocial(social, t, p) {
					p.SendMessage(t.Name() + " was affected by your request.")
				}
			} else if radius, ok := javaParseInt(args[1]); ok {
				var around []world.Object
				for o := range s.world.VisibleInRange(p, int32(radius)) { //nolint:gosec // an int
					around = append(around, o)
				}
				for _, o := range around {
					s.performSocial(social, o, p)
				}
				p.SendMessage(strconv.Itoa(radius) + " units radius affected by your request.")
			} else {
				gmSysMessage(p, "Incorrect parameter")
			}
		case 1:
			social, ok := javaParseInt(args[0])
			if !ok {
				break
			}
			obj := p.Target
			if obj == nil {
				obj = p
			}
			if s.performSocial(social, obj, p) {
				p.SendMessage(objectName(obj) + " was affected by your request.")
			} else {
				p.SendSystemMessage(adminSysMsgNothingHappened)
			}
		default:
			if !menu {
				gmSysMessage(p, "Usage: //social <social_id> [player_name|radius]")
			}
		}
		if menu {
			s.adminEffectsPage(p, "social_menu")
		}
	}
}

// performSocial is AdminEffects.performSocial: a creature plays the action
// for everyone who sees it. Chests do nothing, NPCs only actions 1 to 3,
// players 2 to 18.
func (s *Server) performSocial(action int, o world.Object, gm *Player) bool {
	c, ok := o.(Creature)
	if !ok || s.skillBaseOf(c) == nil {
		return false
	}
	if n, isNpc := c.(*Npc); isNpc {
		if n.Template.Type == "Chest" || action < 1 || action > 3 {
			gm.SendSystemMessage(adminSysMsgNothingHappened)
			return false
		}
	}
	if _, isPlayer := c.(*Player); isPlayer && (action < 2 || (action > 18 && action != serverpackets.SocialActionLevelUp)) {
		gm.SendSystemMessage(adminSysMsgNothingHappened)
		return false
	}
	s.world.Broadcast(c, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
		ObjectID: int32(c.ID()), //nolint:gosec // object IDs fit
		ActionID: int32(action), //nolint:gosec // an int
	}), true)
	return true
}

// gmEffect is //effect <skill> [level [hit time]] and //effect_menu (Mobius
// AdminEffects): the GM's target (or the GM) shows the skill's casting
// animation toward the GM.
func (s *Server) gmEffect(menu bool) GMCommand {
	return func(p *Player, args []string) {
		defer func() {
			if menu {
				s.adminEffectsPage(p, "effect_menu")
			}
		}()
		vals := []int{0, 1, 1} // skill, level, hit time
		valid := len(args) > 0
		for i := 0; valid && i < min(len(args), len(vals)); i++ {
			vals[i], valid = javaParseInt(args[i])
		}
		if !valid {
			gmSysMessage(p, "Usage: //effect skill [level | level hittime]")
			return
		}
		obj := p.Target
		if obj == nil {
			obj = p
		}
		c, ok := obj.(Creature)
		if !ok || s.skillBaseOf(c) == nil {
			p.SendSystemMessage(gmSysMsgInvalidTarget)
			return
		}
		from, to := c.Position(), p.Position()
		s.world.Broadcast(c, serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
			CasterID:   int32(c.ID()), //nolint:gosec // object IDs fit
			TargetID:   p.ObjectID(),
			SkillID:    int32(vals[0]), //nolint:gosec // an int
			SkillLevel: int32(vals[1]), //nolint:gosec // an int
			HitTime:    time.Duration(vals[2]) * time.Millisecond,
			X:          from.X,
			Y:          from.Y,
			Z:          from.Z,
			TargetX:    to.X,
			TargetY:    to.Y,
			TargetZ:    to.Z,
		}), true)
		p.SendMessage(fmt.Sprintf("%s performs MSU %d/%d by your request.", c.Name(), vals[0], vals[1]))
	}
}

// gmAbnormal is //ave_abnormal <effect> [radius] and its special and event
// versions (Mobius AdminEffects): the abnormal visual effect switches on or
// off on the GM's target (or the GM), or on every creature within the
// radius; then the page of the effects shows. An unknown effect does
// nothing.
func (s *Server) gmAbnormal(name string) GMCommand {
	return func(p *Player, args []string) {
		if len(args) == 0 {
			gmSysMessage(p, "Usage: //"+name+" <AbnormalVisualEffect> [radius]")
			s.adminEffectsPage(p, name)
			return
		}
		ave, ok := data.ParseAbnormalVisualEffect(args[0])
		if !ok {
			return
		}
		radius, param2 := 0, ""
		if len(args) == 2 {
			param2 = args[1]
			if adminNumeric(param2) {
				radius, _ = javaParseInt(param2)
			}
		}
		if radius > 0 {
			var around []world.Object
			for o := range s.world.VisibleInRange(p, int32(radius)) { //nolint:gosec // an int
				around = append(around, o)
			}
			for _, o := range around {
				s.performAbnormal(ave, o)
			}
			gmSysMessage(p, "Affected all characters in radius "+param2+" by "+args[0]+" abnormal visual effect.")
		} else {
			obj := p.Target
			if obj == nil {
				obj = p
			}
			if s.performAbnormal(ave, obj) {
				p.SendMessage(objectName(obj) + " affected by " + args[0] + " abnormal visual effect.")
			} else {
				p.SendSystemMessage(adminSysMsgNothingHappened)
			}
		}
		s.adminEffectsPage(p, name)
	}
}

// adminNumeric is StringUtil.isNumeric: digits only.
func adminNumeric(v string) bool {
	return v != "" && strings.Trim(v, "0123456789") == ""
}

// performAbnormal switches the visual effect of a creature.
func (s *Server) performAbnormal(ave data.AbnormalVisualEffect, o world.Object) bool {
	c, ok := o.(Creature)
	if !ok {
		return false
	}
	b := s.skillBaseOf(c)
	if b == nil {
		return false
	}
	setAdminVisual(b, ave, !hasAdminVisual(b, ave))
	return true
}

// gmEarthquake is //earthquake <intensity> <duration> (Mobius
// AdminEffects): an earthquake around the GM, for it and the players around.
func (s *Server) gmEarthquake(menu bool) GMCommand {
	return func(p *Player, args []string) {
		var intensity, duration int
		ok := len(args) >= 2
		if ok {
			intensity, ok = javaParseInt(args[0])
		}
		if ok {
			duration, ok = javaParseInt(args[1])
		}
		if ok {
			pos := p.Position()
			w := packets.NewWriter(25)
			w.WriteUint8(opEarthquake)
			w.WriteInt32(pos.X)
			w.WriteInt32(pos.Y)
			w.WriteInt32(pos.Z)
			w.WriteInt32(int32(intensity)) //nolint:gosec // an int
			w.WriteInt32(int32(duration))  //nolint:gosec // an int
			w.WriteInt32(0)
			s.world.Broadcast(p, w.Bytes(), true)
		} else {
			gmSysMessage(p, "Usage: //earthquake <intensity> <duration>")
		}
		if menu {
			s.adminEffectsPage(p, "earthquake_menu")
		}
	}
}

// gmAtmosphere is //atmosphere sky <day|night|red> <duration> (Mobius
// AdminEffects): every player sees sunrise, sunset or a red sky (10 seconds
// for a duration of 0). Mobius has no other type (its Seven Signs sky
// buttons get the usage too).
func (s *Server) gmAtmosphere(menu bool) GMCommand {
	return func(p *Player, args []string) {
		defer func() {
			if menu {
				s.adminEffectsPage(p, "atmosphere_menu")
			}
		}()
		const usage = "Usage: //atmosphere <sky day|night|red> <duration>"
		if len(args) < 3 {
			gmSysMessage(p, usage)
			return
		}
		duration, ok := javaParseInt(args[2])
		if !ok {
			gmSysMessage(p, usage)
			return
		}
		if args[0] != "sky" {
			gmSysMessage(p, usage)
			return
		}
		var body []byte
		switch args[1] {
		case "night":
			body = []byte{opSunSet}
		case "day":
			body = []byte{opSunRise}
		case "red":
			if duration == 0 {
				duration = 10
			}
			w := packets.NewWriter(7)
			w.WriteUint8(opExServer)
			w.WriteUint16(opExRedSky)
			w.WriteInt32(int32(duration)) //nolint:gosec // an int
			body = w.Bytes()
		}
		if body != nil {
			s.world.BroadcastAll(body)
		}
	}
}

// gmPlaySounds is //play_sounds [page] (Mobius AdminEffects): the page of
// sounds.
func (s *Server) gmPlaySounds(p *Player, args []string) {
	s.ShowAdminPage(p, "songs/songs"+strings.Join(args, " ")+".htm")
}

// gmPlaySound is //play_sound <sound> (Mobius AdminEffects): the GM and the
// players around hear the sound (the GM twice, as in Mobius).
func (s *Server) gmPlaySound(p *Player, args []string) {
	if len(args) == 0 {
		gmSysMessage(p, "Usage: //play_sound <soundname>")
		return
	}
	sound := strings.Join(args, " ")
	body := serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Type: 1, Sound: sound})
	p.Send(body)
	s.world.Broadcast(p, body, true)
	gmSysMessage(p, "Playing "+sound+".")
}

// gmEventTrigger is //event_trigger <id> <true|false> (Mobius AdminEffects):
// the GM and the players around get OnEventTrigger.
func (s *Server) gmEventTrigger(p *Player, args []string) {
	if len(args) < 2 {
		gmSysMessage(p, "Usage: //event_trigger id [true | false]")
		return
	}
	id, ok := javaParseInt(args[0])
	if !ok {
		gmSysMessage(p, "Usage: //event_trigger id [true | false]")
		return
	}
	w := packets.NewWriter(6)
	w.WriteUint8(opEventTrigger)
	w.WriteInt32(int32(id)) //nolint:gosec // an int
	if strings.EqualFold(args[1], "true") {
		w.WriteUint8(1)
	} else {
		w.WriteUint8(0)
	}
	body := w.Bytes()
	for v := range s.world.VisiblePlayers(p) {
		world.Send(v, body)
	}
	p.Send(body)
}

// The skill pages (Mobius AdminSkill).

// gmShowSkills is //show_skills: the skill editor page of the target player.
func (s *Server) gmShowSkills(p *Player, args []string) {
	if len(args) == 0 {
		s.gmShowSkillsPage(p)
	}
}

// gmRemoveSkills is //remove_skills <page>: the target player's skills to
// remove, 10 to a page.
func (s *Server) gmRemoveSkills(p *Player, args []string) {
	if len(args) == 0 {
		return
	}
	if page, ok := javaParseInt(strings.Join(args, " ")); ok && page >= 0 {
		s.gmRemoveSkillsPage(p, page)
	}
}

// adminEnumName is StringUtil.enumToString: "DARK_ELF" as "Dark Elf".
func adminEnumName(name string) string {
	words := strings.Split(strings.ToLower(name), "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// adminClassTree returns the complete skill tree of a class.
func (s *Server) adminClassTree(c data.PlayerClass) []data.SkillLearn {
	if s.skills.data == nil {
		return nil
	}
	tree, _ := s.skills.data.SkillTree(int(c))
	return tree
}

// adminParseClass returns the class with the Mobius constant name.
func adminParseClass(name string) (data.PlayerClass, bool) {
	for _, c := range data.PlayerClasses() {
		if c.String() == name {
			return c, true
		}
	}
	return 0, false
}

// gmSkillList is //skill_list (Mobius adminAddSkills): the races whose
// classes have skills, each a link to its classes, and a form to add a
// skill by ID.
func (s *Server) gmSkillList(p *Player, _ []string) {
	var races []data.Race
	for _, c := range data.PlayerClasses() {
		if r := c.Race(); !slices.Contains(races, r) && len(s.adminClassTree(c)) > 0 {
			races = append(races, r)
		}
	}
	var b strings.Builder
	for _, r := range races {
		b.WriteString(`<tr><td><a action="bypass admin_skill_race ` + r.String() + `">` + adminEnumName(r.String()) + `</a></td></tr>`)
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/charskills_add.htm").replace("%racelist%", b.String()))
}

// gmSkillRace is //skill_race <RACE> (Mobius adminAddRaceSkills): the
// race's classes with skills, colored by class level.
func (s *Server) gmSkillRace(p *Player, args []string) {
	if len(args) != 1 {
		s.gmSkillList(p, nil)
		return
	}
	race, ok := data.ParseRace(args[0])
	if !ok {
		return
	}
	var b strings.Builder
	for _, c := range data.PlayerClasses() {
		if c.Race() != race || len(s.adminClassTree(c)) == 0 {
			continue
		}
		color := "777777"
		switch c.Level() {
		case 1:
			color = "999999"
		case 2:
			color = "F2B705"
		case 3:
			color = "F26005"
		case 4:
			color = "FF0022"
		}
		b.WriteString(`<tr><td><a action="bypass admin_skill_class ` + c.String() + `"><font color="` + color + `">` + s.className(int(c)) + `</font></a></td></tr>`)
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/charskills_race.htm").
		replace("%racename%", adminEnumName(race.String())).
		replace("%classlist%", b.String()))
}

// adminPaging is Mobius PageBuilder with ButtonsStyle and DefaultPageHandler:
// the elements of the page (7 to a page) and the pager's table, "" for a
// single page. Like Mobius's pager, it leaves out the last page's button.
func adminPaging(n, page int, bypass string) (start, end int, pager string) {
	pages := n / adminPageLimit
	if n%adminPageLimit > 0 {
		pages++
	}
	page = max(page, 0)
	if pages > 1 {
		var b strings.Builder
		b.WriteString("<table width=280 cellspacing=0><tr>")
		last := pages - 1 // DefaultPageHandler.apply gets pages - 1
		button := func(i int) {
			if i == page {
				b.WriteString("<td>" + strconv.Itoa(i+1) + "</td>")
				return
			}
			fmt.Fprintf(&b, `<td><button action="%s %d" value="%d" width="30" height="15" back="sek.cbui94" fore="sek.cbui92"></td>`, bypass, i, i+1)
		}
		const offset = 2
		from, to := max(page-offset, 0), min(page+offset+1, last)
		if from > offset {
			for i := range offset {
				button(i)
			}
			b.WriteString("<td align=center> | </td>")
		}
		for i := from; i < to; i++ {
			button(i)
		}
		if last > to {
			b.WriteString("<td align=center> | </td>")
			for i := last - offset; i < last; i++ {
				button(i)
			}
		}
		b.WriteString("</tr></table>")
		pager = b.String()
	}
	if page > pages {
		page = pages - 1
	}
	start = min(max(adminPageLimit*page, 0), n)
	end = min(start+adminPageLimit, n)
	return start, end, pager
}

// adminSkillRow is a row of the skill pages: the icon and a link.
func adminSkillRow(b *strings.Builder, icon, action, text string) {
	b.WriteString(`<table width="295" bgcolor="000000"><tr><td height=40 width=40><img src="` + icon +
		`" width=32 height=32></td><td width=190><font color="B09878"><a action="` + action + `">` + text +
		`</a></font></td></tr></table><img src="L2UI.SquareGray" width=295 height=1>`)
}

// gmSkillClass is //skill_class <CLASS> [page] (Mobius adminAddClassSkills):
// the skills of the class's tree, 7 to a page, each a link to its levels.
func (s *Server) gmSkillClass(p *Player, args []string) {
	if len(args) == 0 {
		s.gmSkillList(p, nil)
		return
	}
	class, ok := adminParseClass(args[0])
	page := 0
	if ok && len(args) > 1 {
		page, ok = javaParseInt(args[1])
	}
	if !ok {
		return
	}
	var list []*skills.Skill
	for _, l := range s.adminClassTree(class) {
		if slices.ContainsFunc(list, func(sk *skills.Skill) bool { return sk.ID == l.SkillID }) {
			continue
		}
		if sk, found := s.Skill(l.SkillID, 1); found {
			list = append(list, sk)
		}
	}
	start, end, pager := adminPaging(len(list), page, "bypass admin_skill_class "+class.String())
	var b strings.Builder
	for _, sk := range list[start:end] {
		id := strconv.Itoa(sk.ID)
		adminSkillRow(&b, sk.Icon, "bypass admin_skill_add_list "+class.String()+" "+id, sk.Name+" ("+id+")")
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/charskills_class.htm").
		replace("%classname%", s.className(int(class))).
		replace("%skilllist%", b.String()).
		replace("%pages%", pager))
}

// gmSkillAddList is //skill_add_list <CLASS> [skill] [page] (Mobius
// adminAddClassSkillList): the levels of the skill in the class's tree,
// each a link to //add_skill.
func (s *Server) gmSkillAddList(p *Player, args []string) {
	if len(args) == 0 {
		s.gmSkillList(p, nil)
		return
	}
	class, ok := adminParseClass(args[0])
	skillID, page := 1, 0
	if ok && len(args) > 1 {
		skillID, ok = javaParseInt(args[1])
	}
	if ok && len(args) > 2 {
		page, ok = javaParseInt(args[2])
	}
	if !ok {
		return
	}
	byLevel := make(map[int]*skills.Skill)
	for _, l := range s.adminClassTree(class) {
		if l.SkillID == skillID {
			if sk, found := s.Skill(l.SkillID, l.Level); found {
				byLevel[l.Level] = sk
			}
		}
	}
	levels := make([]int, 0, len(byLevel))
	for lv := range byLevel {
		levels = append(levels, lv)
	}
	slices.Sort(levels)
	start, end, pager := adminPaging(len(levels), page, "bypass admin_skill_add_list "+class.String()+" "+strconv.Itoa(skillID))
	var b strings.Builder
	for _, lv := range levels[start:end] {
		sk := byLevel[lv]
		adminSkillRow(&b, sk.Icon, "bypass admin_add_skill "+strconv.Itoa(sk.ID)+" "+strconv.Itoa(lv), sk.Name+" ("+strconv.Itoa(lv)+")")
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/charskills_add_list.htm").
		replace("%skillid%", strconv.Itoa(skillID)).
		replace("%skilllist%", b.String()).
		replace("%classId%", class.String()).
		replace("%pages%", pager))
}

// gmSkillIndex is //skill_index <page> (Mobius AdminSkill): the page
// skills/<page>.htm of the admin folder.
func (s *Server) gmSkillIndex(p *Player, args []string) {
	if len(args) > 0 {
		s.ShowAdminPage(p, "skills/"+strings.Join(args, " ")+".htm")
	}
}

// gmGetSkills is //get_skills (Mobius adminGetSkills): the GM swaps its
// skills for the target player's (saved, as in Mobius) until
// //reset_skills; then the skill editor page shows.
func (s *Server) gmGetSkills(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	if t.Name() == p.Name() {
		p.SendSystemMessage(adminSysMsgCannotUseOnYourself)
	} else {
		theirs := t.Skills()
		p.admin.taken = p.Skills()
		for _, sk := range p.admin.taken {
			s.RemoveSkill(p, sk.ID, true, true)
		}
		for _, sk := range theirs {
			s.AddSkill(p, sk, true)
		}
		gmSysMessage(p, "You now have all the skills of "+t.Name()+".")
		p.SendSkillList()
	}
	s.gmShowSkillsPage(p)
}

// gmResetSkills is //reset_skills (Mobius adminResetSkills): the target
// player gets the GM's current skills, and the GM its own back.
func (s *Server) gmResetSkills(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	if p.admin.taken == nil {
		gmSysMessage(p, "You must get the skills of someone in order to do this.")
	} else {
		theirs := t.Skills()
		for _, sk := range theirs {
			s.RemoveSkill(t, sk.ID, true, true)
		}
		for _, sk := range p.Skills() {
			s.AddSkill(t, sk, true)
		}
		for _, sk := range theirs {
			s.RemoveSkill(p, sk.ID, true, true)
		}
		for _, sk := range p.admin.taken {
			s.AddSkill(p, sk, true)
		}
		t.SendMessage("[GM]" + p.Name() + " updated your skills.")
		gmSysMessage(p, "You now have all your skills back.")
		p.admin.taken = nil
		p.SendSkillList()
		t.SendSkillList()
	}
	s.gmShowSkillsPage(p)
}

// gmGiveAllFS is //give_all_skills_fs (Mobius adminGiveAllSkills with the
// Forgotten Scroll skills and the skills that need items).
func (s *Server) gmGiveAllFS(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	n := s.giveAvailableSkills(t, true, true)
	gmSysMessage(p, "You gave "+strconv.Itoa(n)+" skills to "+t.Name())
	t.SendSkillList()
}

// gmRemoveAllSkills is //remove_all_skills (Mobius AdminSkill): the target
// player loses every skill.
func (s *Server) gmRemoveAllSkills(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	for _, sk := range t.Skills() {
		s.RemoveSkill(t, sk.ID, true, true)
	}
	gmSysMessage(p, "You have removed all skills from "+t.Name()+".")
	t.SendMessage("Admin removed all skills from you.")
	t.SendSkillList()
	t.BroadcastUserInfo()
}

// gmGiveClanSkills is //give_clan_skills and //give_all_clan_skills (Mobius
// AdminSkill): a target without a clan gets Mobius's answer; clans do not
// exist yet.
func (s *Server) gmGiveClanSkills(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(p)
	switch {
	case !ok:
		p.SendSystemMessage(gmSysMsgInvalidTarget)
	case t.char.ClanID == 0:
		p.SendSystemMessage(adminSysMsgMustBeClanMember)
	default:
		p.SendMessage("Clans are not available yet.")
	}
}

// gmAddClanSkill is //add_clan_skill <id> <level> (Mobius AdminSkill): no
// player leads a clan yet, so it answers "... is not a clan leader." and
// shows the skill editor page.
func (s *Server) gmAddClanSkill(p *Player, args []string) {
	if len(args) < 2 {
		gmSysMessage(p, "Usage: //add_clan_skill <skill_id> <level>")
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
	} else {
		p.SendSystemMessage(adminSysMsgNotClanLeader, serverpackets.TextParam(t.Name()))
	}
	s.gmShowSkillsPage(p)
}

// gmSetSkill is //setskill <id> <level> (Mobius AdminSkill): the GM gets
// the skill level, not saved.
func (s *Server) gmSetSkill(p *Player, args []string) {
	if len(args) < 2 {
		return
	}
	id, ok1 := javaParseInt(args[0])
	level, ok2 := javaParseInt(args[1])
	if !ok1 || !ok2 {
		return
	}
	sk, found := s.Skill(id, level)
	if !found {
		return
	}
	s.AddSkill(p, sk, false)
	p.SendSkillList()
	gmSysMessage(p, fmt.Sprintf("You added yourself skill %s(%d) level %d", sk.Name, id, level))
}

// The buff pages (Mobius AdminBuffs).

// gmViewSkillEffects is //viewskilleffects <objectId> <skillId> <level>
// [page] and its passive version: the effects of a skill level on a
// creature, each with its time and a button to stop the skill.
func (s *Server) gmViewSkillEffects(passive bool) GMCommand {
	return func(p *Player, args []string) {
		vals := make([]int, 4)
		for i := range min(len(args), 4) {
			v, ok := javaParseInt(args[i])
			if !ok {
				gmSysMessage(p, "Failed viewing skill effects: "+javaParseIntError(args[i]))
				gmSysMessage(p, "Usage: //viewskilleffects <objectId skillId skillLevel>")
				return
			}
			vals[i] = v
		}
		if len(args) < 3 {
			gmSysMessage(p, "Failed viewing skill effects: null")
			gmSysMessage(p, "Usage: //viewskilleffects <objectId skillId skillLevel>")
			return
		}
		target := s.gmCreatureByID(vals[0])
		if target == nil {
			gmSysMessage(p, "Target with object id "+strconv.Itoa(vals[0])+" not found.")
			return
		}
		el := s.skillBaseOf(target).el
		infos := el.Effects()
		bypass := "bypass admin_viewskilleffects "
		if passive {
			infos = el.Passives()
			bypass = "bypass admin_viewskilleffects_ps "
		}
		var list []*skills.BuffInfo
		count := 0
		for _, info := range infos {
			if sk := info.Skill(); sk != nil && sk.ID == vals[1] && sk.Level == vals[2] {
				list = append(list, info)
				count += len(info.Effects())
			}
		}
		objID := strconv.Itoa(vals[0])
		start, end, pager := adminPaging(len(list), vals[3], bypass+objID+" "+strconv.Itoa(vals[1])+" "+strconv.Itoa(vals[2]))
		var b strings.Builder
		for _, info := range list[start:end] {
			sk := info.Skill()
			red, endRed := "", ""
			if !info.InUse() {
				red, endRed = adminFontRed, adminFontEnd
			}
			left := strconv.Itoa(info.Time()) + "s"
			switch {
			case sk.IsToggle():
				left = "T"
			case sk.IsPassive():
				left = "P"
			}
			for _, e := range info.Effects() {
				b.WriteString(`<table width="295" bgcolor="000000"><tr><td width=750>` + red + sk.Name + " Lv. " + strconv.Itoa(sk.Level) +
					" (" + e.Name().String() + ")" + endRed + `</td><td width=180><center>` + left +
					`</center></td><td width=200><button value="X" action="bypass admin_stopbuff ` + objID + " " + strconv.Itoa(sk.ID) +
					`" width=30 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr></table><img src="L2UI.SquareGray" width=295 height=1>`)
			}
		}
		buffsText, passives := "Show Passives", "_ps"
		if passive {
			buffsText, passives = "Hide Passives", ""
		}
		s.sendPage(p, s.pageFile(p, 0, "admin/getbuff_effects.htm").
			replace("%pages%", pager).
			replace("%buffsText%", buffsText).
			replace("%passives%", passives).
			replace("%targetName%", target.Name()).
			replace("%targetObjId%", objID).
			replace("%buffs%", b.String()).
			replace("%effectSize%", strconv.Itoa(count)))
	}
}

// gmViewBlockedEffects is //viewblockedeffects <objectId> (Mobius
// AdminBuffs): the abnormal types the creature's effect list refuses.
func (s *Server) gmViewBlockedEffects(p *Player, args []string) {
	if len(args) == 0 {
		gmSysMessage(p, "Failed viewing blocked effects: null")
		gmSysMessage(p, "Usage: //viewblockedeffects <objectId>")
		return
	}
	id, ok := javaParseInt(args[0])
	if !ok {
		gmSysMessage(p, "Failed viewing blocked effects: "+javaParseIntError(args[0]))
		gmSysMessage(p, "Usage: //viewblockedeffects <objectId>")
		return
	}
	target := s.gmCreatureByID(id)
	if target == nil {
		gmSysMessage(p, "Target with object id "+strconv.Itoa(id)+" not found.")
		return
	}
	var b strings.Builder
	b.WriteString(`<html><table width="100%"><tr><td width=45><button value="Main" action="bypass admin_admin" width=45 height=15 back="sek.cbui94" fore="sek.cbui92"></td><td width=180><center><font color="LEVEL">Blocked effects of `)
	b.WriteString(target.Name())
	b.WriteString(`</font></td><td width=45><button value="Back" action="bypass admin_getbuffs ` + strconv.Itoa(id) + `" width=45 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr></table><br>`)
	if blocked := s.skillBaseOf(target).el.BlockedAbnormals(); len(blocked) > 0 {
		b.WriteString("<br>Blocked buff slots: ")
		for _, t := range blocked {
			b.WriteString("<br>" + t.String())
		}
	}
	b.WriteString("</html>")
	s.sendPage(p, newPage(0, b.String()))
}

// gmAreaCancel is //areacancel <radius> (Mobius AdminBuffs): every effect
// of the players within the radius ends.
func (s *Server) gmAreaCancel(p *Player, args []string) {
	if len(args) == 0 {
		gmSysMessage(p, "Usage: //areacancel <radius>")
		return
	}
	radius, ok := javaParseInt(args[0])
	if !ok {
		gmSysMessage(p, "Invalid radius value.")
		return
	}
	var around []*Player
	for v := range s.world.VisiblePlayersInRange(p, int32(radius)) { //nolint:gosec // an int
		if vp, ok := v.(*Player); ok {
			around = append(around, vp)
		}
	}
	for _, v := range around {
		v.Effects().StopAll()
		v.BroadcastUserInfo()
	}
	gmSysMessage(p, fmt.Sprintf("Effects canceled for %d player(s) within radius %d", len(around), radius))
}
