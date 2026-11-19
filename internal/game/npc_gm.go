package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

const (
	// npcGMSpawnMaxCount bounds //spawn's count, so that a typo does not
	// stall the world (Mobius has no bound).
	npcGMSpawnMaxCount = 1000
	// npcSysMsgTargetNotFound is "Your target cannot be found." (Mobius
	// YOUR_TARGET_CANNOT_BE_FOUND), the answer to a //spawn that fails.
	npcSysMsgTargetNotFound serverpackets.SystemMessageID = 50
)

// setupNpcGM registers the NPC GM commands.
func (s *Server) setupNpcGM() error {
	return errors.Join(
		s.RegisterGMCommand("spawn", storage.AccessLevelAdmin, s.gmSpawn),
		s.RegisterGMCommand("delete", storage.AccessLevelAdmin, s.gmDelete),
		s.RegisterGMCommand("respawn", storage.AccessLevelAdmin, s.gmRespawn),
		s.RegisterGMCommand("npcinfo", storage.AccessLevelAdmin, s.gmNpcInfo),
	)
}

// gmSpawn is //spawn <npcId|name> [count] (Mobius admin_spawn): count NPCs
// of the template (1 by default) at the GM's target, or at the GM, facing
// the way the GM faces. A name may have several words; underscores stand
// for spaces. A GM spawn never respawns and is not written to the spawn
// lists.
func (s *Server) gmSpawn(p *Player, args []string) {
	const usage = "Usage: //spawn <npcId|name> [count]"
	if len(args) == 0 {
		p.SendMessage(usage)
		return
	}
	t, rest := s.gmSpawnTemplate(args)
	if t == nil {
		p.SendSystemMessage(npcSysMsgTargetNotFound)
		return
	}
	count := 1
	if len(rest) > 0 {
		c, err := strconv.Atoi(rest[0])
		if err != nil || c < 1 {
			p.SendMessage(usage)
			return
		}
		count = min(c, npcGMSpawnMaxCount)
	}
	if _, err := s.spawnableTemplate(t.ID); err != nil {
		p.SendSystemMessage(npcSysMsgTargetNotFound)
		return
	}

	var at world.Object = p
	if p.Target != nil {
		at = p.Target
	}
	pos := at.Position()
	pos.Heading = p.Position().Heading
	for range count {
		sp := &npcSpawn{tmpl: t, point: pos, gm: true, index: -1}
		s.addSpawnSlot(sp)
		if s.spawnSlot(sp) == nil {
			s.deleteSpawn(sp)
			p.SendSystemMessage(npcSysMsgTargetNotFound)
			return
		}
	}
	p.SendMessage(fmt.Sprintf("Created %s on %d", t.Name, at.ID()))
}

// gmSpawnTemplate finds the template //spawn names and returns the words
// after its name: an NPC ID, or the longest run of words that is a
// template's name, ignoring case (Mobius getTemplateByName).
func (s *Server) gmSpawnTemplate(args []string) (*data.NpcTemplate, []string) {
	if s.worldData == nil {
		return nil, nil
	}
	if id, err := strconv.Atoi(args[0]); err == nil {
		t, _ := s.worldData.Npc(id)
		return t, args[1:]
	}
	for i := len(args); i >= 1; i-- {
		name := strings.ReplaceAll(strings.Join(args[:i], " "), "_", " ")
		for j := range s.worldData.Npcs {
			if t := &s.worldData.Npcs[j]; strings.EqualFold(t.Name, name) {
				return t, args[i:]
			}
		}
	}
	return nil, nil
}

// gmDelete is //delete [range] (Mobius admin_delete): the targeted NPC, or
// every NPC within range of the GM, leaves the world for good. An NPC of a
// spawn takes its spawn with it, so nothing respawns; for a territory spawn
// that is every spawn of the territory, as Mobius deletes the whole group.
func (s *Server) gmDelete(p *Player, args []string) {
	if len(args) > 0 {
		if r, err := strconv.Atoi(args[0]); err == nil && r > 0 {
			var inRange []*Npc
			for o := range s.world.VisibleInRange(p, int32(min(r, 1<<20))) { //nolint:gosec // clamped
				if n, ok := o.(*Npc); ok {
					inRange = append(inRange, n)
				}
			}
			for _, n := range inRange {
				if n.InWorld() { // a group delete may have taken it already
					s.gmDeleteNpc(p, n)
				}
			}
			return
		}
	}
	n, ok := p.Target.(*Npc)
	if !ok {
		p.SendMessage("Incorrect target.")
		return
	}
	s.gmDeleteNpc(p, n)
}

// gmDeleteNpc deletes n for good, with its spawn or its territory's spawns,
// and tells the GM (Mobius deleteNpc).
func (s *Server) gmDeleteNpc(p *Player, n *Npc) {
	switch sp := n.spawn; {
	case sp == nil:
		s.DespawnNpc(n)
	case sp.territory != nil:
		s.deleteTerritorySpawns(sp.territory.Name)
		p.SendMessage(fmt.Sprintf("Deleted %s group from %d.", n.Name(), n.ID()))
		return
	default:
		s.deleteSpawn(sp)
	}
	p.SendMessage(fmt.Sprintf("Deleted %s from %d.", n.Name(), n.ID()))
}

