// Package scripttest runs Lua scripts in Go tests: load scripts, fire
// events, talk to scripted NPCs, and check what the handlers decided,
// said and logged.
//
//	h := scripttest.Load(t, "../../scripts", "events/character_create.lua")
//	res := h.CharacterCreate(scripting.CharacterCreate{Name: "Announcer"})
//	if !res.Rejected { ... }
//
//	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 5}
//	guard := &scripttest.Npc{ObjectID: 2, ID: 30733, Name: "Guard"}
//	reply, ok := h.FirstTalk(hero, guard)
//
// The harness calls the engine on the test's goroutine, the way the world
// loop does on the server. Do not call it from other goroutines.
package scripttest

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
)

// callTimeout bounds one call into the engine, on top of the engine's own
// handler timeout.
const callTimeout = 5 * time.Second

// Harness is a script engine for one test, with a fake world (World), a
// fake clock and a random source tests can script (SetRand).
type Harness struct {
	t      testing.TB
	dir    string
	engine *scripting.Engine
	logs   *recorder
	out    *output
	world  *World
	rand   *switchSource
}

// switchSource is the harness's random source: a seeded one until the test
// sets its own.
type switchSource struct{ rnd.Source }

// Load loads the scripts in dir. With no files, every script runs, as on
// the server; otherwise only the given scripts (slash paths relative to
// dir) run, and lib/ stays available to require. The test fails if a
// script does not load. The engine is closed when the test ends.
func Load(t testing.TB, dir string, files ...string) *Harness {
	t.Helper()
	opts := scripting.Options{Dir: dir}
	if len(files) > 0 {
		opts.Include = func(path string) bool { return slices.Contains(files, path) }
	}
	return start(t, opts)
}

// LoadWithTables is Load with more constant tables on top of Constants
// (e.g. the generated SystemMessageId of data.LoadConstants).
func LoadWithTables(t testing.TB, dir string, tables map[string]map[string]any, files ...string) *Harness {
	t.Helper()
	opts := scripting.Options{Dir: dir, Constants: scripting.Constants{Tables: tables}}
	if len(files) > 0 {
		opts.Include = func(path string) bool { return slices.Contains(files, path) }
	}
	return start(t, opts)
}

// LoadSource writes files (slash path relative to the scripts folder →
// Lua source) to a temporary scripts folder and loads all of them.
func LoadSource(t testing.TB, files map[string]string) *Harness {
	t.Helper()
	return start(t, scripting.Options{Dir: WriteScripts(t, files)})
}

// WriteScripts writes files (slash path → Lua source) to a new temporary
// folder and returns its path.
func WriteScripts(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		WriteScript(t, dir, name, src)
	}
	return dir
}

// WriteScript writes one script below dir, creating folders as needed.
func WriteScript(t testing.TB, dir, name, src string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}

func start(t testing.TB, opts scripting.Options) *Harness {
	t.Helper()
	h := &Harness{
		t: t, dir: opts.Dir, logs: newRecorder(t), out: &output{},
		world: NewWorld(), rand: &switchSource{rnd.New(1)},
	}
	opts.Logger = slog.New(h.logs)
	opts.Output = h.out
	opts.World = h.world
	opts.Rand = h.rand
	consts := Constants()
	maps.Copy(consts.Tables, opts.Constants.Tables)
	opts.Constants = consts
	e, err := scripting.New(t.Context(), opts)
	if err != nil {
		t.Fatalf("loading scripts: %v", err)
	}
	t.Cleanup(e.Close)
	h.engine = e
	h.world.onGone = e.NpcGone
	h.world.onSpawn = func(n scripting.NpcRef) { _ = e.Spawn(context.Background(), n) }
	h.syncCustom()
	return h
}

// syncCustom tells the fake world which quests are custom, as the game
// tells its quest model.
func (h *Harness) syncCustom() {
	clear(h.world.Custom)
	for _, q := range h.engine.Quests() {
		if q.Custom {
			h.world.Custom[q.Name] = true
		}
	}
}

