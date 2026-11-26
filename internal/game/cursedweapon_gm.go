package game

import (
	"errors"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/cursedweapon"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The cursed weapon GM commands (Mobius AdminCursedWeapons): //cw_info,
// //cw_info_menu, //cw_add, //cw_remove, //cw_goto and //cw_reload.

// cwMsgSeparator is "========================" (Mobius EMPTY_3), after each
// weapon of //cw_info.
const cwMsgSeparator serverpackets.SystemMessageID = 490

// cwUsage is AdminCursedWeapons' answer to a command without a weapon.
const cwUsage = "Usage: //cw_remove|//cw_goto|//cw_add <itemid|name>"

// registerCursedWeaponGM registers the commands.
func (s *Server) registerCursedWeaponGM() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("cw_info", admin, s.gmCursedInfo),
		s.RegisterGMCommand("cw_info_menu", admin, s.gmCursedInfoMenu),
		s.RegisterGMCommand("cw_reload", admin, s.gmCursedReload),
		s.RegisterGMCommand("cw_add", admin, s.gmCursedWeapon(s.gmCursedAdd)),
		s.RegisterGMCommand("cw_remove", admin, s.gmCursedWeapon(func(_ *Player, w *cursedweapon.Weapon) { s.cursedEndOfLife(w) })),
		s.RegisterGMCommand("cw_goto", admin, s.gmCursedWeapon(s.gmCursedGoto)),
	)
}

// gmCursedInfo is //cw_info: each weapon's state as text.
func (s *Server) gmCursedInfo(p *Player, _ []string) {
	gmSysMessage(p, "====== Cursed Weapons: ======")
	now := s.Now()
	for _, w := range s.cursed.set.Weapons() {
		gmSysMessage(p, "> "+w.Def.Name+" ("+strconv.Itoa(int(w.ItemID()))+")")
		switch {
		case w.Activated:
			name := "null"
			if h := s.cursed.rt[w.ItemID()].player; h != nil {
				name = h.Name()
			}
			gmSysMessage(p, "  Player holding: "+name)
			gmSysMessage(p, "    Player karma: "+strconv.Itoa(int(w.PlayerKarma)))
			gmSysMessage(p, "    Time Remaining: "+strconv.FormatInt(w.TimeLeft(now)/60000, 10)+" min.")
			gmSysMessage(p, "    Kills : "+strconv.Itoa(int(w.NbKills)))
		case w.Dropped:
			gmSysMessage(p, "  Lying on the ground.")
			gmSysMessage(p, "    Time Remaining: "+strconv.FormatInt(w.TimeLeft(now)/60000, 10)+" min.")
			gmSysMessage(p, "    Kills : "+strconv.Itoa(int(w.NbKills)))
		default:
			gmSysMessage(p, "  Don't exist in the world.")
		}
		p.SendSystemMessage(cwMsgSeparator)
	}
}

