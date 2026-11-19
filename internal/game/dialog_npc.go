package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// The Mobius Player.ini settings for players with karma (Mobius defaults):
// they may shop and use warehouses, but gatekeepers send them away.
const (
	karmaCanShop         = true  // AltKarmaPlayerCanShop
	karmaCanUseGK        = false // AltKarmaPlayerCanUseGK
	karmaCanUseWarehouse = true  // AltKarmaPlayerCanUseWareHouse
)

// Mobius page folders that are not NPC type folders.
const (
	sevenSignsHTML = "seven_signs/" // SevenSigns.SEVEN_SIGNS_HTML_PATH
)

// npcFolders are the page folders of the Mobius NPC classes whose
// getHtmlPath names one (entity/actor/instance), most specific class first:
// a Teleporter is also a Merchant. Their pages are taken as they are named;
// a missing one shows Mobius's missing-page text. Other NPCs use default/
// when the page exists, otherwise npcdefault.htm.
var npcFolders = []struct{ class, folder string }{
	{"Teleporter", "teleporter"},
	{"DungeonGatekeeper", "teleporter"},
	{"PetManager", "petmanager"},
	{"Fisherman", "fisherman"},
	{"Merchant", "merchant"},
	{"Warehouse", "warehouse"},
	{"Trainer", "trainer"},
	{"VillageMaster", "villagemaster"},
	{"Guard", "guard"},
	{"Adventurer", "adventurer_guildsman"},
	{"SchemeBuffer", "mods/SchemeBuffer"},
}

// npcTypeIs reports whether the Mobius NPC class typ is class or extends
// it. Without world data only the class itself matches.
func npcTypeIs(w *data.World, typ, class string) bool {
	if w == nil {
		return typ == class
	}
	return w.NpcTypeIs(typ, class)
}

// npcIs reports whether n is an instance of the Mobius NPC class class.
func (s *Server) npcIs(n *Npc, class string) bool {
	return n.Template != nil && npcTypeIs(s.worldData, n.Template.Type, class)
}

// npcPagePath returns the page of NPC template t for page number value
// (Mobius Npc.getHtmlPath and the overrides of its subclasses):
// "<folder>/<npcId>.htm", or "<folder>/<npcId>-<value>.htm" for a page
// other than 0. exists reports whether a page of the HTML folder exists.
func npcPagePath(w *data.World, t *data.NpcTemplate, value int, exists func(rel string) bool) string {
	name := strconv.Itoa(t.ID)
	if value != 0 {
		name += "-" + strconv.Itoa(value)
	}
	for _, f := range npcFolders {
		if npcTypeIs(w, t.Type, f.class) {
			return f.folder + "/" + name + ".htm"
		}
	}
	if page := "default/" + name + ".htm"; exists(page) {
		return page
	}
	return "npcdefault.htm"
}

// npcInteract opens n's dialog for p: the second click on a targeted NPC in
// range (registered with OnNpcInteract). A script that owns the NPC's first
// talk answers first; otherwise, or when it returns nil, the NPC's own first
// page shows (Mobius NpcClick).
func (s *Server) npcInteract(p *Player, n *Npc) {
	if n == nil || n.Template == nil {
		return
	}
	p.LastNpc = n.ID()
	if s.scripts != nil && len(s.scripts.TalkScripts(n.Template.ID)) > 0 {
		s.dialogState(p).lastQuestNpc = n.ID()
	}
	if s.scripts != nil && s.scripts.HasFirstTalk(n.Template.ID) {
		s.scriptFirstTalk(p, n)
		return
	}
	s.showFirstPage(p, n)
}

