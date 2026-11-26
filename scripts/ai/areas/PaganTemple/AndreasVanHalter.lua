-- Andreas Van Halter, the raid boss of the Pagan Temple altar, ported from
-- Mobius ai/areas/PaganTemple/AndreasVanHalter (Altur).
--
-- While the boss waits behind the altar doors, players clear the royal
-- guards of the balcony. The 16th kill swaps the doors (balcony closed,
-- altar open), four Altar Gatekeepers and a ritual offering appear, and a
-- cinematic plays: the boss kills the offering with a curse, then the
-- fight starts. Five hours without a hit reset the altar. A boss dragged
-- below the altar floor or to an altar door is put back.
--
-- The state lives in the boss's variables, so a reload keeps it: status
-- (0 dormant, 1 cinematic, 2 fight), kills, last_attack (ms), the object
-- IDs of the offering (victim), the camera and every NPC the ritual
-- spawned (spawned, comma separated).

local s = script.new("AndreasVanHalter")

-- NPCs
local BOSS = 29062
local RITUAL_OFFERING = 32038
local ALTAR_GATEKEEPER = 32051
local CAMERA_DUMMY = 13014
local BALCONY_MOBS = {
  22175, -- Captain of the Royal Guard
  22188, -- Captain of the Royal Guard
  22176, -- Royal Guard
  22189, -- Royal Guard's Aid
}

-- Doors: the balcony doors, and the altar doors with where they stand
-- (Doors.xml; doors never move).
local BALCONY_DOOR_1, BALCONY_DOOR_2 = 19160014, 19160015
local ALTAR_DOOR_1, ALTAR_DOOR_2 = 19160016, 19160017
local ALTAR_DOOR_1_LOC = { x = -15363, y = -54861, z = -10402 }
local ALTAR_DOOR_2_LOC = { x = -17423, y = -54861, z = -10403 }

-- Locations
local RITUAL_VICTIM_SPAWN = { x = -16384, y = -53197, z = -10439, heading = 15992 }
local CAMERA_SPAWN = { x = -16362, y = -53754, z = -10439 }
local BOSS_FALLBACK = { x = -16393, y = -53433, z = -10439 }
local BALCONY_DOOR_1_FALLBACK = { x = -15690, y = -54030, z = -10439 }
local BALCONY_DOOR_2_FALLBACK = { x = -17150, y = -54064, z = -10439 }
local ALTAR_GATEKEEPER_SPAWNS = {
  { x = -17248, y = -54832, z = -10424, heading = 16384 },
  { x = -15547, y = -54835, z = -10424, heading = 16384 },
  { x = -18116, y = -54831, z = -10579, heading = 16384 },
  { x = -14645, y = -54836, z = -10577, heading = 16384 },
}

-- Curse Poison level 7, cast on the offering.
local RITUAL_SKILL, RITUAL_SKILL_LEVEL = 1168, 7

local BALCONY_KILLS_THRESHOLD = 16
local RAID_RESET_DELAY = 5 * 60 * 60 * 1000 -- ms
-- Below the altar floor (-10439): dragged towards the balcony (-10594).
local ANTI_GLITCH_Z = -10500
local ANTI_GLITCH_DOOR_RANGE = 200

local STATUS_DORMANT, STATUS_CINEMATIC, STATUS_FIGHT = 0, 1, 2

s:add_spawn_id(BOSS)
s:add_attack_id(BOSS)
s:add_kill_id(BOSS, BALCONY_MOBS)

local function open_door(id)
  if doors.is_open(id) == false then
    doors.open(id)
  end
end

local function close_door(id)
  if doors.is_open(id) then
    doors.close(id)
  end
end

-- doors_dormant: the balcony open, the altar closed.
local function doors_dormant()
  open_door(BALCONY_DOOR_1)
  open_door(BALCONY_DOOR_2)
  close_door(ALTAR_DOOR_1)
  close_door(ALTAR_DOOR_2)