// Constants returns the generated constant tables the server passes to the
// engine, built from internal/data: ClassId (with the class helpers) and
// AbnormalType.
func Constants() scripting.Constants {
	c := scripting.Constants{Tables: map[string]map[string]any{"AbnormalType": {}}}
	for i := range data.NumAbnormalTypes {
		name := data.AbnormalType(i).String()
		c.Tables["AbnormalType"][name] = name
	}
	for _, pc := range data.PlayerClasses() {
		parent := -1
		if p, ok := pc.Parent(); ok {
			parent = int(p)
		}
		c.Classes = append(c.Classes, scripting.ClassInfo{
			ID: int(pc), Name: pc.String(), Level: pc.Level(), Race: int(pc.Race()), Parent: parent, Mage: pc.IsMage(),
		})
	}
	return c
}

// World returns the fake world.
func (h *Harness) World() *World { return h.world }

// SetRand makes src the random source of rnd, math.random and the quest
// helpers, e.g. rnd.NewScripted(t, rnd.Get(10, 3)) to force a drop.
func (h *Harness) SetRand(src rnd.Source) { h.rand.Source = src }

// Start runs the scripts' on_start handlers, as the game does once its
// world is ready.
func (h *Harness) Start() {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := h.engine.Start(ctx); err != nil {
		h.t.Fatalf("on_start failed: %v", err)
	}
}

// Advance moves the fake clock by d and runs the quest timers that come
// due, as the world loop does.
func (h *Harness) Advance(d time.Duration) { h.world.Advance(d) }

// KillDelay is Mobius's onKillDelay: the kill event runs this long after
// the death.
const KillDelay = 2500 * time.Millisecond

// Kill runs the kill handlers for npc, killed by player, after the 2.5 s
// kill delay on the fake clock (timers due meanwhile run too). The test
// fails if a handler fails.
func (h *Harness) Kill(player *Player, npc *Npc) {
	h.t.Helper()
	h.world.Add(player, npc)
	var err error
	h.world.After(KillDelay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		err = h.engine.Kill(ctx, npc.Ref(), player.Ref(), false)
	})
	h.world.Advance(KillDelay)
	if err != nil {
		h.t.Fatalf("kill handler failed: %v", err)
	}
}

// Attack runs the attack handlers for npc, hit by player for damage.
func (h *Harness) Attack(player *Player, npc *Npc, damage int) {
	h.t.Helper()
	h.world.Add(player, npc)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := h.engine.Attack(ctx, npc.Ref(), player.Ref(), damage, false, nil); err != nil {
		h.t.Fatalf("attack handler failed: %v", err)
	}
}

// EnterWorld runs the on_enter_world handlers for player.
func (h *Harness) EnterWorld(player *Player) {
	h.t.Helper()
	h.world.Add(player)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := h.engine.EnterWorld(ctx, player.Ref()); err != nil {
		h.t.Fatalf("enter-world handler failed: %v", err)
	}
}

// Engine returns the engine, for calls the harness has no helper for.
func (h *Harness) Engine() *scripting.Engine { return h.engine }

// Dir returns the scripts folder.
func (h *Harness) Dir() string { return h.dir }

// Reload loads the scripts again and swaps them in, as the server does
// after a change. On a load error the old scripts stay and the error is
// returned.
func (h *Harness) Reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	p, err := h.engine.Prepare(ctx)
	if err != nil {
		return err
	}
	h.engine.Swap(p)
	h.syncCustom()
	return nil
}

// Fire runs the handlers of ev and returns the result. The test fails if
// a handler fails or the event cannot run.
func (h *Harness) Fire(ev scripting.Event) scripting.Result {
	h.t.Helper()
	res := h.FireAllowErrors(ev)
	if res.Err != nil {
		h.t.Fatalf("%s handler failed: %v", ev.Name, res.Err)
	}
	return res
}

// FireAllowErrors is Fire for tests that expect handler failures: those
// are returned in Result.Err instead of failing the test.
func (h *Harness) FireAllowErrors(ev scripting.Event) scripting.Result {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return h.engine.Invoke(ctx, ev)
}

// CharacterCreate fires events.on_character_create.
func (h *Harness) CharacterCreate(ev scripting.CharacterCreate) scripting.Result {
	h.t.Helper()
	return h.Fire(ev.Event())
}

// Chat fires events.on_chat. Use Player.Ref for msg.Player.
func (h *Harness) Chat(msg scripting.ChatMessage) scripting.Result {
	h.t.Helper()
	return h.Fire(msg.Event())
}