// showFirstPage shows n's first page (Mobius Npc.showChatWindow(Player)
// with the overrides of its subclasses). NPCs of residences and Seven
// Signs show the page Mobius shows a player without a clan, castle or
// cabal, since those do not exist yet.
func (s *Server) showFirstPage(p *Player, n *Npc) {
	t := n.Template
	id := n.ID()
	switch {
	case s.npcIs(n, "Teleporter"):
		if s.onCastleGround(n.Position()) {
			// A castle gatekeeper: only the clan owning the castle may use
			// it, and nobody owns a castle yet.
			s.sendPage(p, s.pageFile(p, id, "teleporter/castleteleporter-no.htm").replaceObjectID(id).replace("%npcname%", t.Name))
			return
		}
		s.showNpcPage(p, n, 0)
	case s.npcIs(n, "PetManager"):
		s.sendPage(p, s.pageFile(p, id, "petmanager/"+strconv.Itoa(t.ID)+".htm").replaceObjectID(id).replace("%npcname%", t.Name))
	case s.npcIs(n, "ClanHallDoorman"):
		// Mobius shows nothing without the clan hall's data.
		p.SendActionFailed()
	case s.npcIs(n, "CastleDoorman"):
		p.SendActionFailed()
		s.sendPage(p, s.pageFile(p, id, "doorman/"+strconv.Itoa(t.ID)+"-no.htm").replaceObjectID(id))
	case s.npcIs(n, "Doorman"):
		p.SendActionFailed()
		s.sendPage(p, s.pageFile(p, id, "doorman/"+strconv.Itoa(t.ID)+".htm").replaceObjectID(id))
	case s.npcIs(n, "ClanHallManager"):
		p.SendActionFailed()
		s.sendPage(p, s.pageFile(p, id, "clanHallManager/chamberlain-no.htm").replaceObjectID(id).replace("%npcId%", strconv.Itoa(t.ID)))
	case s.npcIs(n, "Auctioneer"):
		s.sendPage(p, s.pageFile(p, id, "auction/auction.htm").replaceObjectID(id).replace("%npcId%", strconv.Itoa(t.ID)).replace("%npcname%", t.Name))
	case s.npcIs(n, "DawnPriest"):
		p.SendActionFailed()
		s.sendPage(p, s.pageFile(p, id, sevenSignsHTML+"dawn_priest_1a.htm").replaceObjectID(id))
	case s.npcIs(n, "DuskPriest"):
		p.SendActionFailed()
		s.sendPage(p, s.pageFile(p, id, sevenSignsHTML+"dusk_priest_1a.htm").replaceObjectID(id))
	default:
		s.showNpcPage(p, n, 0)
	}
}

// showNpcPage shows page value of n (Mobius Npc.showChatWindow(Player,
// int)): the page of its getHtmlPath with %npcname% and %objectId% filled,
// then ActionFailed. An NPC that cannot be talked to only sends
// ActionFailed; a player with karma gets the gatekeeper's "-pk" page where
// one exists; a few NPCs have fixed pages by ID.
func (s *Server) showNpcPage(p *Player, n *Npc, value int) {
	t := n.Template
	if !t.Status.Talkable {
		p.SendActionFailed()
		return
	}
	if s.npcIs(n, "BroadcastingTower") {
		s.showTowerPage(p, n, value)
		return
	}
	if p.Karma() > 0 {
		var folder string
		switch {
		case !karmaCanShop && s.npcIs(n, "Merchant"):
			folder = "merchant"
		case !karmaCanUseGK && s.npcIs(n, "Teleporter"):
			folder = "teleporter"
		case !karmaCanUseWarehouse && s.npcIs(n, "Warehouse"):
			folder = "warehouse"
		case !karmaCanShop && s.npcIs(n, "Fisherman"):
			folder = "fisherman"
		}
		if folder != "" && s.showPkPage(p, n, folder) {
			return
		}
	}
	if strings.EqualFold(t.Type, "Auctioneer") && value == 0 {
		return
	}

	var file string
	switch npcID := t.ID; {
	case npcID >= 31127 && npcID <= 31131: // Dawn Festival Guides
		file = sevenSignsHTML + "festival/dawn_guide.htm"
	case npcID >= 31137 && npcID <= 31141: // Dusk Festival Guides
		file = sevenSignsHTML + "festival/dusk_guide.htm"
	case npcID == 31092: // Black Marketeer of Mammon
		file = sevenSignsHTML + "blkmrkt_1.htm"
	case npcID == 31113: // Merchant of Mammon (StrictSevenSigns is off)
		file = sevenSignsHTML + "mammmerch_1.htm"
	case npcID == 31126: // Blacksmith of Mammon
		file = sevenSignsHTML + "mammblack_1.htm"
	case npcID >= 31132 && npcID <= 31136, npcID >= 31142 && npcID <= 31146: // Festival Witches
		file = sevenSignsHTML + "festival/festival_witch.htm"
	case npcID >= 31865 && npcID <= 31918: // Guardians of the Border (Dimensional Rift)
		file = sevenSignsHTML + "rift/GuardianOfBorder.htm"
		if value != 0 {
			file = sevenSignsHTML + "rift/GuardianOfBorder-" + strconv.Itoa(value) + ".htm"
		}
	case npcID >= 31093 && npcID <= 31094, npcID >= 31172 && npcID <= 31201, npcID >= 31239 && npcID <= 31254:
		// Mobius shows these NPCs nothing.
		return
	default:
		// The Olympiad managers show their noble and hero pages, and
		// Blacksmith Pinter (30298) his academy page, only to nobles,
		// heroes and academy members, who do not exist yet.
		file = npcPagePath(s.worldData, t, value, s.dialog.html.exists)
	}
	s.sendPage(p, s.pageFile(p, n.ID(), file).replace("%npcname%", t.Name).replaceObjectID(n.ID()))
	p.SendActionFailed()
}

