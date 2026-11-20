package game

import (
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// The grand boss GM commands (Mobius AdminGrandBoss): //grandboss shows the
// grand boss page, //grandboss <id> a boss's status page; //grandboss_skip,
// //grandboss_respawn, //grandboss_minions and //grandboss_abort send their
// event to the Antharas or Baium script.

// The grand bosses of the GM page and the NoRestartZones that count their
// players (AdminGrandBoss).
const (
	gmAntharas      = 29019
	gmAntharasZone  = 70050
	gmValakas       = 29028
	gmValakasZone   = 70052
	gmBaium         = 29020
	gmBaiumZone     = 70051
	gmQueenAnt      = 29001
	gmOrfen         = 29014
	gmCore          = 29006
	gmZaken         = 29022
	gmZakenZone     = 70053
	gmFrintezza     = 29045
	gmFrintezzaZone = 70054

	// gmRespawnFormat is RESPAWN_FORMAT ("yyyy-MM-dd HH:mm:ss").
	gmRespawnFormat = "2006-01-02 15:04:05"
)

// gmGrandBossPage is what manageHtml shows for each boss.
type gmGrandBossPage struct {
	page string
	zone int // the NoRestartZone counted, 0 for none
}

var gmGrandBossPages = map[int]gmGrandBossPage{
	gmAntharas:  {"grandboss/grandboss_antharas.htm", gmAntharasZone},
	gmValakas:   {"grandboss/grandboss_valakas.htm", gmValakasZone},
	gmBaium:     {"grandboss/grandboss_baium.htm", gmBaiumZone},
	gmQueenAnt:  {"grandboss/grandboss_queenant.htm", 0},
	gmOrfen:     {"grandboss/grandboss_orfen.htm", 0},
	gmCore:      {"grandboss/grandboss_core.htm", 0},
	gmZaken:     {"grandboss/grandboss_zaken.htm", gmZakenZone},
	gmFrintezza: {"grandboss/grandboss_frintezza.htm", gmFrintezzaZone},
}

// registerGrandBossGM registers the commands.
func (s *Server) registerGrandBossGM() error {
	admin := storage.AccessLevelAdmin
	event := func(usage string, events map[int]string, page bool) GMCommand {
		return func(p *Player, args []string) { s.gmGrandBossEvent(p, args, usage, events, page) }
	}
	return errors.Join(
		s.RegisterGMCommand("grandboss", admin, s.gmGrandBoss),
		s.RegisterGMCommand("grandboss_skip", admin, event("//grandboss_skip Id",
			map[int]string{gmAntharas: "SKIP_WAITING"}, true)),
		s.RegisterGMCommand("grandboss_respawn", admin, event("//grandboss_respawn Id",
			map[int]string{gmAntharas: "RESPAWN_ANTHARAS", gmBaium: "RESPAWN_BAIUM"}, true)),
		s.RegisterGMCommand("grandboss_minions", admin, event("//grandboss_minions Id",
			map[int]string{gmAntharas: "DESPAWN_MINIONS", gmBaium: "DESPAWN_MINIONS"}, false)),
		s.RegisterGMCommand("grandboss_abort", admin, event("//grandboss_abort Id",
			map[int]string{gmAntharas: "ABORT_FIGHT", gmBaium: "ABORT_FIGHT"}, true)),
	)
}

// gmGrandBoss is //grandboss [id].
func (s *Server) gmGrandBoss(p *Player, args []string) {
	if len(args) == 0 {
		s.ShowAdminPage(p, "grandboss/grandboss.htm")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		gmSysMessage(p, "Wrong ID!")
		return
	}
	s.gmGrandBossPage(p, id)
}

// gmGrandBossEvent sends the event of the boss to its script (Mobius
// notifyEvent(event, null, player)) and shows the boss page again when page
// is set.
func (s *Server) gmGrandBossEvent(p *Player, args []string, usage string, events map[int]string, page bool) {
	if len(args) == 0 {
		gmSysMessage(p, "Usage: "+usage)
		return
	}
	id, err := strconv.Atoi(args[0])
	ev, ok := events[id]
	if err != nil || !ok {
		gmSysMessage(p, "Wrong ID!")
		return
	}
	script := "Antharas"
	if id == gmBaium {
		script = "Baium"
	}
	if s.scripts == nil || !s.scripts.HasScript(script) {
		gmSysMessage(p, "The "+script+" script is not loaded.")
		return
	}
	s.scriptEventReply(p, nil, script, ev)
	if page {
		s.gmGrandBossPage(p, id)
	}
}

// gmGrandBossPage is AdminGrandBoss.manageHtml: the boss's status (green
// Alive, yellow Waiting, orange In Fight, red Dead; Queen Ant, Orfen, Core
// and Zaken know only Alive and Dead), its respawn time while dead, and the
// players in its zone.
func (s *Server) gmGrandBossPage(p *Player, id int) {
	pg, ok := gmGrandBossPages[id]
	if !ok {
		gmSysMessage(p, "Wrong ID!")
		return
	}
	status, _ := s.GrandBossStatus(id)
	var text, color string
	dead := 1
	if slices.Contains([]int{gmAntharas, gmValakas, gmBaium, gmFrintezza}, id) {
		dead = 3
		switch status {
		case 0:
			text, color = "Alive", "00FF00"
		case 1:
			text, color = "Waiting", "FFFF00"
		case 2:
			text, color = "In Fight", "FF9900"
		case 3:
			text, color = "Dead", "FF0000"
		}
	} else {
		switch status {
		case 0:
			text, color = "Alive", "00FF00"
		case 1:
			text, color = "Dead", "FF0000"
		}
	}
	respawn := "Already respawned!"
	if status == dead {
		info, _ := s.GrandBossInfoOf(id)
		at := time.UnixMilli(0)
		if !info.RespawnTime.IsZero() {
			at = info.RespawnTime
		}
		respawn = at.In(time.Local).Format(gmRespawnFormat)
	}
	inside := "Zone not found!"
	if pg.zone != 0 && s.worldData != nil {
		if _, found := s.worldData.Zone(pg.zone); found {
			n := 0
			for q := range s.Players() {
				if q.InGame() && q.Zones.In(pg.zone) {
					n++
				}
			}
			inside = strconv.Itoa(n)
		}
	}
	page := s.adminPageFile(p, pg.page)
	page.replace("%bossStatus%", text)
	page.replace("%bossColor%", color)
	page.replace("%respawnTime%", respawn)
	page.replace("%playersInside%", inside)
	s.sendPage(p, page)
}