// FirstTalk runs the first-talk handler of npc's template, as when player
// clicks npc. The test fails if the handler fails.
func (h *Harness) FirstTalk(player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("first talk", func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.FirstTalk(ctx, npc.ID, player.Ref(), npc.Ref())
	}, player, npc)
}

// Talk runs the on_talk handler script registered for npc's template, as
// when player clicks "Script <script>" at npc. The test fails if the
// handler fails.
func (h *Harness) Talk(script string, player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("talk", func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.Talk(ctx, script, npc.ID, player.Ref(), npc.Ref())
	}, player, npc)
}

// Event runs the on_event handler of script, as when player clicks
// "Script <script> <event>" near npc (which may be nil). The test fails if
// the handler fails.
func (h *Harness) Event(script, event string, player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("event "+event, func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.Event(ctx, script, event, player.Ref(), npc.Ref())
	}, player, npc)
}

func (h *Harness) npcCall(what string, call func(context.Context) (scripting.Reply, bool), objs ...any) (scripting.Reply, bool) {
	h.t.Helper()
	h.world.Add(objs...)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	reply, ok := call(ctx)
	if reply.Err != nil {
		h.t.Fatalf("%s handler failed: %v", what, reply.Err)
	}
	return reply, ok
}

// Player is a fake player. Change its fields between calls as needed.
type Player struct {
	ObjectID int32
	Name     string
	Level    int
	ClassID  int
	Race     int

	// Quest-era details (scripting.PlayerDetails).
	Sex        int
	BaseClass  int
	Mage       bool
	Newbie     bool
	Dead       bool
	Categories []int

	X, Y, Z, Heading int

	// Items maps item IDs to counts; Equipped paperdoll slots to item IDs.
	Items    map[int]int64
	Equipped map[int]int
	Exp, Sp  int64
	// Vars are the player variables (player:var).
	Vars map[string]string

	// Noble is player:is_noble(); player:set_noble sets it.
	Noble bool
	// ClassIndex is player:class_index(), 0 for the base class; a class
	// index above 0 makes player:is_sub_class_active() true. SubClasses is
	// player:total_sub_classes().
	ClassIndex, SubClasses int
	// Weapon is the item ID of player:active_weapon(); 0 takes the item in
	// Equipped[Paperdoll.RHAND], if any.
	Weapon int
	// Target is player:target(): a *Player, a *Npc or nil.
	Target any
	// Effects are the IDs of the skills whose effects are on the player
	// (player:is_affected_by_skill); skills.apply_effects adds to them.
	Effects []int
	// UserInfos counts player:broadcast_user_info() and
	// player:update_user_info() calls.
	UserInfos int
	// PkKills is player:pk_kills(); player:set_pk_kills sets it.
	PkKills int
	// HP and MaxHP are player:hp() and player:max_hp().
	HP, MaxHP float64
	// Summon is player:summon(): a *Npc or nil.
	Summon *Npc
	// Clan is player:clan() (clan.go); the player leads it when it is the
	// clan's Leader.
	Clan *Clan
	// Offline makes player:is_online() false and hides a clan leader.
	Offline bool
	// Abnormals are the abnormal types of the effects on the player
	// (player:buff_info_by_abnormal_type).
	Abnormals []string
	// GM, Invisible and Hero are player:is_gm(), is_invisible() and
	// is_hero().
	GM, Invisible, Hero bool
	// CursedWeapon is player:is_cursed_weapon_equipped().
	CursedWeapon bool
	// Mount is what the player rides (mount.go), nil on foot.
	Mount *Mount
	// The area scripts' state (areas.go): EffectLevels are the levels of the
	// skills in Effects (1 when missing); Weight is player:weight_penalty();
	// Party the party's members, the leader first (MakeParty), nil alone.
	EffectLevels map[int]int
	Weight       int32
	Party        []*Player
	// InCombat is player:is_in_combat() (areas_b.go).
	InCombat bool
	// HennaWindows records the symbol windows the scripts opened, in order:
	// "equip" (player:show_henna_equip_list()) and "remove" (henna.go).
	HennaWindows []string
	// ClanPrivileges are the Mobius ClanAccess names the player holds
	// (player:has_clan_privilege, castle.go); a clan leader holds all.
	ClanPrivileges []string

	quests map[string]*QuestState
}

// paperdollRHand is Mobius Paperdoll.RHAND.
const paperdollRHand = 7

