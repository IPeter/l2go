package game

import (
	"context"
	"errors"
	"fmt"
	"html"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/quest"
	"github.com/IPeter/l2go/internal/storage"
)

// The quest GM commands, with Mobius's names (AdminShowQuests, AdminQuest,
// AdminEditChar): //charquestmenu, //setcharquest, //show_quests,
// //quest_info, //quest_reload and //setclass. //set_level, //add_level and
// //create_item are in death_gm.go and items_gm.go. The pages are plain
// lists, not Mobius's admin panels.

// registerQuestGMCommands registers the commands for access level 1 and up.
func (s *Server) registerQuestGMCommands() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("charquestmenu", admin, s.gmCharQuestMenu),
		s.RegisterGMCommand("show_quests", admin, s.gmCharQuestMenu),
		s.RegisterGMCommand("setcharquest", admin, s.gmSetCharQuest),
		s.RegisterGMCommand("quest_info", admin, s.gmQuestInfo),
		s.RegisterGMCommand("quest_reload", admin, s.gmQuestReload),
		s.RegisterGMCommand("setclass", admin, s.gmSetClass),
	)
}

// gmTargetPlayer returns the player named by args[0], or the GM's target
// when it is a player, or the GM.
func (s *Server) gmTargetPlayer(gm *Player, args []string) (*Player, bool) {
	if len(args) > 0 {
		t, ok := s.PlayerByName(args[0])
		if !ok {
			gm.SendMessage("The player " + args[0] + " is not online.")
		}
		return t, ok
	}
	if t, ok := gm.Target.(*Player); ok {
		return t, true
	}
	return gm, true
}

// gmCharQuestMenu is //charquestmenu [player] and //show_quests [player]:
// the player's quest states with their variables.
func (s *Server) gmCharQuestMenu(gm *Player, args []string) {
	t, ok := s.gmTargetPlayer(gm, args)
	if !ok {
		return
	}
	var b strings.Builder
	b.WriteString("<html><body><title>Quests of " + html.EscapeString(t.Name()) + "</title>")
	states := s.questsOf(t).book.States()
	if len(states) == 0 {
		b.WriteString("No quests.")
	}
	for _, st := range states {
		id := ""
		if q, ok := st.Quest(); ok {
			id = " (" + strconv.Itoa(int(q.ID)) + ")"
		}
		b.WriteString(html.EscapeString(st.Name()) + id + ": " + st.State().Name())
		vars := st.Vars()
		for _, k := range slices.Sorted(maps.Keys(vars)) {
			b.WriteString("<br1>" + html.EscapeString(k) + " = " + html.EscapeString(vars[k]))
		}
		b.WriteString("<br>")
	}
	b.WriteString("</body></html>")
	s.ShowHTML(gm, 0, b.String())
}

// gmSetCharQuest is //setcharquest <player> <quest> <var> <value>: "state"
// sets the state (Start, Started, Completed), the value "delete" ends the
// quest as exit_quest(true), anything else sets the variable.
func (s *Server) gmSetCharQuest(gm *Player, args []string) {
	if len(args) < 4 {
		gm.SendMessage("Usage: //setcharquest <player> <quest> <var> <value>")
		return
	}
	t, ok := s.PlayerByName(args[0])
	if !ok {
		gm.SendMessage("The player " + args[0] + " is not online.")
		return
	}
	name := s.questName(args[1])
	if name == "" {
		gm.SendMessage("No quest " + args[1] + ".")
		return
	}
	book := s.questsOf(t).book
	st := book.State(name, true)
	variable, value := args[2], strings.Join(args[3:], " ")
	switch {
	case value == "delete":
		if !st.IsStarted() {
			st.SetState(quest.Started)
		}
		st.ExitQuest(true, false)
	case variable == "state" || variable == quest.VarState:
		st.SetState(quest.StateFromName(value))
	default:
		st.Set(variable, value)
	}
	t.Send(s.questListPacket(t))
	gm.SendMessage(fmt.Sprintf("%s: %s %s = %s", t.Name(), name, variable, value))
}

// questName returns the script name of a quest given by name or ID, "" if
// there is none.
func (s *Server) questName(arg string) string {
	if s.scripts == nil {
		return ""
	}
	if id, err := strconv.Atoi(arg); err == nil {
		if q, ok := s.scripts.QuestByID(id); ok {
			return q.Name
		}
		return ""
	}
	if q, ok := s.scripts.Quest(arg); ok {
		return q.Name
	}
	return ""
}

// gmQuestInfo is //quest_info <name or id>: what the quest registered.
func (s *Server) gmQuestInfo(gm *Player, args []string) {
	if len(args) == 0 {
		gm.SendMessage("Usage: //quest_info <quest name or id>")
		return
	}
	name := s.questName(args[0])
	if name == "" {
		gm.SendMessage("No quest " + args[0] + ".")
		return
	}
	q, _ := s.scripts.Quest(name)
	gm.SendMessage(fmt.Sprintf("%s (%d) %q: start NPCs %v, talk NPCs %v, quest items %v, requires %v, custom %v, folder %s",
		q.Name, q.ID, q.Description, q.StartNpcs, q.TalkNpcs, q.QuestItems, q.Requires, q.Custom, q.RelDir))
}

// gmQuestReload is //quest_reload: the scripts load again off the loop and
// are swapped in; the quest states stay, as they live in Go.
func (s *Server) gmQuestReload(gm *Player, _ []string) {
	if s.scripts == nil {
		gm.SendMessage("This server has no scripts.")
		return
	}
	engine := s.scripts
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		p, err := engine.Prepare(ctx)
		s.loop.Post(func() {
			if err != nil {
				gm.SendMessage("Script reload failed: " + err.Error())
				return
			}
			files := p.Files()
			engine.Swap(p)
			gm.SendMessage(fmt.Sprintf("Scripts reloaded: %d files.", files))
		})
	}()
}

// gmSetClass is //setclass <id>: the target player, or the GM, changes
// class (Mobius AdminEditChar setclass).
func (s *Server) gmSetClass(gm *Player, args []string) {
	if len(args) == 0 {
		gm.SendMessage("Usage: //setclass <class id>")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		gm.SendMessage("Usage: //setclass <class id>")
		return
	}
	t, _ := s.gmTargetPlayer(gm, nil)
	if _, ok := s.data.ClassTemplate(id); !ok {
		gm.SendMessage("No class " + args[0] + ".")
		return
	}
	s.ChangeClass(t, id)
}
