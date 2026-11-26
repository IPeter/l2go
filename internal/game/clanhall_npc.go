package game

import (
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/social"
)

// The clan hall NPCs written in Java in Mobius: the hall manager
// (ClanHallManager, a merchant that rents the functions, opens the doors,
// banishes foreigners, buffs, teleports and sells the item creation lists)
// and the doorman (ClanHallDoorman). Their pages are Mobius's
// data/html/clanHallManager/ and clanHallDoorman/.

// The conditions of ClanHallManager.validateCondition.
const (
	hallCondOwnerFalse = 0
	hallCondAllFalse   = 1
	hallCondOwner      = 3
)

// hallDateFormat is ClanHallManager.DATE_FORMAT (dd/MM/yyyy HH:mm).
const hallDateFormat = "02/01/2006 15:04"

const hallManagerHTML = "clanHallManager/"

// hallManagerCondition is ClanHallManager.validateCondition.
func (s *Server) hallManagerCondition(p *Player, h *ClanHall) int {
	switch {
	case h == nil:
		return hallCondAllFalse
	case p.IsGM():
		return hallCondOwner
	case p.Clan() == nil:
		return hallCondAllFalse
	case h.ownerID == p.ClanID():
		return hallCondOwner
	}
	return hallCondOwnerFalse
}

// hallManagerPage sends a manager page with %objectId% and %npcId% filled
// (sendHtmlMessage).
func (s *Server) hallManagerPage(p *Player, n *Npc, pg *htmlPage) {
	s.sendPage(p, pg.replaceObjectID(n.ID()).replace("%npcId%", strconv.Itoa(n.Template.ID)))
}

// hallManagerFile is a page of the manager's folder.
func (s *Server) hallManagerFile(p *Player, n *Npc, name string) *htmlPage {
	return s.pageFile(p, n.ID(), hallManagerHTML+name)
}

// hallChamberlainPage is the owner's first page: chamberlain-<npcId>.htm,
// or chamberlain.htm without one.
func (s *Server) hallChamberlainPage(p *Player, n *Npc) *htmlPage {
	name := hallManagerHTML + "chamberlain-" + strconv.Itoa(n.Template.ID) + ".htm"
	if !s.dialog.html.exists(name) {
		name = hallManagerHTML + "chamberlain.htm"
	}
	return s.pageFile(p, n.ID(), name).replace("%npcname%", n.Template.Name)
}

// showHallManagerPage is ClanHallManager.showChatWindow.
func (s *Server) showHallManagerPage(p *Player, n *Npc) {
	p.SendActionFailed()
	h := s.npcClanHall(n)
	switch s.hallManagerCondition(p, h) {
	case hallCondOwner:
		s.hallManagerPage(p, n, s.hallChamberlainPage(p, n))
	case hallCondOwnerFalse:
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "chamberlain-of.htm"))
	default:
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "chamberlain-no.htm"))
	}
}