// gmCursedInfoMenu is //cw_info_menu: cwinfo.htm with a table per weapon
// and its Remove, Go and Give to Target buttons.
func (s *Server) gmCursedInfoMenu(p *Player, _ []string) {
	var b strings.Builder
	now := s.Now()
	for _, w := range s.cursed.set.Weapons() {
		id := strconv.Itoa(int(w.ItemID()))
		minutes := strconv.FormatInt(w.TimeLeft(now)/60000, 10)
		b.WriteString("<table width=270><tr><td>Name:</td><td>" + w.Def.Name + "</td></tr>")
		switch {
		case w.Activated:
			name := "null"
			if h := s.cursed.rt[w.ItemID()].player; h != nil {
				name = h.Name()
			}
			b.WriteString("<tr><td>Weilder:</td><td>" + name + "</td></tr>")
			b.WriteString("<tr><td>Karma:</td><td>" + strconv.Itoa(int(w.PlayerKarma)) + "</td></tr>")
			b.WriteString("<tr><td>Kills:</td><td>" + strconv.Itoa(int(w.PlayerPKKills)) + "/" + strconv.Itoa(int(w.NbKills)) +
				"</td></tr><tr><td>Time remaining:</td><td>" + minutes + " min.</td></tr>")
			b.WriteString(`<tr><td><button value="Remove" action="bypass -h admin_cw_remove ` + id +
				`" width=73 height=15 back="sek.cbui94" fore="sek.cbui92"></td>`)
			b.WriteString(`<td><button value="Go" action="bypass -h admin_cw_goto ` + id +
				`" width=73 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr>`)
		case w.Dropped:
			b.WriteString("<tr><td>Position:</td><td>Lying on the ground</td></tr><tr><td>Time remaining:</td><td>" + minutes +
				" min.</td></tr><tr><td>Kills:</td><td>" + strconv.Itoa(int(w.NbKills)) + "</td></tr>")
			b.WriteString(`<tr><td><button value="Remove" action="bypass -h admin_cw_remove ` + id +
				`" width=95 height=21 back="bigbutton_over" fore="bigbutton"></td>`)
			b.WriteString(`<td><button value="Go" action="bypass -h admin_cw_goto ` + id +
				`" width=95 height=21 back="bigbutton_over" fore="bigbutton"></td></tr>`)
		default:
			b.WriteString(`<tr><td>Position:</td><td>Doesn't exist.</td></tr><tr><td><button value="Give to Target" action="bypass -h admin_cw_add ` + id +
				`" width=95 height=21 back="bigbutton_over" fore="bigbutton"></td><td></td></tr>`)
		}
		b.WriteString("</table><br>")
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/cwinfo.htm").replace("%cwinfo%", b.String()))
}

// gmCursedReload is //cw_reload. Mobius builds the weapons again from the
// data and the database and loses the ones on the ground; L2Go keeps the
// running state, as the data cannot change while the server runs.
func (s *Server) gmCursedReload(p *Player, _ []string) {
	gmSysMessage(p, "The cursed weapons keep their state; their data loads at startup.")
}

// gmCursedWeapon wraps a command that names a weapon by item ID or by name
// (underscores for spaces): without one it answers the usage and "Unknown
// cursed weapon ID.", as Mobius does.
func (s *Server) gmCursedWeapon(fn func(p *Player, w *cursedweapon.Weapon)) GMCommand {
	return func(p *Player, args []string) {
		if len(args) == 0 {
			gmSysMessage(p, cwUsage)
			gmSysMessage(p, "Unknown cursed weapon ID.")
			return
		}
		w, ok := s.cursed.set.Find(args[0])
		if !ok {
			gmSysMessage(p, "Unknown cursed weapon ID.")
			return
		}
		fn(p, w)
	}
}

// gmCursedAdd is //cw_add <weapon>: the GM's target player, or the GM,
// gets the weapon (and becomes its holder), whose life starts now. A
// weapon in the world is refused.
func (s *Server) gmCursedAdd(p *Player, w *cursedweapon.Weapon) {
	if w.Active() {
		gmSysMessage(p, "This cursed weapon is already active.")
		return
	}
	to := p
	if t, ok := p.Target.(*Player); ok && t != nil && t.InGame() {
		to = t
	}
	if _, err := s.AddItem(to, w.ItemID(), 1, -1, true, false); err != nil {
		s.log.Warn("game: //cw_add cannot give the weapon", append(to.logArgs(), "item", w.ItemID(), "err", err)...)
		return
	}
	// Mobius sets the life and restarts the timer whatever activate did; a
	// weapon that ended at once (its new holder had the other one) stays
	// ended here.
	if !w.Activated {
		return
	}
	w.StartLife(s.Now())
	s.startCursedTimer(w)
	s.cursedSave(w, to)
}

// gmCursedGoto is //cw_goto <weapon>: the GM goes to the holder or to the
// weapon on the ground (teleToLocation with a random offset).
func (s *Server) gmCursedGoto(p *Player, w *cursedweapon.Weapon) {
	pos, ok := s.cursedPosition(w)
	if !ok {
		p.SendMessage(w.Def.Name + " isn't in the World.")
		return
	}
	s.teleportAround(p, data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}, maxOffsetOnTeleport)
}