end

-- live_boss returns the living boss, nil for none.
local function live_boss()
  for _, boss in ipairs(world.npcs_by_id(BOSS)) do
    if not boss:is_dead() then
      return boss
    end
  end
  return nil
end

-- kept returns the NPC whose object ID the boss keeps in name, nil when it
-- is gone.
local function kept(boss, name)
  local id = boss:var_int(name)
  if id == 0 then
    return nil
  end
  return world.find_object(id)
end

-- remember adds a spawned NPC to the ones a reset deletes.
local function remember(boss, npc)
  local list = boss:var("spawned", "")
  boss:set_var("spawned", list == "" and tostring(npc:object_id()) or list .. "," .. npc:object_id())
end

-- despawn_ephemeral deletes what the ritual spawned.
local function despawn_ephemeral(boss)
  for id in boss:var("spawned", ""):gmatch("%d+") do
    local npc = world.find_object(tonumber(id))
    if npc ~= nil and not npc:is_dead() then
      npc:delete()
    end
  end
  boss:unset_var("spawned")
  boss:unset_var("camera")
  boss:unset_var("victim")
end

-- dormant puts the altar back to waiting.
local function dormant(boss)
  boss:set_var("status", STATUS_DORMANT)
  boss:set_var("kills", 0)
  boss:set_var("last_attack", time.now_ms())
end

-- camera shows the players around the boss the cinematic camera.
local function camera(boss, force, angle1, angle2, t, range)
  local cam = kept(boss, "camera")
  if cam ~= nil then
    cinematics.special_camera(boss, cam, force, angle1, angle2, t, range, 0, 0, 1, 0, 0)
  end
end

local function start(boss)
  dormant(boss)
  doors_dormant()
  s:start_quest_timer("RESET_CHECK", 60000, boss, nil, true)
end

s:on_spawn(function(npc)
  start(npc)
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if attacker == nil then
    return
  end
  npc:set_var("last_attack", time.now_ms())
  if npc:z() <= ANTI_GLITCH_Z then
    npc:teleport(BOSS_FALLBACK)
  elseif doors.is_open(ALTAR_DOOR_1) ~= nil and npc:calculate_distance_2d(ALTAR_DOOR_1_LOC) <= ANTI_GLITCH_DOOR_RANGE then
    npc:teleport(BALCONY_DOOR_1_FALLBACK)
  elseif doors.is_open(ALTAR_DOOR_2) ~= nil and npc:calculate_distance_2d(ALTAR_DOOR_2_LOC) <= ANTI_GLITCH_DOOR_RANGE then
    npc:teleport(BALCONY_DOOR_2_FALLBACK)
  end
end)

-- trigger_ritual swaps the doors and starts the cinematic.
local function trigger_ritual(boss)
  boss:set_var("status", STATUS_CINEMATIC)
  close_door(BALCONY_DOOR_1)
  close_door(BALCONY_DOOR_2)
  open_door(ALTAR_DOOR_1)
  open_door(ALTAR_DOOR_2)
  for _, loc in ipairs(ALTAR_GATEKEEPER_SPAWNS) do
    local gatekeeper = s:add_spawn(ALTAR_GATEKEEPER, loc, false, 0)
    if gatekeeper ~= nil then
      remember(boss, gatekeeper)
    end
  end
  local victim = s:add_spawn(RITUAL_OFFERING, RITUAL_VICTIM_SPAWN, false, 0)
  if victim ~= nil then
    remember(boss, victim)
    boss:set_var("victim", victim:object_id())
  end
  s:start_quest_timer("CINEMATIC_1", 3000, boss, nil)
end

