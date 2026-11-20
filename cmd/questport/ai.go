package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The AI scripts (Plan 13, Phase 10.6): the call table of the NPC-control,
// world, state and cinematics API, the AI callbacks, the scan of Mobius's
// scripts/ai folder, and the check of the Lua calls a translation makes
// against the API in scripts/types/l2go.lua (the "flagged calls" of the
// report).

func init() {
	for java, lua := range map[string]string{
		"onEventReceived": "on_event_received",
		"onNodeArrived":   "on_node_arrived",
		"onRouteFinished": "on_route_finished",
	} {
		callbacks[java] = lua
	}
}

// aiMappings is the AI call table: Mobius NPC control, world, state and
// cinematics calls of the scripts under ai/, and the Lua API they map to.
var aiMappings = []*mapEntry{
	// --- Registration ---
	m("Quest", "addEventReceivedId", "int...", "q:add_event_received_id($1*)", "").k(),
	m("Quest", "addNodeArrivedId", "int...", "q:add_node_arrived_id($1*)", "").k(),
	m("Quest", "addRouteFinishedId", "int...", "q:add_route_finished_id($1*)", "").k(),

	// --- NPC control ---
	m("Creature", "disableCoreAI", "bool", "$r:disable_core_ai($1)", ""),
	m("Creature", "isCoreAIDisabled", "", "$r:is_core_ai_disabled()", "boolean"),
	m("Creature", "setMortal", "bool", "$r:set_mortal($1)", ""),
	m("Creature", "isMortal", "", "$r:is_mortal()", "boolean"),
	m("Creature", "setHeading", "int", "$r:set_heading($1)", ""),
	m("Npc", "setRandomWalking", "bool", "$r:set_random_walking($1)", ""),
	m("Attackable", "returnHome", "", "$r:return_home()", ""),
	m("Attackable", "stopHating", "Creature", "$r:stop_hating($1)", ""),
	m("Attackable", "reduceHate", "Creature,int", "$r:reduce_hate($1, $2)", ""),
	m("Npc", "staysInSpawnLoc", "", "$r:stays_in_spawn_loc()", "boolean"),
	m("Npc", "getAIValue", "str", "$r:ai_value($1)", "int"),
	m("Npc", "hasAIValue", "str", "$r:has_ai_value($1)", "boolean"),
	m("Npc", "broadcastEvent", "str,int,any", "$r:broadcast_event($1, $2, $3)", ""),
	m("Npc", "sendScriptEvent", "str,any,any", "$r:send_event($1, $2, $3)", ""),
	m("Npc", "getParameters", "", "$r", "NpcParameters").asView(),
	m("NpcParameters", "getInt", "str,int", "$b:param_int($1, $2)", "int"),
	m("NpcParameters", "getInt", "str", "$b:param_int($1)", "int"),
	m("NpcParameters", "getString", "str,str", "$b:param($1, $2)", "String"),
	m("NpcParameters", "getString", "str", "$b:param($1)", "String"),
	m("NpcVariables", "hasVariable", "str", "$b:has_var($1)", "boolean"),
	m("Monster", "getLeader", "", "$r:leader()", "Monster"),
	m("Npc", "getLeader", "", "$r:leader()", "Npc"),
	m("Chest", "isBox", "", "$r:is_box()", "boolean"),
	m("Chest", "isInteracted", "", "$r:is_interacted()", "boolean"),
	m("Chest", "setInteracted", "", "$r:set_interacted()", ""),
	m("Chest", "treasureBomb", "", "$r:treasure_bomb()", ""),
	m("Skill", "getTargetType", "", "$r:target_type()", "String"),

	// --- Java: the geometry of moving NPCs ---
	m("static:Math", "toRadians", "num", "math.rad($1)", "double"),
	m("static:Math", "toDegrees", "num", "math.deg($1)", "double"),
	m("static:Math", "cos", "num", "math.cos($1)", "double"),
	m("static:Math", "sin", "num", "math.sin($1)", "double"),
	m("static:Math", "atan2", "num,num", "math.atan2($1, $2)", "double"),

	// --- World ---
	m("static:LocationUtil", "calculateAngleFrom", "pos,pos", "util.calculate_angle_from($1, $2)", "double"),
	m("static:GeoEngine", "getInstance", "", "world", "GeoEngine").asView(),
	m("GeoEngine", "getValidLocation", "int,int,int,int,int,int,int", "world.valid_location({x = $1, y = $2, z = $3}, {x = $4, y = $5, z = $6})", "Location"),
	m("static:DoorData", "getInstance", "", "doors", "DoorData").asView(),
	// A door is its ID.
	fnm("DoorData", "getDoor", "int", "Door", argAs("Door")),
	m("Door", "openMe", "", "doors.open($r)", "").w(),
	m("Door", "closeMe", "", "doors.close($r)", "").w(),
	m("Door", "isOpen", "", "doors.is_open($r)", "boolean"),
	m("static:GameTimeTaskManager", "getInstance", "", "time", "GameTime").asView(),
	m("GameTime", "isNight", "", "time.is_night()", "boolean"),
	m("GameTime", "getGameTime", "", "time.game_time()", "int"),
	m("GameTime", "getGameHour", "", "time.game_hour()", "int"),
	m("GameTime", "getGameTicks", "", "time.game_ticks()", "int"),
	// A zone from ZoneManager is its ID.
	m("static:ZoneManager", "getInstance", "", "zones", "ZoneManager").asView(),
	fnm("ZoneManager", "getZoneById", "int", "ZoneRef", argAs("ZoneRef")),
	fnm("ZoneManager", "getZoneById", "int,any", "ZoneRef", argAs("ZoneRef")),
	fnm("ZoneRef", "getId", "", "int", same("int")),
	m("ZoneRef", "getPlayersInside", "", "zones.players_inside($r)", "List<Player>"),
	m("ZoneRef", "isInsideZone", "WorldObject", "zones.is_inside($r, $1)", "boolean"),
	m("ZoneRef", "movePlayersTo", "pos", "zones.move_players($r, $1)", "").w(),
	m("ZoneRef", "oustAllPlayers", "", "zones.oust_all($r)", "").w(),
	m("ZoneRef", "allowPlayerEntry", "Player,int", "zones.allow_entry($r, $1, $2)", ""),
	m("ZoneRef", "removePlayer", "Player", "zones.remove_player($r, $1)", ""),
	fnm("ZoneRef", "broadcastPacket", "any", "", broadcastPacketFn),

	// --- State ---
	m("static:GrandBossManager", "getInstance", "", "grandboss", "GrandBossManager").asView(),
	m("GrandBossManager", "getStatus", "int", "grandboss.status($1)", "int").w(),
	m("GrandBossManager", "setStatus", "int,int", "grandboss.set_status($1, $2)", "").w(),
	m("GrandBossManager", "getBoss", "int", "grandboss.boss($1)", "GrandBoss").w(),
	m("GrandBossManager", "addBoss", "any", "grandboss.add_boss($1)", "").w(),
	m("GrandBossManager", "getStatSet", "int", "grandboss.info($1)", "BossInfo").w(),
	m("GrandBossManager", "setStatSet", "int,any", "grandboss.set_info($1, $2)", "").w(),
	fnm("BossInfo", "getLong", "str", "long", bossInfoField),
	fnm("BossInfo", "getInt", "str", "int", bossInfoField),
	fnm("BossInfo", "getDouble", "str", "double", bossInfoField),
}

