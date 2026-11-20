package game

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The admin panel (Mobius //admin): the pages of the html/admin folder and
// the commands that move between them (Mobius AdminAdmin, AdminHtml,
// AdminHelp, AdminMenu, AdminTeleport, AdminSpawn, AdminCreateItem,
// AdminShop and AdminEffects). The pages' buttons are "bypass admin_<cmd>"
// links, which handleBypass runs as GM commands; a button whose command the
// server does not have yet answers "The command '...' does not exist!", as
// Mobius does.
func init() { registerFeature("admin", setupAdmin) }

const (
	// adminSysMsgCannotUseOnYourself is "You cannot use this on yourself."
	// (Mobius YOU_CANNOT_USE_THIS_ON_YOURSELF).
	adminSysMsgCannotUseOnYourself serverpackets.SystemMessageID = 51
	// adminSysMsgNotCurrentlyOnline is "That player is not currently
	// online." (Mobius THAT_PLAYER_IS_NOT_CURRENTLY_ONLINE).
	adminSysMsgNotCurrentlyOnline serverpackets.SystemMessageID = 161
	// adminGoOffset is how far //gonorth and the others move by default.
	adminGoOffset = 150
	// adminHTMLFolder is the folder of the panel's pages, relative to the
	// HTML folder.
	adminHTMLFolder = "admin/"
)

// adminMenus are the pages of //admin1 to //admin7 (Mobius
// AdminAdmin.showMainPage); //admin and any other number show the first.
var adminMenus = [...]string{"main", "main", "game", "effects", "server", "mods", "char", "gm"}

// adminPages are the commands that only show a page of the admin folder
// (Mobius AdminHelp, AdminMenu, AdminTeleport, AdminCreateItem, AdminSpawn
// and AdminShop show it when the command has no arguments, and do nothing
// otherwise).
var adminPages = map[string]string{
	"help":             "help.htm",
	"char_manage":      "charmanage.htm",
	"show_moves":       "teleports.htm",
	"show_moves_other": "teleports/OtherLocations.htm",
	"tele":             "move.htm",
	"itemcreate":       "itemcreation.htm",
	"show_spawns":      "spawns.htm",
	"show_npcs":        "npcs.htm",
	"gmshop":           "gmshops.htm",
}

// adminGoDirections are the moves of //gonorth and the others (Mobius
// AdminTeleport admin_go<direction>), as unit steps on x, y and z.
var adminGoDirections = map[string][3]int32{
	"north": {0, -1, 0},
	"south": {0, 1, 0},
	"east":  {1, 0, 0},
	"west":  {-1, 0, 0},
	"up":    {0, 0, 1},
	"down":  {0, 0, -1},
}

// setupAdmin registers the admin panel's commands.
func setupAdmin(s *Server) error {
	admin := storage.AccessLevelAdmin
	errs := []error{
		s.RegisterGMCommand("admin", admin, s.gmAdminMenu(0)),
		s.RegisterGMCommand("html", admin, s.gmAdminHTML),
		s.RegisterGMCommand("loadhtml", admin, s.gmAdminLoadHTML),
		s.RegisterGMCommand("invis_menu", admin, s.gmAdminInvisMenu),
		s.RegisterGMCommand("kill_menu", admin, s.gmAdminKillMenu),
		s.RegisterGMCommand("goto_char_menu", admin, s.gmAdminGotoChar),
		s.RegisterGMCommand("recall_char_menu", admin, s.gmAdminRecallChar),
		s.RegisterGMCommand("teleport_character_to_menu", admin, s.gmAdminTeleportCharTo),
		// Mobius admin_spawn_monster is admin_spawn under another name.
		s.RegisterGMCommand("spawn_monster", admin, s.gmSpawn),
	}
	for i := 1; i < len(adminMenus); i++ {
		errs = append(errs, s.RegisterGMCommand("admin"+strconv.Itoa(i), admin, s.gmAdminMenu(i)))
	}
	for name, page := range adminPages {
		errs = append(errs, s.RegisterGMCommand(name, admin, s.gmAdminPage(page)))
	}
	for dir, step := range adminGoDirections {
		errs = append(errs, s.RegisterGMCommand("go"+dir, admin, s.gmAdminGo(step)))
	}
	return errors.Join(errs...)
}

// ShowAdminPage shows p the page rel of the admin folder ("main_menu.htm"),
// not bound to an NPC (Mobius AdminHtml.showAdminHtml). A missing page
// shows Mobius's "My text is missing" text. Loop-only.
func (s *Server) ShowAdminPage(p *Player, rel string) {
	full := adminHTMLFolder + rel
	content, ok := s.htmlFile(p, full)
	if !ok {
		s.log.Warn("game: missing admin page", "page", htmlPrefix+full)
		content = adminMissingPage(htmlPrefix + full)
	}
	if rel == "main_menu.htm" {
		content = adminAddLevelButton(content) // admin_level.go
	}
	s.sendPage(p, newPage(0, content))
}

// adminMissingPage is AdminHtml's page for a file it cannot read.
func adminMissingPage(path string) string {
	return "<html><body>My text is missing:<br>" + path + "</body></html>"
}

// gmAdminMenu is //admin and //admin1 to //admin7 (Mobius AdminAdmin): the
// panel's main page and its menus.
func (s *Server) gmAdminMenu(n int) GMCommand {
	return func(p *Player, _ []string) {
		s.ShowAdminPage(p, adminMenus[n]+"_menu.htm")
	}
}

// gmAdminPage shows page when the command comes without arguments.
func (s *Server) gmAdminPage(page string) GMCommand {
	return func(p *Player, args []string) {
		if len(args) == 0 {
			s.ShowAdminPage(p, page)
		}
	}
}

// gmAdminHTML is //html <path> (Mobius AdminHtml): the page path of the
// admin folder.
func (s *Server) gmAdminHTML(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Usage: //html path")
		return
	}
	s.ShowAdminPage(p, args[0])
}

