package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// aiSource uses the AI calls of Mobius scripts/ai (TurekOrcs, FleeMonsters,
// Chests, boss scripts).
const aiSource = `package ai.others;

public class AiCalls extends Script
{
	private static final int[] MOBS = { 20494, 20495 };
	private static final int ZONE = 12002;

	private AiCalls()
	{
		addAttackId(MOBS);
		addEventReceivedId(MOBS);
		addMoveFinishedId(MOBS);
		addNodeArrivedId(MOBS);
	}

	@Override
	public void onAttack(Npc npc, Player attacker, int damage, boolean isSummon)
	{
		npc.disableCoreAI(true);
		npc.setRunning();
		final double radians = Math.toRadians(LocationUtil.calculateAngleFrom(attacker, npc));
		final Location destination = GeoEngine.getInstance().getValidLocation(npc.getX(), npc.getY(), npc.getZ(), 1, 2, 3, npc.getInstanceId());
		npc.getAI().setIntentionMoveTo(destination);
		if (npc.hasAIValue("fleeX") && !npc.getVariables().hasVariable("isHit"))
		{
			npc.getVariables().set("state", 1);
			npc.broadcastEvent("WARNING", 400, attacker);
		}
		if (npc.staysInSpawnLoc())
		{
			npc.asAttackable().returnHome();
		}
		npc.broadcastPacket(new Earthquake(npc.getX(), npc.getY(), npc.getZ(), 20, 10));
		npc.broadcastPacket(new SpecialCamera(npc, 1200, 180, -5, 0, 10000, 15000, 0, 0, 1, 0));
		attacker.sendPacket(new PlaySound(1, "BS02_A", 1, npc.getObjectId(), npc.getX(), npc.getY(), npc.getZ()));
		final int status = GrandBossManager.getInstance().getStatus(29001);
		GrandBossManager.getInstance().setStatus(29001, 1);
		final long respawn = GrandBossManager.getInstance().getStatSet(29001).getLong("respawn_time");
		if (GameTimeTaskManager.getInstance().isNight())
		{
			DoorData.getInstance().getDoor(21240006).openMe();
		}
		ZoneManager.getInstance().getZoneById(ZONE).oustAllPlayers();
		npc.getParameters().getInt("Skill01_ID", 0);
	}

	@Override
	public String onEventReceived(String eventName, Npc sender, Npc receiver, WorldObject reference)
	{
		return super.onEventReceived(eventName, sender, receiver, reference);
	}

	@Override
	public void onMoveFinished(Npc npc)
	{
		npc.disableCoreAI(false);
	}

	@Override
	public void onNodeArrived(Npc npc)
	{
		npc.broadcastSay(ChatType.NPC_GENERAL, "Here");
	}

	public static void main(String[] args)
	{
		new AiCalls();
	}
}
`

func TestAICallTable(t *testing.T) {
	lua, res := translateScript("AiCalls.java", aiSource, "AiCalls", "ai/others/AiCalls")
	if res.err != nil {
		t.Fatal(res.err)
	}
	for _, n := range res.todos {
		t.Errorf("unexpected todo at java:%d: %s", n.line, n.reason)
	}
	for _, want := range []string{
		"from ai/others/AiCalls",
		"q:add_event_received_id(MOBS)",
		"q:add_move_finished_id(MOBS)",
		"q:add_node_arrived_id(MOBS)",
		"npc:disable_core_ai(true)",
		"util.calculate_angle_from(attacker, npc)",
		"world.valid_location({x = npc:x(), y = npc:y(), z = npc:z()}, {x = 1, y = 2, z = 3})",
		`npc:has_ai_value("fleeX")`,
		`npc:has_var("isHit")`,
		`npc:broadcast_event("WARNING", 400, attacker)`,
		"npc:stays_in_spawn_loc()",
		"npc:return_home()",
		"cinematics.earthquake(npc, {x = npc:x(), y = npc:y(), z = npc:z()}, 20, 10)",
		"cinematics.special_camera(npc, npc, 1200, 180, -5, 0, 10000, 15000, 0, 0, 1, 0)",
		`cinematics.sound(attacker, "BS02_A", world.find_object(npc:object_id()))`,
		"grandboss.status(29001)",
		"grandboss.set_status(29001, 1)",
		"grandboss.info(29001).respawn_time",
		"time.is_night()",
		"doors.open(21240006)",
		"zones.oust_all(ZONE)",
		`npc:param_int("Skill01_ID", 0)`,
		"q:on_event_received(function(eventName, sender, receiver, reference)",
		"q:on_move_finished(function(npc)",
		"q:on_node_arrived(function(npc)",
	} {
		if !strings.Contains(lua, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	if t.Failed() {
		t.Log(lua)
	}
}

// The report flags, per script, the Lua calls the definitions file lacks.
func TestFlaggedCalls(t *testing.T) {
	dir := t.TempDir()
	types := filepath.Join(dir, "l2go.lua")
	defs := "---@meta\nworld = {}\nfunction world.find_object(id) end\nfunction Npc:disable_core_ai(on) end\nfunction Script:add_attack_id(...) end\nzones = {}\n"
	if err := os.WriteFile(types, []byte(defs), 0o600); err != nil {
		t.Fatal(err)
	}
	api, err := loadAPICheck(types)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{
		"world.find_object": false, "world.npc_exists": true, "Creature:disable_core_ai": false,
		"Npc:return_home": true, "q:add_attack_id": false, "J.int": false, "rnd": false,
		"zones.oust_all": true, "seven_signs.*": false, "seven_signs.player_cabal": true,
	} {
		if got := api.lacks(name); got != want {
			t.Errorf("lacks(%s) = %v, want %v", name, got, want)
		}
	}
	_, res := translateScript("AiCalls.java", aiSource, "AiCalls", "ai/others/AiCalls")
	api.flag(res)
	if res.missing["Creature:disable_core_ai"] != 0 || res.missing["Npc:broadcast_event"] == 0 {
		t.Errorf("missing = %v", res.missing)
	}
	rep := &report{results: []*questResult{res}}
	text := rep.String()
	for _, want := range []string{"Lua API the server lacks", "    api    Npc:broadcast_event (1 calls)"} {
		if !strings.Contains(text, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	// The real definitions know every call of the AI call table's test.
	real, err := loadAPICheck(filepath.Join("..", "..", "scripts", "types", "l2go.lua"))
	if err != nil {
		t.Fatal(err)
	}
	_, res = translateScript("AiCalls.java", aiSource, "AiCalls", "ai/others/AiCalls")
	real.flag(res)
	if len(res.missing) != 0 {
		t.Errorf("the API lacks %v", res.missing)
	}
}

func TestAIScriptsFolder(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"others/Chests.java", "others/TownPets/TownPets.java", "bosses/QueenAnt/QueenAnt.java"} {
		path := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("class X {}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := aiDir(dir)
	if err != nil || got != dir {
		t.Fatalf("aiDir = %q, %v", got, err)
	}
	all, err := aiScripts(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range all {
		names = append(names, s.rel)
	}
	if want := []string{"bosses/QueenAnt/QueenAnt.java", "others/Chests.java", "others/TownPets/TownPets.java"}; !slices.Equal(names, want) {
		t.Errorf("scripts = %q", names)
	}
	some, err := aiScripts(dir, []string{"Chests", "bosses/QueenAnt/QueenAnt"})
	if err != nil || len(some) != 2 {
		t.Errorf("filtered = %v, %v", some, err)
	}
	if _, err := aiScripts(dir, []string{"Nope"}); err == nil {
		t.Errorf("an unknown script did not fail")
	}
}