// isScriptMain reports whether md is Mobius's public static void
// main(String[] args) that only creates the script.
func isScriptMain(md *methodDecl) bool {
	if md.name != "main" || !md.mods.has("static") || len(md.params) != 1 || md.body == nil {
		return false
	}
	for _, s := range md.body.stmts {
		es, ok := s.(*exprStmt)
		if !ok {
			return false
		}
		if _, ok := es.x.(*newExpr); !ok {
			return false
		}
	}
	return true
}

// bossInfoFields maps Mobius grandboss StatSet keys to BossInfo fields.
var bossInfoFields = map[string]string{
	"respawn_time": "respawn_time", "loc_x": "x", "loc_y": "y", "loc_z": "z",
	"heading": "heading", "currentHP": "hp", "currentMP": "mp",
}

// bossInfoField is info.getLong("respawn_time") and the like: a field of
// the grandboss.info table.
func bossInfoField(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	key, ok := t.constValue(c.args[0])
	field := bossInfoFields[strings.Trim(key, `"`)]
	if !ok || field == "" {
		return t.todo(c.line, "call BossInfo."+c.name, "grand boss StatSet key other than "+strings.Join(sortedKeys(bossInfoFields), ", "))
	}
	return lx{code: asAtom(recv) + "." + field, prec: precAtom, t: tyUnknown}
}