// gmRespawn is //respawn, a test helper for the spawns: with an NPC of a
// spawn as the target, the spawn puts its NPC into the world again now (at
// a new point for a territory spawn); without one, every spawn waiting for
// its respawn timer spawns now. Deleted spawns stay deleted.
func (s *Server) gmRespawn(p *Player, _ []string) {
	n, ok := p.Target.(*Npc)
	if !ok {
		p.SendMessage(fmt.Sprintf("Respawned %d NPCs.", s.respawnPending()))
		return
	}
	sp := n.spawn
	if sp == nil || sp.deleted() {
		p.SendMessage(fmt.Sprintf("%s has no spawn.", n.Name()))
		return
	}
	if s.respawnSlotNow(sp) == nil {
		p.SendMessage(fmt.Sprintf("%s could not be respawned.", n.Name()))
		return
	}
	p.SendMessage(fmt.Sprintf("Respawned %s.", n.Name()))
}

// gmNpcInfo is //npcinfo: the info page of the targeted NPC, the page a GM
// gets with a Shift-click.
func (s *Server) gmNpcInfo(p *Player, _ []string) {
	n, ok := p.Target.(*Npc)
	if !ok {
		p.SendMessage("Incorrect target.")
		return
	}
	s.sendNpcInfoPage(p, n)
}

// sendNpcInfoPage sends p the info page of n (a short form of Mobius
// admin/npcinfo.htm, which NpcShiftClick fills), as a page bound to no NPC.
// Its buttons are admin bypasses: ShowHTML puts them into p's link cache.
func (s *Server) sendNpcInfoPage(p *Player, n *Npc) {
	s.ShowHTML(p, 0, s.npcInfoPage(p, n))
}

// npcInfoPage builds the info page of n as seen by p.
func (s *Server) npcInfoPage(p *Player, n *Npc) string {
	t := n.Template
	pos := n.Position()
	var b strings.Builder
	b.WriteString(`<html><title>NPC Info</title><body><center>`)
	b.WriteString(`<table width=270 border=0><tr>`)
	b.WriteString(npcInfoButton("Delete", "bypass -h admin_delete"))
	b.WriteString(npcInfoButton("Respawn", "bypass -h admin_respawn"))
	b.WriteString(`</tr></table><img src="L2UI.SquareWhite" width=260 height=1><br>`)
	fmt.Fprintf(&b, `<table width=270 border=0><tr><td><font color="LEVEL">Npc Info</font> : %s (<font color="LEVEL">%d</font>) [<font color=00FF00>%d</font> lvl]</td></tr></table>`,
		t.Name, t.ID, t.Level)
	b.WriteString(`<table width=270 border=0>`)
	rows := [][2]string{
		{"ObjectId", strconv.Itoa(int(n.ID()))},
		{"Type:", t.Type},
		{"Race", t.Race.String()},
	}
	rows = append(rows, npcInfoSpawnRows(n.spawn)...)
	rows = append(rows,
		[2]string{"Location:", fmt.Sprintf("%d %d %d", pos.X, pos.Y, pos.Z)},
		[2]string{"Heading:", strconv.Itoa(int(pos.Heading))},
		[2]string{"Distance:", fmt.Sprintf("2D: %d, 3D: %d", int(p.Position().Distance2D(pos)), int(p.Position().Distance3D(pos)))},
		[2]string{"HP:", fmt.Sprintf("%d/%d", int(n.CurHP), n.MaxHP())},
		[2]string{"MP:", fmt.Sprintf("%d/%d", int(n.CurMP), n.MaxMP())},
	)
	for i, r := range rows {
		bg := ""
		if i%2 == 0 {
			bg = " bgcolor=000000"
		}
		fmt.Fprintf(&b, `<tr><td><table width=270 border=0%s><tr><td width=100><font color="LEVEL">%s</font></td><td align=right width=170>%s</td></tr></table></td></tr>`,
			bg, r[0], r[1])
	}
	b.WriteString(`</table><br><img src="L2UI.SquareWhite" width=260 height=1><br></center></body></html>`)
	return b.String()
}

// npcInfoSpawnRows are the spawn rows of the info page (Mobius
// NpcShiftClick): territory, spawn type, file, point and respawn delay.
func npcInfoSpawnRows(sp *npcSpawn) [][2]string {
	const none = "<font color=FF0000>--</font>"
	if sp == nil {
		return [][2]string{
			{"Territory:", none}, {"Spawn type:", none}, {"Spawn file:", none},
			{"Spawn loc:", "<font color=FF0000>null</font>"}, {"Respawn:", none},
		}
	}
	territory, kind, at := "None", "Fixed", sp.point
	if sp.territory != nil {
		territory, kind, at = sp.territory.Name, "Random", sp.lastPoint
	}
	file := "GM spawn"
	if sp.block != nil {
		file = sp.block.File
	}
	switch sp.period {
	case spawnDayOnly:
		kind += ", by day"
	case spawnNightOnly:
		kind += ", by night"
	}
	resp := "None"
	switch {
	case sp.respawnMin == 0:
	case sp.respawnMax != sp.respawnMin:
		resp = fmt.Sprintf("%d-%d sec", sp.respawnMin/time.Second, sp.respawnMax/time.Second)
	default:
		resp = fmt.Sprintf("%d sec", sp.respawnMin/time.Second)
	}
	return [][2]string{
		{"Territory:", territory},
		{"Spawn type:", kind},
		{"Spawn file:", file},
		{"Spawn loc:", fmt.Sprintf("%d %d %d", at.X, at.Y, at.Z)},
		{"Respawn:", resp},
	}
}

// npcInfoButton is a button cell of the info page.
func npcInfoButton(label, action string) string {
	return `<td><button value="` + label + `" action="` + action +
		`" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td>`
}