// hallManagerCommand is ClanHallManager.onBypassFeedback: nothing during
// the hall's siege or for anyone but the owner clan (and GMs); the owner's
// commands; it reports false for the commands of Merchant (Buy, the
// warehouse, ...), which go on to the bypass handlers.
func (s *Server) hallManagerCommand(p *Player, n *Npc, command string) bool {
	h := s.npcClanHall(n)
	if h != nil && h.siege != nil && h.siege.status == residence.HallRunning {
		return true
	}
	if s.hallManagerCondition(p, h) != hallCondOwner {
		return true
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	val := ""
	if len(fields) > 1 {
		val = fields[1]
	}
	rest := fields[min(2, len(fields)):]
	switch strings.ToLower(fields[0]) {
	case "banish_foreigner":
		pg := s.hallManagerFile(p, n, "not_authorized.htm")
		if p.HasClanAccess(social.AccessHallBanish) {
			switch strings.ToLower(val) {
			case "list":
				pg = s.hallManagerFile(p, n, "banish-list.htm")
			case "banish":
				s.banishForeigners(h)
				pg = s.hallManagerFile(p, n, "banish.htm")
			}
		}
		s.hallManagerPage(p, n, pg)
	case "manage_vault":
		pg := s.hallManagerFile(p, n, "not_authorized.htm")
		if p.HasClanAccess(social.AccessWarehouse) {
			if h.Lease() <= 0 {
				pg = s.hallManagerFile(p, n, "vault-chs.htm")
			} else {
				pg = s.hallManagerFile(p, n, "vault.htm").replace("%rent%", strconv.Itoa(h.Lease())).
					replace("%date%", s.calendar.In(h.PaidUntil()).Format(hallDateFormat))
			}
		}
		s.hallManagerPage(p, n, pg)
	case "door":
		pg := s.hallManagerFile(p, n, "not_authorized.htm")
		if p.HasClanAccess(social.AccessHallOpenDoor) {
			switch strings.ToLower(val) {
			case "open":
				s.openCloseHallDoors(h, true)
				pg = s.hallManagerFile(p, n, "door-open.htm")
			case "close":
				s.openCloseHallDoors(h, false)
				pg = s.hallManagerFile(p, n, "door-close.htm")
			default:
				pg = s.hallManagerFile(p, n, "door.htm")
			}
		}
		s.hallManagerPage(p, n, pg)
	case "functions":
		s.hallFunctionsCommand(p, n, h, strings.ToLower(val), rest)
	case "manage":
		s.hallManageCommand(p, n, h, strings.ToLower(val), rest)
	case "support":
		s.hallSupportCommand(p, n, h, val, rest)
	case "list_back":
		s.hallManagerPage(p, n, s.hallChamberlainPage(p, n))
	case "support_back":
		f := h.functions[residence.HallFuncSupport]
		if f == nil || f.Level == 0 {
			return true
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "support"+strconv.Itoa(f.Level)+".htm").replace("%mp%", strconv.Itoa(int(n.CurMP))))
	case "goto":
		f := h.functions[residence.HallFuncTeleport]
		if f == nil || f.Level <= 0 || val == "" {
			return true
		}
		if idx, err := strconv.Atoi(val); err == nil {
			s.doTeleport(p, n, "tel"+strconv.Itoa(f.Level), idx)
			p.SendActionFailed()
		}
	default:
		return false
	}
	return true
}

// hallFunctionsCommand is the "functions" command: the teleport, item
// creation and support pages of the rented functions.
func (s *Server) hallFunctionsCommand(p *Player, n *Npc, h *ClanHall, val string, rest []string) {
	switch val {
	case "tele":
		f := h.functions[residence.HallFuncTeleport]
		if f == nil {
			s.hallManagerPage(p, n, s.hallManagerFile(p, n, "chamberlain-nac.htm"))
			return
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "tele"+h.Data.Location+strconv.Itoa(f.Level)+".htm"))
	case "item_creation":
		f := h.functions[residence.HallFuncItemCreate]
		if f == nil {
			s.hallManagerPage(p, n, s.hallManagerFile(p, n, "chamberlain-nac.htm"))
			return
		}
		if len(rest) < 1 {
			return
		}
		id, err := strconv.Atoi(rest[0])
		if err != nil {
			return
		}
		s.showBuyWindow(p, n, id+f.Level*100000)
	case "support":
		f := h.functions[residence.HallFuncSupport]
		if f == nil {
			s.hallManagerPage(p, n, s.hallManagerFile(p, n, "chamberlain-nac.htm"))
			return
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "support"+strconv.Itoa(f.Level)+".htm").replace("%mp%", strconv.Itoa(int(n.CurMP))))
	case "back":
		s.showHallManagerPage(p, n)
	default:
		lvl := func(t int) string {
			if f := h.functions[t]; f != nil {
				return strconv.Itoa(f.Level)
			}
			return "0"
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "functions.htm").replace("%xp_regen%", lvl(residence.HallFuncRestoreExp)).
			replace("%hp_regen%", lvl(residence.HallFuncRestoreHP)).replace("%mp_regen%", lvl(residence.HallFuncRestoreMP)))
	}
}