// Quest returns the player's state of a quest in the fake world, or nil.
func (p *Player) Quest(name string) *QuestState { return p.quests[name] }

// Quests returns the names of the player's quest states, sorted.
func (p *Player) Quests() []string { return slices.Sorted(maps.Keys(p.quests)) }

// Give adds items to the player.
func (p *Player) Give(itemID int, count int64) { p.addItem(itemID, count) }

func (p *Player) addItem(itemID int, count int64) {
	if p.Items == nil {
		p.Items = make(map[int]int64)
	}
	p.Items[itemID] += count
}

// Equip puts an item into a paperdoll slot (the item must be in Items to
// be counted).
func (p *Player) Equip(slot, itemID int) {
	if p.Equipped == nil {
		p.Equipped = make(map[int]int)
	}
	p.Equipped[slot] = itemID
}

// Ref returns the player as the engine sees it; nil for a nil Player. It
// reads the fields at call time.
func (p *Player) Ref() scripting.PlayerRef {
	if p == nil {
		return nil
	}
	return playerRef{p}
}

type playerRef struct{ p *Player }

var (
	_ scripting.PlayerItems    = playerRef{}
	_ scripting.PlayerDetails  = playerRef{}
	_ scripting.PlayerVars     = playerRef{}
	_ scripting.PlayerClasses  = playerRef{}
	_ scripting.PlayerCombat   = playerRef{}
	_ scripting.PlayerNoble    = playerRef{}
	_ scripting.PlayerSubclass = playerRef{}
	_ scripting.PlayerStatus   = playerRef{}
	_ scripting.Located        = playerRef{}
)

func (r playerRef) PkKills() int     { return r.p.PkKills }
func (r playerRef) SetPkKills(n int) { r.p.PkKills = n }
func (r playerRef) UpdateUserInfo()  { r.p.UserInfos++ }

