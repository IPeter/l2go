package scripting

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/IPeter/l2go/internal/rnd"
)

// Mobius item IDs the quest helpers know.
const (
	adenaID = 57
)

// Mobius's built-in pages for quests (Quest.DEFAULT_*_MSG), used when the
// HTML folder has no noquest.htm or alreadycompleted.htm.
const (
	DefaultNoQuestMsg          = "<html><body>You are either not on a quest that involves this NPC, or you don't meet this NPC's minimum quest requirements.</body></html>"
	DefaultAlreadyCompletedMsg = "<html><body>This quest has already been completed.</body></html>"
)

// Mobius QuestSound names the API plays itself.
const (
	soundAccept  = "ItemSound.quest_accept"
	soundMiddle  = "ItemSound.quest_middle"
	soundFinish  = "ItemSound.quest_finish"
	soundItemGet = "ItemSound.quest_itemget"
)

// questEnv is what the quest API works with.
type questEnv struct {
	world   QuestWorld
	actions QuestActions // the world's, nil if it has none
	rand    rnd.Source
	consts  Constants
	rates   Rates
}

// needWorld returns the quest world for fn. While the scripts load, and
// without a world (scripts check), it raises an error.
func (v *vm) needWorld(L *lua.LState, fn string) QuestWorld {
	if v.loading {
		L.RaiseError("%s: the world cannot be used while the scripts load; do it in q:on_start", fn)
		return nil
	}
	if v.quest.world == nil {
		L.RaiseError("%s: this server has no quest world", fn)
		return nil
	}
	return v.quest.world
}

// needActions returns the world's QuestActions for fn, or raises an error.
func (v *vm) needActions(L *lua.LState, fn string) QuestActions {
	v.needWorld(L, fn)
	if v.quest.actions == nil {
		L.RaiseError("%s: this server cannot do that yet", fn)
		return nil
	}
	return v.quest.actions
}

// checkPlayerArg returns argument n, a Player.
func checkPlayerArg(L *lua.LState, n int, fn string) PlayerRef {
	p := optPlayerArg(L, n, fn)
	if p == nil {
		L.ArgError(n, fn+": a player expected, got nil")
	}
	return p
}

// optPlayerArg returns argument n, a Player or nil.
func optPlayerArg(L *lua.LState, n int, fn string) PlayerRef {
	switch x := L.Get(n).(type) {
	case *lua.LNilType:
		return nil
	case *lua.LUserData:
		if b, ok := x.Value.(*playerBox); ok {
			if b.ref == nil {
				L.ArgError(n, fn+": this player was passed to an earlier handler; keep player:object_id() instead of the player")
			}
			return b.ref
		}
	}
	L.ArgError(n, fn+": a player expected, got "+L.Get(n).Type().String())
	return nil
}

// optNpcArg returns argument n, an Npc or nil.
func optNpcArg(L *lua.LState, n int, fn string) NpcRef {
	switch x := L.Get(n).(type) {
	case *lua.LNilType:
		return nil
	case *lua.LUserData:
		if b, ok := x.Value.(*npcBox); ok {
			ref := b.resolve()
			if ref == nil {
				L.ArgError(n, fn+": "+goneNpcMsg)
			}
			return ref
		}
	}
	L.ArgError(n, fn+": an NPC expected, got "+L.Get(n).Type().String())
	return nil
}

// objectArg returns argument n, a Player or an Npc (as PlayerRef or
// NpcRef), or nil.
func objectArg(L *lua.LState, n int) any {
	ud, ok := L.Get(n).(*lua.LUserData)
	if !ok {
		return nil
	}
	switch b := ud.Value.(type) {
	case *playerBox:
		if b.ref != nil {
			return b.ref
		}
	case *npcBox:
		if ref := b.resolve(); ref != nil {
			return ref
		}
	}
	return nil
}

// checkInt returns argument n as an integer (Java int or long).
func checkInt(L *lua.LState, n int, fn string) int64 {
	f := float64(L.CheckNumber(n))
	if f != math.Trunc(f) || math.Abs(f) > 1<<53 {
		L.ArgError(n, fn+": an integer expected, got "+L.Get(n).String())
		return 0
	}
	return int64(f)
}