// hallFuncSpec describes a function on the manage pages: its group
// ("recovery", "other" or "deco") and key in the links, its name and use on
// the apply page, and whether its level reads as a percentage.
type hallFuncSpec struct {
	typ        int
	group, key string
	name       string
	use        func(level int) string
	percent    bool
}

// hallFuncSpecs are the manage pages' functions (ClanHallManager's edit_*
// and apply branches).
var hallFuncSpecs = []hallFuncSpec{
	{typ: residence.HallFuncRestoreHP, group: "recovery", key: "hp", name: "Fireplace (HP Recovery Device)", percent: true,
		use: func(l int) string {
			return "Provides additional HP recovery for clan members in the clan hall.<font color=\"00FFFF\">" + strconv.Itoa(l) + "%</font>"
		}},
	{typ: residence.HallFuncRestoreMP, group: "recovery", key: "mp", name: "Carpet (MP Recovery)", percent: true,
		use: func(l int) string {
			return "Provides additional MP recovery for clan members in the clan hall.<font color=\"00FFFF\">" + strconv.Itoa(l) + "%</font>"
		}},
	{typ: residence.HallFuncRestoreExp, group: "recovery", key: "exp", name: "Chandelier (EXP Recovery Device)", percent: true,
		use: func(l int) string {
			return "Restores the Exp of any clan member who is resurrected in the clan hall.<font color=\"00FFFF\">" + strconv.Itoa(l) + "%</font>"
		}},
	{typ: residence.HallFuncItemCreate, group: "other", key: "item", name: "Magic Equipment (Item Production Facilities)",
		use: func(int) string { return "Allow the purchase of special items at fixed intervals." }},
	{typ: residence.HallFuncSupport, group: "other", key: "support", name: "Insignia (Supplementary Magic)",
		use: func(int) string { return "Enables the use of supplementary magic." }},
	{typ: residence.HallFuncTeleport, group: "other", key: "tele", name: "Mirror (Teleportation Device)",
		use: func(l int) string {
			return "Teleports clan members in a clan hall to the target <font color=\"00FFFF\">Stage " + strconv.Itoa(l) + "</font> staging area"
		}},
	{typ: residence.HallFuncCurtains, group: "deco", key: "curtains", name: "Curtains (Decoration)",
		use: func(int) string { return "These curtains can be used to decorate the clan hall." }},
	{typ: residence.HallFuncFrontPlatform, group: "deco", key: "fixtures", name: "Front Platform (Decoration)",
		use: func(int) string { return "Used to decorate the clan hall." }},
}

// hallFuncSpecOf finds a function by group and key.
func hallFuncSpecOf(group, key string) (hallFuncSpec, bool) {
	i := slices.IndexFunc(hallFuncSpecs, func(f hallFuncSpec) bool { return f.group == group && f.key == key })
	if i < 0 {
		return hallFuncSpec{}, false
	}
	return hallFuncSpecs[i], true
}

// hallLevelLinks are the levels each grade offers on the edit pages, by
// function (ClanHallManager's *_grade strings).
var hallLevelLinks = map[int][4][]int{
	residence.HallFuncRestoreHP:  {{20, 40, 220}, {40, 100, 160}, {80, 140, 200, 260}, {80, 120, 180, 240, 300}},
	residence.HallFuncRestoreExp: {{5, 10, 25}, {5, 15, 30}, {5, 15, 25, 40}, {15, 25, 35, 50}},
	residence.HallFuncRestoreMP:  {{5, 10, 25}, {5, 15, 25}, {5, 15, 30}, {5, 15, 30, 40}},
	residence.HallFuncSupport:    {{1, 2}, {1, 2, 4}, {3, 4, 5}, {3, 5, 7, 8}},
}

// hallLinks are the level links of a function on its edit page.
func hallLinks(spec hallFuncSpec, grade int) string {
	var levels []int
	if g, ok := hallLevelLinks[spec.typ]; ok {
		levels = g[min(max(grade, 0), 3)]
	} else {
		levels = []int{1, 2}
		if spec.typ == residence.HallFuncItemCreate {
			levels = []int{1, 2, 3}
		}
	}
	var b strings.Builder
	for _, l := range levels {
		label := "Level " + strconv.Itoa(l)
		if spec.percent {
			label = strconv.Itoa(l) + "%"
		}
		b.WriteString(`[<a action="bypass -h npc_%objectId%_manage ` + spec.group + ` edit_` + spec.key + " " + strconv.Itoa(l) + `">` + label + `</a>]`)
	}
	return b.String()
}