// ActiveQuests returns the IDs of the player's started quests, read from
// their names ("Q00001_LettersOfLove" is 1), sorted.
func (r playerRef) ActiveQuests() []int {
	var ids []int
	for name, st := range r.p.quests {
		if st.state != scripting.StateStarted || st.exited {
			continue
		}
		if id := questIDOf(name); id >= 1 && id <= 19999 {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// questIDOf returns the number after the Q of a quest name, 0 if none.
func questIDOf(name string) int {
	if !strings.HasPrefix(name, "Q") {
		return 0
	}
	digits := name[1:]
	if i := strings.IndexFunc(digits, func(c rune) bool { return c < '0' || c > '9' }); i >= 0 {
		digits = digits[:i]
	}
	id, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return id
}

// SetClassID changes the class; tests read ClassID.
func (r playerRef) SetClassID(id int) { r.p.ClassID = id }

// SetBaseClassID sets BaseClass.
func (r playerRef) SetBaseClassID(id int) { r.p.BaseClass = id }

// BroadcastUserInfo counts in UserInfos.
func (r playerRef) BroadcastUserInfo() { r.p.UserInfos++ }

func (r playerRef) IsNoble() bool       { return r.p.Noble }
func (r playerRef) SetNoble(noble bool) { r.p.Noble = noble }

func (r playerRef) IsSubClassActive() bool { return r.p.ClassIndex > 0 }
func (r playerRef) ClassIndex() int        { return r.p.ClassIndex }
func (r playerRef) TotalSubClasses() int   { return r.p.SubClasses }

func (r playerRef) ActiveWeaponID() int {
	if r.p.Weapon != 0 {
		return r.p.Weapon
	}
	return r.p.Equipped[paperdollRHand]
}

func (r playerRef) TargetObject() any {
	switch t := r.p.Target.(type) {
	case *Player:
		if t != nil {
			return t.Ref()
		}
	case *Npc:
		if t != nil {
			return t.Ref()
		}
	}
	return nil
}

func (r playerRef) ObjectID() int32 { return r.p.ObjectID }
func (r playerRef) Name() string    { return r.p.Name }
func (r playerRef) Level() int      { return r.p.Level }
func (r playerRef) ClassID() int    { return r.p.ClassID }
func (r playerRef) Race() int       { return r.p.Race }

func (r playerRef) Sex() int          { return r.p.Sex }
func (r playerRef) BaseClassID() int  { return r.p.BaseClass }
func (r playerRef) IsMageClass() bool { return r.p.Mage }
func (r playerRef) IsNewbie() bool    { return r.p.Newbie }
func (r playerRef) IsDead() bool      { return r.p.Dead }
func (r playerRef) InCategory(c int) bool {
	return slices.Contains(r.p.Categories, c)
}

func (r playerRef) Location() scripting.Location {
	return scripting.Location{X: r.p.X, Y: r.p.Y, Z: r.p.Z, Heading: r.p.Heading}
}

func (r playerRef) Var(name string) (string, bool) {
	v, ok := r.p.Vars[name]
	return v, ok
}

func (r playerRef) SetVar(name, value string) {
	if r.p.Vars == nil {
		r.p.Vars = make(map[string]string)
	}
	r.p.Vars[name] = value
}

func (r playerRef) Adena() int64               { return r.p.Items[57] }
func (r playerRef) ItemCount(itemID int) int64 { return r.p.Items[itemID] }
func (r playerRef) GiveItems(itemID int, count int64) bool {
	r.p.addItem(itemID, count)
	return true
}

func (r playerRef) TakeItems(itemID int, count int64) bool {
	have := r.p.Items[itemID]
	if count < 0 || count >= have {
		delete(r.p.Items, itemID)
		return count < 0 || count == have
	}
	r.p.Items[itemID] = have - count
	return true
}

// Npc is a fake NPC. ID is its template ID.
type Npc struct {
	ObjectID int32
	ID       int
	Name     string

	X, Y, Z, Heading int
	Dead             bool
	ScriptValue      int
	// Types are the Mobius NPC classes npc:is_type answers true for.
	Types []string
	// Deleted is set by npc:delete().
	Deleted bool

	// Level, HP and MaxHP are npc:level(), npc:hp() and npc:max_hp().
	Level     int
	HP, MaxHP float64
	// Running is set by npc:set_running().
	Running bool
	// Respawn is set by npc:start_respawn() and cleared by
	// npc:stop_respawn().
	Respawn bool
	// Effects are the IDs of the skills whose effects are on the NPC;
	// skills.apply_effects adds to them.
	Effects []int

	// The AI state the NpcControl calls read and write (Plan 13).
	Walking, CoreAIDisabled, Immobilized, Invul, Immortal, NoRandomWalk bool
	// Moving, InCombat and CastingNow are npc:is_moving() and the like.
	Moving, InCombat, CastingNow bool
	// Intention is npc:intention(); "" reads as "ACTIVE". attack, move_to,
	// follow and set_intention set it.
	Intention string
	// Dest is where the last npc:move_to sent the NPC; Harness.Arrive puts
	// it there.
	Dest *scripting.Location
	// Spawn is the spawn point (npc:spawn_location()); nil for none.
	Spawn *scripting.Location
	// AIData are the spawn's AI values (npc:ai_value), Params the template
	// parameters (npc:param).
	AIData map[string]int
	Params map[string]string
	// Hate is the aggro list, most hated first; add_damage_hate adds to it.
	Hate []Hate
	// Leader and Minions are the minion group; npc:add_minion spawns into
	// Minions.
	Leader  *Npc
	Minions []*Npc
	// Box, Interacted and Bombed are a treasure chest's state.
	Box, Interacted, Bombed bool
	// NoReturn is set by npc:set_can_return_to_spawn_point(false).
	NoReturn bool
	// CannotCast makes npc:check_do_cast_conditions false (areas.go).
	CannotCast bool
	// The monster flags (monsters.go): NotLethalable and SeeThroughSilentMove
	// are set by their npc: setters; InactiveRegion makes
	// npc:is_in_active_region() false; ParamMinions are the template's
	// minion lists (npc:param_minions).
	NotLethalable, SeeThroughSilentMove, InactiveRegion bool
	ParamMinions                                        map[string][]scripting.TemplateMinion
	// The boss scripts' state (boss.go): Target is npc:target(), a *Player
	// or *Npc that npc:set_target sets; MP and MaxMP are npc:mp() and
	// npc:max_mp().
	Target    any
	MP, MaxMP float64
	// More area scripts' state (areas_b.go): AttackingNow is
	// npc:is_attacking_now(); DisabledSkills the skill IDs
	// npc:is_skill_disabled answers true for; AggroRange and ClanHelpRange
	// the ranges; ParamSkills the template's skill parameters
	// (npc:param_skill).
	AttackingNow              bool
	DisabledSkills            []int
	AggroRange, ClanHelpRange int
	ParamSkills               map[string]scripting.Skill
	// Castle is npc:castle() (castle.go), nil for none.
	Castle *Castle
	// Hall is clanhall.of_npc(npc) (clanhall.go), nil for none.
	Hall *ClanHall
}

// Hate is one row of a fake NPC's aggro list. Target is a *Player or *Npc.
type Hate struct {
	Target any
	Hate   int64
	Damage int64
}

// Ref returns the NPC as the engine sees it; nil for a nil Npc.
func (n *Npc) Ref() scripting.NpcRef {
	if n == nil {
		return nil
	}
	return npcRef{n}
}

type npcRef struct{ n *Npc }

var (
	_ scripting.NpcDetails = npcRef{}
	_ scripting.NpcStatus  = npcRef{}
	_ scripting.Located    = npcRef{}
)

func (r npcRef) Level() int         { return r.n.Level }
func (r npcRef) CurrentHP() float64 { return r.n.HP }
func (r npcRef) MaxHP() float64     { return r.n.MaxHP }

func (r npcRef) ObjectID() int32 { return r.n.ObjectID }
func (r npcRef) ID() int         { return r.n.ID }
func (r npcRef) Name() string    { return r.n.Name }

func (r npcRef) IsDead() bool             { return r.n.Dead }
func (r npcRef) ScriptValue() int         { return r.n.ScriptValue }
func (r npcRef) SetScriptValue(v int)     { r.n.ScriptValue = v }
func (r npcRef) IsType(class string) bool { return slices.Contains(r.n.Types, class) }
func (r npcRef) Location() scripting.Location {
	return scripting.Location{X: r.n.X, Y: r.n.Y, Z: r.n.Z, Heading: r.n.Heading}
}

// Speech is one npc:say.
type Speech struct {
	NpcID    int   // template ID
	ObjectID int32 // the NPC's object ID
	Text     string
}

// Said returns what NPCs said so far, in order.
func (h *Harness) Said() []Speech {
	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	return slices.Clone(h.out.said)
}

// ClearSaid forgets what NPCs said so far.
func (h *Harness) ClearSaid() {
	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	h.out.said = nil
}

// output is the harness's scripting.Output. It is safe for concurrent use,
// for tests that drive Engine() from a server's goroutines.
type output struct {
	mu   sync.Mutex
	said []Speech
}

func (o *output) NpcSay(npc scripting.NpcRef, text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.said = append(o.said, Speech{NpcID: npc.ID(), ObjectID: npc.ObjectID(), Text: text})
}

// Log is one record written by the engine or a script.
type Log struct {
	Level slog.Level
	Msg   string
	Attrs map[string]any
}

// Logs returns the records logged so far.
func (h *Harness) Logs() []Log { return h.logs.all() }

// ClearLogs forgets the records logged so far.
func (h *Harness) ClearLogs() { h.logs.clear() }

// recorder is a slog.Handler that keeps every record and copies it to the
// test log. Handlers made by WithAttrs share the records. It is safe for
// concurrent use (the deprecated Engine.Call logs from server goroutines).
type recorder struct {
	t     testing.TB
	attrs []slog.Attr

	mu   *sync.Mutex
	logs *[]Log
}

func newRecorder(t testing.TB) *recorder {
	return &recorder{t: t, mu: new(sync.Mutex), logs: new([]Log)}
}

func (r *recorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	l := Log{Level: rec.Level, Msg: rec.Message, Attrs: make(map[string]any)}
	for _, a := range r.attrs {
		l.Attrs[a.Key] = a.Value.Any()
	}
	rec.Attrs(func(a slog.Attr) bool {
		l.Attrs[a.Key] = a.Value.Any()
		return true
	})
	r.t.Logf("%s %s %v", l.Level, l.Msg, l.Attrs)
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.logs = append(*r.logs, l)
	return nil
}

func (r *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *r
	c.attrs = append(slices.Clip(r.attrs), attrs...)
	return &c
}

// WithGroup is not used by the engine; groups are flattened.
func (r *recorder) WithGroup(string) slog.Handler { return r }

func (r *recorder) all() []Log {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(*r.logs)
}

func (r *recorder) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.logs = nil
}