// cinematicCall turns a cinematic packet broadcast around (or sent to) to
// into a cinematics call; to may also be a zone ID (ZoneRef).
func cinematicCall(t *translator, to lx, p lx) (lx, bool) {
	pa := p.pargs
	var fn, args string
	switch {
	case p.packet == "PlaySound" && len(pa) == 1:
		fn, args = "sound", pa[0].code
	case p.packet == "PlaySound" && len(pa) == 7:
		// PlaySound(1, name, 1, objectId, x, y, z): the sound comes from the
		// object.
		fn, args = "sound", pa[1].code+", world.find_object("+pa[3].code+")"
	case p.packet == "Earthquake" && len(pa) == 5:
		fn, args = "earthquake", "{x = "+pa[0].code+", y = "+pa[1].code+", z = "+pa[2].code+"}, "+pa[3].code+", "+pa[4].code
	case p.packet == "SpecialCamera" && (len(pa) == 11 || len(pa) == 12) && (isIntegral(pa[1].t) || isUnknown(pa[1].t)):
		fn, args = "special_camera", joinCodes(pa)
	case p.packet == "ExShowScreenMessage" && len(pa) == 2:
		fn, args = "screen_message", pa[0].code+", "+pa[1].code
	case p.packet == "ExShowScreenMessage" && len(pa) == 3 && !isA(pa[0].t, "SystemMessageId"):
		fn, args = "screen_message", pa[0].code+", "+pa[2].code+", "+pa[1].code
	default:
		return lx{}, false
	}
	t.useAPI("cinematics." + fn)
	return lx{code: "cinematics." + fn + "(" + to.code + ", " + args + ")", prec: precAtom, t: tyVoid, call: true}, true
}

// allMappings is the quest call table and the AI call table.
func allMappings() []*mapEntry {
	return append(slices.Clip(mappings), aiMappings...)
}

// --- The AI scripts folder ---

// aiDir finds Mobius's scripts/ai folder under a checkout.
func aiDir(mobius string) (string, error) {
	for _, d := range []string{
		filepath.Join(mobius, "dist", "game", "data", "scripts", "ai"),
		filepath.Join(mobius, "L2J_Mobius_CT_0_Interlude", "dist", "game", "data", "scripts", "ai"),
		mobius,
	} {
		if st, err := os.Stat(filepath.Join(d, "others")); err == nil && st.IsDir() {
			return d, nil
		}
	}
	return "", fmt.Errorf("no scripts/ai folder found under %s", mobius)
}

// aiScript is one Java file of the AI folder.
type aiScript struct {
	name string // class name, e.g. "Chests"
	rel  string // slash path of the file relative to the AI folder
}