s:on_kill(function(npc, killer, is_summon)
  if npc:id() == BOSS then
    -- The raid is over: back to dormant.
    dormant(npc)
    despawn_ephemeral(npc)
    doors_dormant()
    s:cancel_quest_timer("RESET_CHECK", npc, nil)
    return
  end
  local boss = live_boss()
  if boss == nil or boss:var_int("status") ~= STATUS_DORMANT then
    return
  end
  local kills = boss:var_int("kills") + 1
  boss:set_var("kills", kills)
  if kills == BALCONY_KILLS_THRESHOLD then
    trigger_ritual(boss)
  end
end)

-- end_cinematic lets the boss fight.
local function end_cinematic(boss)
  boss:set_immobilized(false)
  local cam = kept(boss, "camera")
  if cam ~= nil then
    cam:delete()
  end
  boss:unset_var("camera")
  boss:set_var("status", STATUS_FIGHT)
end

s:on_event(function(event, npc, player)
  if npc == nil or not npc:valid() then
    return nil
  end
  local boss = npc
  if event == "CINEMATIC_1" then
    local victim = kept(boss, "victim")
    if boss:is_dead() or victim == nil then
      return nil
    end
    boss:set_heading(16384)
    local cam = s:add_spawn(CAMERA_DUMMY, CAMERA_SPAWN, false, 30000)
    if cam ~= nil then
      remember(boss, cam)
      boss:set_var("camera", cam:object_id())
    end
    boss:set_target(victim)
    boss:cast(skills.get(RITUAL_SKILL, RITUAL_SKILL_LEVEL))
    boss:set_immobilized(true)
    cinematics.sound(boss, "BS04_A", boss)
    camera(boss, 1500, 88, 89, 0, 5000)
    s:start_quest_timer("CINEMATIC_2", 300, boss, nil)
  elseif event == "CINEMATIC_2" then
    local victim = kept(boss, "victim")
    if victim ~= nil and not victim:is_dead() then
      victim:social_action(1)
      victim:set_hp_mp(0, victim:mp())
    end
    camera(boss, 1500, 88, 89, 0, 5000)
    s:start_quest_timer("CINEMATIC_3", 300, boss, nil)
  elseif event == "CINEMATIC_3" then
    local victim = kept(boss, "victim")
    if victim ~= nil and not victim:is_dead() then
      victim:delete()
    end
    boss:unset_var("victim")
    camera(boss, 450, 88, 3, 5500, 5000)
    s:start_quest_timer("CINEMATIC_4", 9400, boss, nil)
  elseif event == "CINEMATIC_4" then
    camera(boss, 500, 88, 4, 5000, 5000)
    s:start_quest_timer("CINEMATIC_5", 5000, boss, nil)
  elseif event == "CINEMATIC_5" then
    camera(boss, 3000, 88, 4, 6000, 5000)
    s:start_quest_timer("CINEMATIC_END", 6000, boss, nil)
  elseif event == "CINEMATIC_END" then
    -- The acolytes (29063, 29064) come from the boss's minion list.
    end_cinematic(boss)
  elseif event == "RESET_CHECK" then
    if not boss:is_dead() and not boss:is_attacking_now()
      and time.now_ms() - boss:var_int("last_attack") >= RAID_RESET_DELAY then
      dormant(boss)
      despawn_ephemeral(boss)
      doors_dormant()
    end
  end
  return nil
end)

-- After a reload the boss keeps its state but not its timers: the reset
-- check starts again, and a cinematic cut short ends at once. A boss that
-- spawned before the script was there starts as on_spawn would.
s:on_start(function(ev)
  local boss = live_boss()
  if boss == nil then
    return
  end
  if not boss:has_var("status") then
    start(boss)
    return
  end
  s:start_quest_timer("RESET_CHECK", 60000, boss, nil, true)
  if ev.reload and boss:var_int("status") == STATUS_CINEMATIC then
    local victim = kept(boss, "victim")
    if victim ~= nil and not victim:is_dead() then
      victim:delete()
    end
    boss:unset_var("victim")
    end_cinematic(boss)
  end
end)