// optInt returns argument n as an integer, or def when it is nil.
func optInt(L *lua.LState, n int, fn string, def int64) int64 {
	if L.Get(n) == lua.LNil {
		return def
	}
	return checkInt(L, n, fn)
}

// questHelpers are the Quest helper methods of Script values (Mobius Quest
// and AbstractScript methods, in snake case).
func (v *vm) questHelpers() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get_quest_state":               v.qGetQuestState,
		"new_quest_state":               v.qNewQuestState,
		"get_no_quest_msg":              v.qPage("get_no_quest_msg", "noquest.htm", DefaultNoQuestMsg),
		"get_already_completed_msg":     v.qPage("get_already_completed_msg", "alreadycompleted.htm", DefaultAlreadyCompletedMsg),
		"get_htm":                       v.qGetHtm,
		"show_html_file":                v.qShowHTMLFile,
		"give_items":                    v.qGiveItems,
		"reward_items":                  v.qRewardItems,
		"give_adena":                    v.qGiveAdena,
		"take_items":                    v.qTakeItems,
		"take_items_ids":                v.qTakeItemsIDs,
		"get_quest_items_count":         v.qGetQuestItemsCount,
		"has_quest_items":               v.qHasQuestItems,
		"has_at_least_one_quest_item":   v.qHasAtLeastOneQuestItem,
		"give_item_randomly":            v.qGiveItemRandomly,
		"add_exp_and_sp":                v.qAddExpAndSp,
		"play_sound":                    v.qPlaySound,
		"get_item_equipped":             v.qGetItemEquipped,
		"get_enchant_level":             v.qGetEnchantLevel,
		"start_quest_timer":             v.qStartQuestTimer,
		"get_quest_timer":               v.qGetQuestTimer,
		"cancel_quest_timer":            v.qCancelQuestTimer,
		"cancel_quest_timers":           v.qCancelQuestTimers,
		"get_random_party_member":       v.qGetRandomPartyMember,
		"get_random_party_member_state": v.qGetRandomPartyMemberState,
		"execute_for_each_player":       v.qExecuteForEachPlayer,
		"add_spawn":                     v.qAddSpawn,
		"add_radar":                     v.qRadar("add_radar", 0),
		"remove_radar":                  v.qRadar("remove_radar", 1),
		"clear_radar":                   v.qClearRadar,
		"show_on_screen_msg":            v.qShowOnScreenMsg,
	}
}

// questState returns the player's state of s, creating it when create is
// set (Mobius getQuestState).
func (v *vm) questState(L *lua.LState, s *npcScript, p PlayerRef, create bool, fn string) QuestState {
	w := v.needWorld(L, fn)
	st := w.QuestState(p, s.name)
	if st == nil && create {
		st = w.NewQuestState(p, s.name)
	}
	return st
}

// qGetQuestState is q:get_quest_state(player, create).
func (v *vm) qGetQuestState(L *lua.LState) int {
	s := checkScript(L, "get_quest_state")
	p := checkPlayerArg(L, 2, "q:get_quest_state")
	st := v.questState(L, s, p, L.ToBool(3), "q:get_quest_state")
	L.Push(v.newQuestState(L, p, st))
	return 1
}

// qNewQuestState is q:new_quest_state(player).
func (v *vm) qNewQuestState(L *lua.LState) int {
	s := checkScript(L, "new_quest_state")
	p := checkPlayerArg(L, 2, "q:new_quest_state")
	st := v.needWorld(L, "q:new_quest_state").NewQuestState(p, s.name)
	L.Push(v.newQuestState(L, p, st))
	return 1
}

// qPage returns q:get_no_quest_msg(player) and the like: the page from
// the HTML folder, or Mobius's built-in text.
func (v *vm) qPage(method, page, def string) lua.LGFunction {
	return func(L *lua.LState) int {
		checkScript(L, method)
		p := optPlayerArg(L, 2, "q:"+method)
		w := v.needWorld(L, "q:"+method)
		if html, ok := w.HTML(p, page); ok && html != "" {
			L.Push(lua.LString(html))
		} else {
			L.Push(lua.LString(def))
		}
		return 1
	}
}