// feeDays is the payment period in days as Mobius prints it.
func feeDays(ratio int64) string { return strconv.FormatInt(ratio/1000/60/60/24, 10) }

// hallManageCommand is the "manage" command: the recovery, other and
// decoration pages, their edit and cancel pages, and the changes.
func (s *Server) hallManageCommand(p *Player, n *Npc, h *ClanHall, val string, rest []string) {
	if !p.HasClanAccess(social.AccessHallManageFunctions) {
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "not_authorized.htm"))
		return
	}
	switch val {
	case "recovery", "other":
	case "deco":
		if h.siege != nil {
			s.hallManagePage(p, n, h)
			return
		}
	case "back":
		s.showHallManagerPage(p, n)
		return
	default:
		s.hallManagePage(p, n, h)
		return
	}
	group := val
	if len(rest) == 0 {
		s.hallEditPage(p, n, h, group)
		return
	}
	if h.ownerID == 0 {
		p.SendMessage("This clan hall has no owner, you cannot change the configuration.")
		return
	}
	sub := strings.ToLower(rest[0])
	args := rest[1:]
	switch {
	case strings.HasSuffix(sub, "_cancel"):
		spec, ok := hallFuncSpecOf(group, strings.TrimSuffix(sub, "_cancel"))
		if !ok {
			s.hallEditPage(p, n, h, group)
			return
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "functions-cancel.htm").replace("%apply%", group+" "+spec.key+" 0"))
	case strings.HasPrefix(sub, "edit_"):
		spec, ok := hallFuncSpecOf(group, strings.TrimPrefix(sub, "edit_"))
		if !ok || len(args) == 0 {
			return
		}
		level, err := strconv.Atoi(args[0])
		if err != nil {
			return
		}
		t := s.feeTable(spec.typ)
		cost := residence.FunctionFee(spec.typ, level, t)
		if level == 0 { // Mobius's switch gives level 0 the default fee on the edit page
			cost = residence.FunctionFee(spec.typ, -1, t)
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "functions-apply.htm").replace("%name%", spec.name).
			replace("%cost%", strconv.Itoa(cost)+"</font>Adena /"+feeDays(t.Ratio)+" Day</font>)").
			replace("%use%", spec.use(level)).replace("%apply%", group+" "+spec.key+" "+strconv.Itoa(level)))
	default:
		spec, ok := hallFuncSpecOf(group, sub)
		if !ok {
			s.hallEditPage(p, n, h, group)
			return
		}
		if len(args) == 0 {
			return
		}
		level, err := strconv.Atoi(args[0])
		if err != nil {
			return
		}
		s.hallApplyFunction(p, n, h, spec, level, args[0])
	}
}

// hallManagePage is manage.htm, or manage_siegable.htm for a contestable
// hall.
func (s *Server) hallManagePage(p *Player, n *Npc, h *ClanHall) {
	name := "manage.htm"
	if h.siege != nil {
		name = "manage_siegable.htm"
	}
	s.hallManagerPage(p, n, s.hallManagerFile(p, n, name))
}

// hallApplyFunction rents, changes or cancels a function: the same level
// shows "already in use", level 0 cancels, the fee must come from the
// inventory ("low_adena.htm" otherwise). The members inside get the new
// AgitDecoInfo (revalidateDeco).
func (s *Server) hallApplyFunction(p *Player, n *Npc, h *ClanHall, spec hallFuncSpec, level int, raw string) {
	if f := h.functions[spec.typ]; f != nil && f.Level == level {
		shown := "Stage " + raw
		if spec.percent {
			shown = raw + "%"
		}
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "functions-used.htm").replace("%val%", shown))
		return
	}
	t := s.feeTable(spec.typ)
	fee := residence.FunctionFee(spec.typ, level, t)
	page := "functions-apply_confirmed.htm"
	if level == 0 {
		page = "functions-cancel_confirmed.htm"
	}
	if !s.updateHallFunction(p, h, spec.typ, level, fee) {
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "low_adena.htm"))
		return
	}
	if ch := s.ClanHallByOwner(p.Clan()); ch != nil && ch.siege == nil {
		p.Send(s.agitDecoPacket(ch))
	}
	s.hallManagerPage(p, n, s.hallManagerFile(p, n, page))
}