// gmAdminLoadHTML is //loadhtml <path> (Mobius AdminHtml): the page at path
// below the data folder ("data/html/admin/main_menu.htm"), read again from
// the disk and kept for the next uses, so that an edited page shows without
// a restart.
func (s *Server) gmAdminLoadHTML(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Usage: //loadhtml path")
		return
	}
	content, ok := "", false
	if rel, isHTML := strings.CutPrefix(args[0], htmlPrefix); isHTML {
		content, ok = s.dialog.html.reload(rel)
	}
	if !ok {
		content = adminMissingPage(args[0])
	}
	s.sendPage(p, newPage(0, content))
}

// reload reads the page at rel from the disk again and keeps it, as Mobius
// HtmCache.loadFile does; a page that cannot be read is left as it was.
func (l *htmlLoader) reload(rel string) (string, bool) {
	p, ok := cleanHTMLPath(rel)
	if !ok {
		return "", false
	}
	content, ok := readHTMLFile(filepath.Join(l.dir, filepath.FromSlash(p)))
	if ok {
		l.pages[p] = htmlEntry{content: content, ok: true}
	}
	return content, ok
}

// gmAdminInvisMenu is //invis_menu (Mobius AdminEffects): the GM's
// invisibility is switched, then the GM menu shows again.
func (s *Server) gmAdminInvisMenu(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	if p.Invisible() {
		s.gmVisible(p, nil)
	} else {
		s.gmInvisible(p, nil)
	}
	s.ShowAdminPage(p, "gm_menu.htm")
}

// gmAdminKillMenu is //kill_menu (Mobius AdminMenu.handleKill): the GM's
// target creature dies like with //kill, then the character page (after a
// player) or the main page shows. Mobius does nothing when the command has
// arguments.
func (s *Server) gmAdminKillMenu(p *Player, args []string) {
	if len(args) > 0 {
		return
	}
	page := "main_menu.htm"
	if c, ok := p.Target.(Creature); ok && c != nil {
		s.gmKillCreature(p, c)
		if _, isPlayer := c.(*Player); isPlayer {
			page = "charmanage.htm"
		}
	} else {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
	}
	s.ShowAdminPage(p, page)
}

// gmAdminGotoChar is //goto_char_menu <name> (Mobius AdminMenu): the GM
// teleports to that player, then the character page shows.
func (s *Server) gmAdminGotoChar(p *Player, args []string) {
	t, ok := s.PlayerByName(strings.Join(args, " "))
	if !ok {
		p.SendSystemMessage(adminSysMsgNotCurrentlyOnline)
		return
	}
	if t == p {
		p.SendSystemMessage(adminSysMsgCannotUseOnYourself)
	} else {
		pos := t.Position()
		s.teleportAround(p, data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}, maxOffsetOnTeleport)
		p.SendMessage("You're teleporting yourself to character " + t.Name())
	}
	s.ShowAdminPage(p, "charmanage.htm")
}

// gmAdminRecallChar is //recall_char_menu <name> (Mobius AdminMenu): the
// player (if online) comes to the GM, the character page shows and the GM's
// target is dropped.
func (s *Server) gmAdminRecallChar(p *Player, args []string) {
	t, _ := s.PlayerByName(strings.Join(args, " "))
	s.adminTeleportChar(p, t, p.Position())
	s.SetTarget(p, nil)
}

// gmAdminTeleportCharTo is //teleport_character_to_menu <name> <x> <y> <z>
// (Mobius AdminMenu): that player goes to the place, then the character
// page shows.
func (s *Server) gmAdminTeleportCharTo(p *Player, args []string) {
	if len(args) == 4 {
		var c [3]int32
		valid := true
		for i := range c {
			v, err := strconv.ParseInt(args[i+1], 10, 32)
			if err != nil {
				valid = false
				break
			}
			c[i] = int32(v)
		}
		if t, ok := s.PlayerByName(args[0]); ok && valid {
			s.adminTeleportChar(p, t, world.Position{X: c[0], Y: c[1], Z: c[2]})
			return
		}
	}
	s.ShowAdminPage(p, "charmanage.htm")
}

// adminTeleportChar is AdminMenu.teleportCharacter: t (when not nil) is told
// and teleported near pos, then the GM sees the character page.
func (s *Server) adminTeleportChar(gm, t *Player, pos world.Position) {
	if t != nil {
		t.SendMessage("Admin is teleporting you.")
		s.teleportAround(t, data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}, maxOffsetOnTeleport)
	}
	s.ShowAdminPage(gm, "charmanage.htm")
}

// gmAdminGo is //gonorth [offset] and the other directions (Mobius
// AdminTeleport admin_go<direction>): the GM teleports offset units (150 by
// default) that way, then the teleport menu shows.
func (s *Server) gmAdminGo(step [3]int32) GMCommand {
	return func(p *Player, args []string) {
		offset := int32(adminGoOffset)
		if len(args) > 0 {
			v, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				p.SendMessage("Usage: //go<north|south|east|west|up|down> [offset] (default 150)")
				return
			}
			offset = int32(v)
		}
		pos := p.Position()
		pos.X += step[0] * offset
		pos.Y += step[1] * offset
		pos.Z += step[2] * offset
		pos.Heading = 0
		if err := s.Teleport(p, pos); errors.Is(err, world.ErrOutOfWorld) {
			p.SendMessage("Those coordinates are outside the world.")
		}
		s.ShowAdminPage(p, "move.htm")
	}
}