// htm returns a page of s: next to the script, then in the HTML folder
// under scripts/<the script's folder>/ (Mobius getHtm).
func (v *vm) htm(L *lua.LState, s *npcScript, p PlayerRef, file, fn string) (string, bool) {
	if !fs.ValidPath(file) || strings.Contains(file, `\`) {
		L.ArgError(3, fmt.Sprintf("%s: invalid page name %q", fn, file))
		return "", false
	}
	if data, err := os.ReadFile(filepath.Join(s.dir, filepath.FromSlash(file))); err == nil {
		return string(data), true
	}
	w := v.needWorld(L, fn)
	return w.HTML(p, path.Join("scripts", s.rel, file))
}

// qGetHtm is q:get_htm(player, file): the page's HTML, or nil.
func (v *vm) qGetHtm(L *lua.LState) int {
	s := checkScript(L, "get_htm")
	p := optPlayerArg(L, 2, "q:get_htm")
	if html, ok := v.htm(L, s, p, L.CheckString(3), "q:get_htm"); ok {
		L.Push(lua.LString(html))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// qShowHTMLFile is q:show_html_file(player, file[, npc]): shows the page
// and returns its HTML (nil if it does not exist).
func (v *vm) qShowHTMLFile(L *lua.LState) int {
	s := checkScript(L, "show_html_file")
	p := checkPlayerArg(L, 2, "q:show_html_file")
	file := L.CheckString(3)
	npc := optNpcArg(L, 4, "q:show_html_file")
	html, ok := v.htm(L, s, p, file, "q:show_html_file")
	if !ok {
		v.logger.Debug("q:show_html_file: no such page", "script", s.name, "page", file)
		L.Push(lua.LNil)
		return 1
	}
	v.quest.world.ShowReply(p, npc, Reply{Kind: ReplyPage, Content: file, Script: s.name, Dir: s.dir})
	L.Push(lua.LString(html))
	return 1
}

// validItem reports whether an item ID and count are worth passing on; the
// others are ignored like Mobius does, logged at debug level.
func (v *vm) validItem(L *lua.LState, fn string, id int, count int64) bool {
	if id <= 0 || count <= 0 {
		v.logger.Debug(fn+": ignored", "item", id, "count", count, "script", where(L))
		return false
	}
	return true
}

// itemArgs reads the item of q:give_items and the like from argument 3: an
// item ID and a count (arguments 3 and 4), or an item {id =, count =}
// (Mobius's ItemHolder overloads). next is the argument after them.
func itemArgs(L *lua.LState, fn string) (id int, count int64, next int) {
	if t, ok := L.Get(3).(*lua.LTable); ok {
		idv, okID := t.RawGetString("id").(lua.LNumber)
		cv, okCount := t.RawGetString("count").(lua.LNumber)
		if !okID || !okCount || float64(idv) != math.Trunc(float64(idv)) || float64(cv) != math.Trunc(float64(cv)) ||
			math.Abs(float64(idv)) > math.MaxInt32 || math.Abs(float64(cv)) > 1<<53 {
			L.ArgError(3, fn+": an item {id = integer, count = integer} expected")
		}
		return int(idv), int64(cv), 4
	}
	return int(checkInt(L, 3, fn)), checkInt(L, 4, fn), 5
}

// qGiveItems is q:give_items(player, id, count[, enchant]) or
// q:give_items(player, {id =, count =}[, enchant]).
func (v *vm) qGiveItems(L *lua.LState) int {
	checkScript(L, "give_items")
	p := checkPlayerArg(L, 2, "q:give_items")
	id, count, next := itemArgs(L, "q:give_items")
	enchant := int(optInt(L, next, "q:give_items", 0))
	w := v.needWorld(L, "q:give_items")
	if v.validItem(L, "q:give_items", id, count) {
		w.GiveItems(p, id, count, enchant)
	}
	return 0
}

// qRewardItems is q:reward_items(player, id, count) or
// q:reward_items(player, {id =, count =}).
func (v *vm) qRewardItems(L *lua.LState) int {
	checkScript(L, "reward_items")
	p := checkPlayerArg(L, 2, "q:reward_items")
	id, count, _ := itemArgs(L, "q:reward_items")
	w := v.needWorld(L, "q:reward_items")
	if v.validItem(L, "q:reward_items", id, count) {
		w.RewardItems(p, id, count)
	}
	return 0
}

// qGiveAdena is q:give_adena(player, count, apply_rates).
func (v *vm) qGiveAdena(L *lua.LState) int {
	checkScript(L, "give_adena")
	p := checkPlayerArg(L, 2, "q:give_adena")
	count := checkInt(L, 3, "q:give_adena")
	rates := L.ToBool(4)
	w := v.needWorld(L, "q:give_adena")
	if !v.validItem(L, "q:give_adena", adenaID, count) {
		return 0
	}
	if rates {
		w.RewardItems(p, adenaID, count)
	} else {
		w.GiveItems(p, adenaID, count, 0)
	}
	return 0
}

// qTakeItems is q:take_items(player, id, count): count items, -1 all of
// them; or q:take_items(player, {id =, count =}). Returns true, as Mobius
// does.
func (v *vm) qTakeItems(L *lua.LState) int {
	checkScript(L, "take_items")
	p := checkPlayerArg(L, 2, "q:take_items")
	id, count, _ := itemArgs(L, "q:take_items")
	w := v.needWorld(L, "q:take_items")
	if id > 0 && count != 0 {
		w.TakeItems(p, id, count)
	}
	L.Push(lua.LTrue)
	return 1
}

// qTakeItemsIDs is q:take_items_ids(player, count, id...): Mobius's
// takeItems(player, amount, itemIds...).
func (v *vm) qTakeItemsIDs(L *lua.LState) int {
	checkScript(L, "take_items_ids")
	p := checkPlayerArg(L, 2, "q:take_items_ids")
	count := checkInt(L, 3, "q:take_items_ids")
	ids := checkIDList(L, 4, "item", true)
	w := v.needWorld(L, "q:take_items_ids")
	for _, id := range ids {
		if id > 0 && count != 0 {
			w.TakeItems(p, id, count)
		}
	}
	L.Push(lua.LTrue)
	return 1
}

// itemCounts returns the counts of ids (arguments from 3 on).
func (v *vm) itemCounts(L *lua.LState, fn string) []int64 {
	checkScript(L, strings.TrimPrefix(fn, "q:"))
	p := checkPlayerArg(L, 2, fn)
	ids := checkIDList(L, 3, "item", true)
	w := v.needWorld(L, fn)
	counts := make([]int64, len(ids))
	for i, id := range ids {
		if id > 0 {
			counts[i] = w.ItemCount(p, id)
		}
	}
	return counts
}

// qGetQuestItemsCount is q:get_quest_items_count(player, id...): the sum,
// capped at the int maximum like Mobius.
func (v *vm) qGetQuestItemsCount(L *lua.LState) int {
	var sum int64
	for _, c := range v.itemCounts(L, "q:get_quest_items_count") {
		sum += c
		if sum > math.MaxInt32 {
			sum = math.MaxInt32
		}
	}
	L.Push(lua.LNumber(sum))
	return 1
}

// qHasQuestItems is q:has_quest_items(player, id...): all of them; false
// for no IDs.
func (v *vm) qHasQuestItems(L *lua.LState) int {
	counts := v.itemCounts(L, "q:has_quest_items")
	ok := len(counts) > 0
	for _, c := range counts {
		ok = ok && c > 0
	}
	L.Push(lua.LBool(ok))
	return 1
}

// qHasAtLeastOneQuestItem is q:has_at_least_one_quest_item(player, id...).
func (v *vm) qHasAtLeastOneQuestItem(L *lua.LState) int {
	ok := false
	for _, c := range v.itemCounts(L, "q:has_at_least_one_quest_item") {
		ok = ok || c > 0
	}
	L.Push(lua.LBool(ok))
	return 1
}

// qGiveItemRandomly is q:give_item_randomly(player, npc, id, min, max,
// limit, chance, play_sound), Mobius giveItemRandomly with its draws in
// Mobius's order. Champions do not exist yet, so their bonus is left out.
func (v *vm) qGiveItemRandomly(L *lua.LState) int {
	const fn = "q:give_item_randomly"
	checkScript(L, "give_item_randomly")
	p := checkPlayerArg(L, 2, fn)
	npc := optNpcArg(L, 3, fn)
	id := int(checkInt(L, 4, fn))
	minAmount := checkInt(L, 5, fn)
	maxAmount := checkInt(L, 6, fn)
	limit := checkInt(L, 7, fn)
	chance := float64(L.CheckNumber(8))
	sound := L.ToBool(9)
	w := v.needWorld(L, fn)

	current := w.ItemCount(p, id)
	if limit > 0 && current >= limit {
		L.Push(lua.LTrue)
		return 1
	}
	mult := v.quest.rates.QuestItemDropAmount
	minBonus := int64(float64(minAmount) * mult)
	maxBonus := int64(float64(maxAmount) * mult)
	amount := minBonus
	if minBonus != maxBonus {
		amount = int64(v.quest.rand.Between(int(minBonus), int(maxBonus)))
	}
	random := v.quest.rand.Float64()
	if chance >= random && amount > 0 && w.CanAddItem(p, id) {
		if limit > 0 && current+amount > limit {
			amount = limit - current
		}
		if w.AddItem(p, id, amount, npc) {
			if current+amount == limit {
				if sound {
					w.PlaySound(p, soundMiddle)
				}
				L.Push(lua.LTrue)
				return 1
			}
			if sound {
				w.PlaySound(p, soundItemGet)
			}
			if limit <= 0 {
				L.Push(lua.LTrue)
				return 1
			}
		}
	}
	L.Push(lua.LFalse)
	return 1
}

// qAddExpAndSp is q:add_exp_and_sp(player, exp, sp).
func (v *vm) qAddExpAndSp(L *lua.LState) int {
	checkScript(L, "add_exp_and_sp")
	p := checkPlayerArg(L, 2, "q:add_exp_and_sp")
	exp := checkInt(L, 3, "q:add_exp_and_sp")
	sp := checkInt(L, 4, "q:add_exp_and_sp")
	v.needWorld(L, "q:add_exp_and_sp").AddExpAndSp(p, exp, sp)
	return 0
}

// qPlaySound is q:play_sound(player, sound): a QuestSound value or any
// sound name.
func (v *vm) qPlaySound(L *lua.LState) int {
	checkScript(L, "play_sound")
	p := checkPlayerArg(L, 2, "q:play_sound")
	sound := L.CheckString(3)
	v.needWorld(L, "q:play_sound").PlaySound(p, sound)
	return 0
}

// qGetItemEquipped is q:get_item_equipped(player, slot).
func (v *vm) qGetItemEquipped(L *lua.LState) int {
	checkScript(L, "get_item_equipped")
	p := checkPlayerArg(L, 2, "q:get_item_equipped")
	slot := int(checkInt(L, 3, "q:get_item_equipped"))
	L.Push(lua.LNumber(v.needWorld(L, "q:get_item_equipped").EquippedItemID(p, slot)))
	return 1
}

// qGetEnchantLevel is q:get_enchant_level(player, id).
func (v *vm) qGetEnchantLevel(L *lua.LState) int {
	checkScript(L, "get_enchant_level")
	p := checkPlayerArg(L, 2, "q:get_enchant_level")
	id := int(checkInt(L, 3, "q:get_enchant_level"))
	L.Push(lua.LNumber(v.needWorld(L, "q:get_enchant_level").EnchantLevel(p, id)))
	return 1
}

// partyRange is the range of the party helpers (Mobius ALT_PARTY_RANGE).
func (v *vm) partyRange() int {
	if f, ok := v.quest.consts.Tables["Config"]["ALT_PARTY_RANGE"]; ok {
		if n, ok := toNumber(f); ok {
			return int(n)
		}
	}
	return defaultPartyRange
}

// inRangeOf is Mobius checkDistanceToTarget: no target, or the player
// within the party range of it.
func (v *vm) inRangeOf(p PlayerRef, target NpcRef) bool {
	return target == nil || inRange(v.partyRange(), p, target, true)
}

// qGetRandomPartyMember is q:get_random_party_member(player[, cond |
// var, value | npc]). There are no parties yet, so these are Mobius's
// paths for a player without one.
func (v *vm) qGetRandomPartyMember(L *lua.LState) int {
	const fn = "q:get_random_party_member"
	s := checkScript(L, "get_random_party_member")
	p := optPlayerArg(L, 2, fn)
	if p == nil {
		L.Push(lua.LNil)
		return 1
	}
	var match bool
	switch x := L.Get(3).(type) {
	case *lua.LNilType: // getRandomPartyMember(player)
		match = true
	case lua.LNumber: // (player, cond)
		match = v.stateVarIs(L, s, p, "cond", lua.LVAsString(x), fn)
	case lua.LString: // (player, var, value)
		match = v.stateVarIs(L, s, p, string(x), L.CheckString(4), fn)
	case *lua.LUserData: // (player, npc)
		npc := optNpcArg(L, 3, fn)
		match = v.inRangeOf(p, npc) && v.checkPartyMember(L, s, p, npc)
	default:
		L.ArgError(3, fn+": a cond, a variable name or an NPC expected")
	}
	if match {
		L.Push(v.newPlayer(L, p))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// stateVarIs reports whether the player's state of s has the variable set
// to value, ignoring case.
func (v *vm) stateVarIs(L *lua.LState, s *npcScript, p PlayerRef, name, value, fn string) bool {
	st := v.questState(L, s, p, false, fn)
	if st == nil {
		return false
	}
	got, ok := st.Get(name)
	return ok && strings.EqualFold(got, value)
}

// checkPartyMember runs on_check_party_member(player, npc); true without
// a handler.
func (v *vm) checkPartyMember(L *lua.LState, s *npcScript, p PlayerRef, npc NpcRef) bool {
	reg, ok := s.handlerFor(hookCheckPartyMember, 0)
	if !ok {
		return true
	}
	L.Push(reg.fn)
	L.Push(v.newPlayer(L, p))
	L.Push(v.newNpc(L, npc))
	L.Call(2, 1)
	ret := L.Get(-1)
	L.Pop(1)
	return lua.LVAsBool(ret)
}

// checkPartyMemberState runs on_check_party_member_state(st, npc); true
// without a handler.
func (v *vm) checkPartyMemberState(L *lua.LState, s *npcScript, st lua.LValue, npc NpcRef) bool {
	reg, ok := s.handlerFor(hookCheckPartyMemberState, 0)
	if !ok {
		return true
	}
	L.Push(reg.fn)
	L.Push(st)
	L.Push(v.newNpc(L, npc))
	L.Call(2, 1)
	ret := L.Get(-1)
	L.Pop(1)
	return lua.LVAsBool(ret)
}

// qGetRandomPartyMemberState is q:get_random_party_member_state(player,
// state) → Player, or (player, cond, chance, npc) → QuestState; cond -1
// only checks that the quest is started. Solo paths, as above.
func (v *vm) qGetRandomPartyMemberState(L *lua.LState) int {
	const fn = "q:get_random_party_member_state"
	s := checkScript(L, "get_random_party_member_state")
	p := optPlayerArg(L, 2, fn)
	if L.GetTop() <= 3 {
		state := int(checkInt(L, 3, fn))
		if p == nil {
			L.Push(lua.LNil)
			return 1
		}
		if st := v.questState(L, s, p, false, fn); st != nil && st.State() == state {
			L.Push(v.newPlayer(L, p))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}
	cond := checkInt(L, 3, fn)
	chance := checkInt(L, 4, fn)
	npc := optNpcArg(L, 5, fn)
	if p == nil || chance < 1 {
		L.Push(lua.LNil)
		return 1
	}
	st := v.questState(L, s, p, false, fn)
	if st == nil {
		L.Push(lua.LNil)
		return 1
	}
	matches := st.State() == StateStarted
	if cond != -1 {
		matches = int64(st.Cond()) == cond
	}
	stv := v.newQuestState(L, p, st)
	if !matches || !v.checkPartyMemberState(L, s, stv, npc) || !v.inRangeOf(p, npc) {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(stv)
	return 1
}

// qExecuteForEachPlayer is q:execute_for_each_player(player, npc,
// is_summon, include_party, include_command_channel): it runs
// on_action_for_each_player for the player (parties come later).
func (v *vm) qExecuteForEachPlayer(L *lua.LState) int {
	const fn = "q:execute_for_each_player"
	s := checkScript(L, "execute_for_each_player")
	p := optPlayerArg(L, 2, fn)
	npc := optNpcArg(L, 3, fn)
	summon := L.ToBool(4)
	if p == nil {
		return 0
	}
	reg, ok := s.handlerFor(hookActionForEachPlayer, 0)
	if !ok {
		return 0
	}
	L.Push(reg.fn)
	L.Push(v.newPlayer(L, p))
	L.Push(v.newNpc(L, npc))
	L.Push(lua.LBool(summon))
	L.Call(3, 0)
	return 0
}

// locationArg reads a position argument: a table {x=, y=, z=, heading=}
// or a Player or Npc (its position).
func locationArg(L *lua.LState, n int, fn string) Location {
	switch x := L.Get(n).(type) {
	case *lua.LTable:
		get := func(k string) int {
			num, _ := x.RawGetString(k).(lua.LNumber)
			return int(num)
		}
		return Location{X: get("x"), Y: get("y"), Z: get("z"), Heading: get("heading")}
	case *lua.LUserData:
		if obj := objectArg(L, n); obj != nil {
			if loc, ok := obj.(Located); ok {
				return loc.Location()
			}
			L.ArgError(n, fn+": this object has no position")
		}
	}
	L.ArgError(n, fn+": a position {x=, y=, z=, heading=}, a player or an NPC expected")
	return Location{}
}

// qAddSpawn is q:add_spawn(npc_id, pos[, random_offset, despawn_ms,
// summon]) → Npc or nil.
func (v *vm) qAddSpawn(L *lua.LState) int {
	const fn = "q:add_spawn"
	checkScript(L, "add_spawn")
	id := int(checkInt(L, 2, fn))
	loc := locationArg(L, 3, fn)
	offset := L.ToBool(4)
	despawn := optInt(L, 5, fn, 0)
	summon := L.ToBool(6)
	a := v.needActions(L, fn)
	npc := a.AddSpawn(id, loc, offset, time.Duration(despawn)*time.Millisecond, summon)
	L.Push(v.newNpc(L, npc))
	return 1
}

// qRadar returns q:add_radar(player, x, y, z) or q:remove_radar.
func (v *vm) qRadar(method string, op int) lua.LGFunction {
	return func(L *lua.LState) int {
		fn := "q:" + method
		checkScript(L, method)
		p := checkPlayerArg(L, 2, fn)
		x, y, z := int(checkInt(L, 3, fn)), int(checkInt(L, 4, fn)), int(checkInt(L, 5, fn))
		v.needActions(L, fn).Radar(p, op, x, y, z)
		return 0
	}
}

// qClearRadar is q:clear_radar(player).
func (v *vm) qClearRadar(L *lua.LState) int {
	checkScript(L, "clear_radar")
	p := checkPlayerArg(L, 2, "q:clear_radar")
	v.needActions(L, "q:clear_radar").Radar(p, 2, 0, 0, 0)
	return 0
}

// qShowOnScreenMsg is q:show_on_screen_msg(player, text, ms).
func (v *vm) qShowOnScreenMsg(L *lua.LState) int {
	checkScript(L, "show_on_screen_msg")
	p := checkPlayerArg(L, 2, "q:show_on_screen_msg")
	text := L.CheckString(3)
	ms := int(checkInt(L, 4, "q:show_on_screen_msg"))
	v.needActions(L, "q:show_on_screen_msg").ShowOnScreenMsg(p, text, ms)
	return 0
}