// hallEditPage is the overview of a group: edit_recovery.htm,
// edit_other.htm or deco.htm with each function's level, fee and next
// payment, and the links to change it.
func (s *Server) hallEditPage(p *Player, n *Npc, h *ClanHall, group string) {
	type slot struct {
		key, value, period, change string
	}
	page := map[string]string{"recovery": "edit_recovery.htm", "other": "edit_other.htm", "deco": "deco.htm"}[group]
	pg := s.hallManagerFile(p, n, page)
	// Mobius's placeholder names per function.
	names := map[string][3]string{
		"hp": {"%hp_recovery%", "%hp_period%", "%change_hp%"}, "mp": {"%mp_recovery%", "%mp_period%", "%change_mp%"},
		"exp": {"%exp_recovery%", "%exp_period%", "%change_exp%"}, "tele": {"%tele%", "%tele_period%", "%change_tele%"},
		"support": {"%support%", "%support_period%", "%change_support%"}, "item": {"%item%", "%item_period%", "%change_item%"},
		"curtains": {"%curtain%", "%curtain_period%", "%change_curtain%"}, "fixtures": {"%fixture%", "%fixture_period%", "%change_fixture%"},
	}
	for _, spec := range hallFuncSpecs {
		if spec.group != group {
			continue
		}
		ph := names[spec.key]
		links := hallLinks(spec, h.Grade())
		f := h.functions[spec.typ]
		if f == nil {
			pg.replace(ph[0], "none").replace(ph[1], "none").replace(ph[2], links)
			continue
		}
		t := s.feeTable(spec.typ)
		level := "Stage " + strconv.Itoa(f.Level)
		if spec.percent {
			level = strconv.Itoa(f.Level) + "%"
		}
		pg.replace(ph[0], level+"</font> (<font color=\"FFAABB\">"+strconv.Itoa(f.Lease)+"</font>Adena /"+feeDays(t.Ratio)+" Day)")
		pg.replace(ph[1], "Withdraw the fee for the next time at "+s.calendar.In(msTime(f.EndTime)).Format(hallDateFormat))
		pg.replace(ph[2], `[<a action="bypass -h npc_%objectId%_manage `+group+" "+spec.key+`_cancel">Deactivate</a>]`+links)
	}
	s.hallManagerPage(p, n, pg)
}

// hallSupportCommand is the "support" command: the manager casts the skill
// on p (summons p casts itself), with the NPC's MP unless
// clanHalls.mpBuffFree, then shows support-done.htm.
func (s *Server) hallSupportCommand(p *Player, n *Npc, h *ClanHall, val string, rest []string) {
	if p.CursedWeaponEquipped() {
		p.SendMessage("The wielder of a cursed weapon cannot receive outside heals or buffs")
		return
	}
	if val == "" {
		return
	}
	id, err := strconv.Atoi(val)
	level := 0
	if err == nil && len(rest) > 0 {
		level, err = strconv.Atoi(rest[0])
	}
	var sk *skills.Skill
	if err == nil {
		sk, _ = s.Skill(id, level)
	}
	if sk == nil {
		p.SendMessage("Invalid skill level, contact your admin!")
		return
	}
	if sk.HasEffectType(skills.TypeSummon) {
		// Mobius has the player cast a summon at once (doSimultaneousCast);
		// the support pages offer none.
		s.log.Debug("game: clan hall support summon skipped", append(p.logArgs(), "skill", id)...)
		return
	}
	mpCost := sk.MPConsume + sk.MPInitialConsume
	if n.CurMP < float64(mpCost) && !s.halls.cfg.MpBuffFree {
		s.hallManagerPage(p, n, s.hallManagerFile(p, n, "support-no_mana.htm").replace("%mp%", strconv.Itoa(int(n.CurMP))))
		return
	}
	s.NpcCast(n, p, sk)
	if f := h.functions[residence.HallFuncSupport]; f == nil || f.Level == 0 {
		return
	}
	s.hallManagerPage(p, n, s.hallManagerFile(p, n, "support-done.htm").replace("%mp%", strconv.Itoa(int(n.CurMP))))
}