// aiScripts lists the Java files under dir, or the ones named in only
// (class names, or relative paths without .java).
func aiScripts(dir string, only []string) ([]aiScript, error) {
	var all []aiScript
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		all = append(all, aiScript{name: strings.TrimSuffix(d.Name(), ".java"), rel: rel})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the AI folder: %w", err)
	}
	slices.SortFunc(all, func(a, b aiScript) int { return strings.Compare(a.rel, b.rel) })
	if len(only) == 0 {
		return all, nil
	}
	var out []aiScript
	for _, want := range only {
		found := false
		for _, s := range all {
			if s.name == want || strings.TrimSuffix(s.rel, ".java") == want {
				out = append(out, s)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("AI script %s not found in %s", want, dir)
		}
	}
	return out, nil
}

// translateAIAll translates the AI scripts into out/<folder>/<Name>.lua.
func translateAIAll(dir string, scripts []aiScript, out string, api *apiCheck) (*report, error) {
	rep := &report{}
	for _, s := range scripts {
		path := filepath.Join(dir, filepath.FromSlash(s.rel))
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		lua, res := translateScript(path, string(src), s.name, "ai/"+strings.TrimSuffix(s.rel, ".java"))
		api.flag(res)
		rep.results = append(rep.results, res)
		if res.err != nil {
			continue
		}
		target := filepath.Join(out, filepath.FromSlash(pathpkg.Dir(s.rel)), s.name+".lua")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // the script folder is meant to be readable
			return nil, fmt.Errorf("create output folder: %w", err)
		}
		if err := os.WriteFile(target, []byte(lua), 0o644); err != nil { //nolint:gosec // generated scripts are public
			return nil, fmt.Errorf("write %s: %w", target, err)
		}
		res.out = target
	}
	return rep, nil
}

// --- Flagged calls: the Lua API a translation needs that the server lacks ---

// apiCheck knows the Lua API from the definitions file.
type apiCheck struct {
	funcs   map[string]bool // "module.fn"
	methods map[string]bool // method names of any class
	modules map[string]bool // module tables
}

var (
	typesFuncDecl   = regexp.MustCompile(`^function ([A-Za-z_]\w*)([.:])([A-Za-z_]\w*)\(`)
	typesModuleDecl = regexp.MustCompile(`^([a-z_]\w*) = \{\}$`)
)

// loadAPICheck reads the definitions file; a nil check flags nothing.
func loadAPICheck(path string) (*apiCheck, error) {
	f, err := os.Open(path) //nolint:gosec // a path the user gave
	if err != nil {
		return nil, fmt.Errorf("read the API definitions: %w", err)
	}
	defer f.Close()
	c := &apiCheck{funcs: map[string]bool{}, methods: map[string]bool{}, modules: map[string]bool{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if m := typesFuncDecl.FindStringSubmatch(line); m != nil {
			if m[2] == ":" {
				c.methods[m[3]] = true
			} else {
				c.funcs[m[1]+"."+m[3]] = true
			}
		}
		if m := typesModuleDecl.FindStringSubmatch(line); m != nil {
			c.modules[m[1]] = true
		}
	}
	return c, sc.Err()
}

// lacks reports whether the server's API lacks a Lua call the report
// counted ("Npc:attack", "world.npc_exists", "q:give_items"). Calls of
// Lua's own libraries and lib/java.lua are not checked.
func (c *apiCheck) lacks(name string) bool {
	if c == nil || name == "" {
		return false
	}
	if i := strings.IndexByte(name, ':'); i >= 0 {
		return !c.methods[name[i+1:]]
	}
	mod, fn, ok := strings.Cut(name, ".")
	if !ok {
		return false
	}
	switch mod {
	case "J", "table", "string", "math":
		return false
	}
	if !c.modules[mod] {
		return fn != "*"
	}
	return fn != "*" && !c.funcs[name]
}

// flag records the calls of res the API lacks.
func (c *apiCheck) flag(res *questResult) {
	if c == nil {
		return
	}
	for name, n := range res.api {
		if c.lacks(name) {
			if res.missing == nil {
				res.missing = map[string]int{}
			}
			res.missing[name] = n
		}
	}
}