// showPkPage shows the "<folder>/<npcId>-pk.htm" page of n to a player with
// karma when it exists (Mobius showPkDenyChatWindow).
func (s *Server) showPkPage(p *Player, n *Npc, folder string) bool {
	content, ok := s.htmlFile(p, folder+"/"+strconv.Itoa(n.Template.ID)+"-pk.htm")
	if !ok {
		return false
	}
	s.insertObjectIDAndShow(p, n, content)
	p.SendActionFailed()
	return true
}

// insertObjectIDAndShow shows content as n's page with %objectId% filled
// first (Mobius insertObjectIdAndShowChatWindow).
func (s *Server) insertObjectIDAndShow(p *Player, n *Npc, content string) {
	s.sendPage(p, newPage(n.ID(), strings.ReplaceAll(content, "%objectId%", strconv.Itoa(int(n.ID())))))
}

// broadcastingTowerOracle are the places of the broadcasting towers that
// look into the Oracle of Dawn and Dusk (Mobius BroadcastingTower).
var broadcastingTowerOracle = []world.Position{{X: -79884, Y: 86529}, {X: -78858, Y: 111358}, {X: -76973, Y: 87136}, {X: -75850, Y: 111968}}

// showTowerPage shows page value of a broadcasting tower (Mobius
// BroadcastingTower.showChatWindow).
func (s *Server) showTowerPage(p *Player, n *Npc, value int) {
	name := "observation/" + strconv.Itoa(n.Template.ID)
	pos := n.Position()
	for _, o := range broadcastingTowerOracle {
		if pos.Distance2D(o) <= 50 {
			name += "-Oracle"
			break
		}
	}
	if value != 0 {
		name += "-" + strconv.Itoa(value)
	}
	s.sendPage(p, s.pageFile(p, n.ID(), name+".htm").replaceObjectID(n.ID()))
}

// notYetHTML is the page of the NPC services that come in later milestones
// (decision N6): shops, warehouses, skill trainers and the like.
const (
	notYetHTML = `<html><body>%npcname%:<br>I am sorry, this service is not available yet.<br><br>` +
		`<a action="bypass -h npc_%objectId%_Chat 0">Back</a></body></html>`
	notYetPlainHTML = `<html><body>This service is not available yet.</body></html>`
)

// showNotYet answers a service that does not exist yet with a small page
// (decision N6). With an NPC, the page links back to its first page.
func (s *Server) showNotYet(p *Player, n *Npc) {
	if n == nil {
		s.sendPage(p, newPage(0, notYetPlainHTML))
		return
	}
	s.sendPage(p, newPage(n.ID(), notYetHTML).replace("%npcname%", n.Template.Name).replaceObjectID(n.ID()))
}