// ----- doorman -----

// hallDoormanOwner is ClanHallDoorman.isOwnerClan.
func (s *Server) hallDoormanOwner(p *Player, h *ClanHall) bool {
	return p.Clan() != nil && h != nil && p.ClanID() == h.ownerID
}

// showHallDoormanPage is ClanHallDoorman.showChatWindow: the owner's
// members get doorman1.htm (doorman2.htm in a hall with pet evolution),
// others the owner's names, or emptyowner.htm for a free hall; nothing
// without a hall.
func (s *Server) showHallDoormanPage(p *Player, n *Npc) {
	p.SendActionFailed()
	h := s.hallOfDoorman(n)
	if h == nil {
		return
	}
	owner := h.Owner()
	var pg *htmlPage
	switch {
	case s.hallDoormanOwner(p, h):
		name := "clanHallDoorman/doorman1.htm"
		if hallHasEvolve(h) {
			name = "clanHallDoorman/doorman2.htm"
		}
		pg = s.pageFile(p, n.ID(), name).replace("%clanname%", owner.Name)
	case owner != nil && owner.Leader() != nil:
		pg = s.pageFile(p, n.ID(), "clanHallDoorman/doorman-no.htm").replace("%leadername%", owner.LeaderName()).replace("%clanname%", owner.Name)
	default:
		pg = s.pageFile(p, n.ID(), "clanHallDoorman/emptyowner.htm").replace("%hallname%", h.Name)
	}
	s.sendPage(p, pg.replaceObjectID(n.ID()))
}

// hallOfDoorman is ClanHallDoorman.getClanHall: an auctionable hall within
// 500 (a contestable hall's doormen have none in Mobius).
func (s *Server) hallOfDoorman(n *Npc) *ClanHall {
	h := s.npcClanHall(n)
	if h != nil && h.siege != nil {
		return nil
	}
	return h
}

// hallsWithEvolve are ClanHallDoorman.CH_WITH_EVOLVE.
var hallsWithEvolve = []int32{36, 37, 38, 39, 40, 41, 51, 52, 53, 54, 55, 56, 57}

func hallHasEvolve(h *ClanHall) bool { return slices.Contains(hallsWithEvolve, h.ID) }

// hallDoormanCommand handles the doorman's open_doors and close_doors for
// the owner clan (Doorman.onBypassFeedback with ClanHallDoorman's openDoors
// and closeDoors: every door of the hall). It reports whether it handled
// the command. Pet evolution needs the pet manager's evolve and is answered
// "evolve-no.htm".
func (s *Server) hallDoormanCommand(p *Player, n *Npc, command string) bool {
	h := s.hallOfDoorman(n)
	switch {
	case strings.HasPrefix(command, "evolve"):
		if h != nil && hallHasEvolve(h) && s.hallDoormanOwner(p, h) {
			s.sendPage(p, s.pageFile(p, n.ID(), "clanHallDoorman/evolve-no.htm"))
		}
		return true
	case strings.HasPrefix(command, "open_doors"), strings.HasPrefix(command, "close_doors"):
		if h == nil || !s.hallDoormanOwner(p, h) {
			return true
		}
		open := strings.HasPrefix(command, "open_doors")
		s.openCloseHallDoors(h, open)
		page := "clanHallDoorman/doorman-closed.htm"
		if open {
			page = "clanHallDoorman/doorman-opened.htm"
		}
		s.sendPage(p, s.pageFile(p, n.ID(), page).replaceObjectID(n.ID()))
		return true
	}
	return false
}
