-- Sailren, ported from Mobius ai/bosses/Sailren (BazookaRpm).
--
-- A party leader with Gazkh (8784) talks to Shilen's Stone Statue (32109):
-- the party members near the statue go into Sailren's lair (zone 70049).
-- A minute later three Velociraptors come; their death calls a Pterosaur,
-- its death a Tyrannosaurus, and 3 minutes after the Tyrannosaurus Sailren
-- comes with a 25 s scene in which he cannot be hurt. Killed, Sailren
-- leaves a teleport cube for 5 minutes and comes back after an hour (the
-- global variable SailrenRespawn). A fight ends after 53 minutes, or after
-- 10 minutes without a hit: the players go to town and the lair empties.
-- Sailren is a raid boss: the grand boss store is not used.
--
-- Mobius keeps the raid's status in the script; here a reload finds a
-- fight still going from the dinosaurs and Sailren in the world and starts
-- its timeout again.

local boss = require("lib.boss")

local s = script.new("Sailren")

local STATUE = 32109
local MOVIE = 32110
local SAILREN = 29065
local VELOCIRAPTOR = 22218
local PTEROSAUR = 22199
local TREX = 22217
local TELEPORT_CUBE = 32107

local GAZKH = 8784
local ANIMATION_SKILL = 5090
local ZONE = 70049

local LAIR_CENTER = { x = 27549, y = -6638, z = -2008 }
local DINO_SPAWN = { x = 27313, y = -6766, z = -1975 }
local CUBE_SPAWN = { x = 27644, y = -6638, z = -2008 }

local RESPAWN_MS = 3600000
local MAX_FIGHT_MS = 3200 * 1000
local INACTIVITY_MS = 600000
local INACTIVITY_CHECK_MS = 120000
local VELOCIRAPTOR_DELAY_MS = 60000
local TREX_TO_SAILREN_MS = 180000
local CUBE_LIFETIME_MS = 300000

local RESPAWN_VAR = "SailrenRespawn"

local ALIVE = 0
local IN_FIGHT = 1
local DEAD = 2

s:add_start_npc(STATUE, TELEPORT_CUBE)
s:add_talk_id(STATUE, TELEPORT_CUBE)
s:add_first_talk_id(STATUE)
s:add_kill_id(VELOCIRAPTOR, PTEROSAUR, TREX, SAILREN)
s:add_attack_id(VELOCIRAPTOR, PTEROSAUR, TREX, SAILREN)

local status = ALIVE
local kill_count = 0
local last_attack = 0

local function in_zone(obj)
  return obj ~= nil and zones.is_inside(ZONE, obj)
end

s:on_start(function(ev)
  local remaining = globals.get_long(RESPAWN_VAR, 0) - time.now_ms()
  if remaining > 0 then
    status = DEAD
    s:start_quest_timer("CLEAR_STATUS", remaining, nil, nil)
    return
  end
  status = ALIVE
  -- A reload during a fight: its dinosaurs or Sailren are there.
  for _, id in ipairs({ VELOCIRAPTOR, PTEROSAUR, TREX, SAILREN }) do
    for _, n in ipairs(world.npcs_by_id(id)) do
      if not n:is_dead() and in_zone(n) then
        status = IN_FIGHT
        last_attack = time.now_ms()
      end
    end
  end
  if status == IN_FIGHT then
    s:start_quest_timer("TIME_OUT", MAX_FIGHT_MS, nil, nil)
    s:start_quest_timer("CHECK_ATTACK", INACTIVITY_CHECK_MS, nil, nil)
  end
end)

local function enter(npc, player)
  local party = player:party()
  if party == nil then
    return "32109-01.html"
  elseif status == DEAD then
    return "32109-04.html"
  elseif status == IN_FIGHT then
    return "32109-05.html"
  elseif party.leader:object_id() ~= player:object_id() then
    return "32109-03.html"
  elseif not s:has_quest_items(player, GAZKH) then
    return "32109-02.html"
  end
  s:take_items(player, GAZKH, 1)
  status = IN_FIGHT
  kill_count = 0
  last_attack = time.now_ms()
  for _, member in ipairs(party.members) do
    if member:is_inside_radius_3d(npc, 1000) then
      member:teleport(LAIR_CENTER)
    end
  end
  s:start_quest_timer("SPAWN_VELOCIRAPTORS", VELOCIRAPTOR_DELAY_MS, nil, nil)
  s:start_quest_timer("TIME_OUT", MAX_FIGHT_MS, nil, nil)
  s:start_quest_timer("CHECK_ATTACK", INACTIVITY_CHECK_MS, nil, nil)
  return nil
end

local function camera(npc, ...)
  cinematics.special_camera(ZONE, npc, ...)
end

local function spawn_sailren()
  local sailren = s:add_spawn(SAILREN, LAIR_CENTER, false, 0)
  if sailren == nil then
    log.warn("Sailren: failed to spawn Sailren", "npc", SAILREN)
    return
  end
  local movie = s:add_spawn(MOVIE, { x = sailren:x(), y = sailren:y(), z = sailren:z() + 30, heading = 0 }, false, 26000)
  if movie == nil then
    log.warn("Sailren: failed to spawn the camera NPC", "npc", MOVIE)
    return
  end
  sailren:set_invul(true)
  sailren:set_immobilized(true)
  camera(movie, 60, 110, 30, 4000, 1500, 20000, 0, 65, 1, 0, 0)
  s:start_quest_timer("ATTACK", 24600, sailren, nil)
  s:start_quest_timer("ANIMATION", 2000, movie, nil)
  s:start_quest_timer("CAMERA_1", 4100, movie, nil)
end

-- timeout ends the fight: the players inside go to town, the NPCs inside
-- go.
local function timeout()
  if status == IN_FIGHT then
    status = ALIVE
  end
  for _, p in ipairs(zones.players_inside(ZONE)) do
    p:teleport_to_town()
  end
  for _, n in ipairs(zones.npcs_inside(ZONE)) do
    n:delete()
  end
  kill_count = 0
  last_attack = 0
end

local function check_inactivity()
  if status ~= IN_FIGHT or #zones.players_inside(ZONE) == 0 then
    return
  end
  if last_attack + INACTIVITY_MS < time.now_ms() then
    s:cancel_quest_timer("TIME_OUT", nil, nil)
    s:cancel_quest_timer("CHECK_ATTACK", nil, nil)
    timeout()
  else
    s:start_quest_timer("CHECK_ATTACK", INACTIVITY_CHECK_MS, nil, nil)
  end
end

-- The intro cameras: {next event, delay to it, camera arguments}.
local CAMERAS = {
  CAMERA_1 = { "CAMERA_2", 3000, { 100, 180, 30, 3000, 1500, 20000, 0, 50, 1, 0, 0 } },
  CAMERA_2 = { "CAMERA_3", 3000, { 150, 270, 25, 3000, 1500, 20000, 0, 30, 1, 0, 0 } },
  CAMERA_3 = { "CAMERA_4", 3000, { 160, 360, 20, 3000, 1500, 20000, 10, 15, 1, 0, 0 } },
  CAMERA_4 = { "CAMERA_5", 3000, { 160, 450, 10, 3000, 1500, 20000, 0, 10, 1, 0, 0 } },
  CAMERA_5 = { "CAMERA_6", 7000, { 160, 560, 0, 3000, 1500, 20000, 0, 10, 1, 0, 0 } },
  CAMERA_6 = { nil, 0, { 70, 560, 0, 500, 1500, 7000, -15, 20, 1, 0, 0 } },
}

s:on_event(function(event, npc, player)
  if event == "32109-01.html" or event == "32109-01a.html" or event == "32109-02a.html" or event == "32109-03a.html" then
    return event
  elseif event == "ENTER" then
    return enter(npc, player)
  elseif event == "EXIT" then
    if player ~= nil then
      player:teleport_to_town()
    end
  elseif event == "SPAWN_VELOCIRAPTORS" then
    for _ = 1, 3 do
      local x = DINO_SPAWN.x + rnd(150)
      local y = DINO_SPAWN.y + rnd(150)
      s:add_spawn(VELOCIRAPTOR, { x = x, y = y, z = DINO_SPAWN.z, heading = 0 }, false, 0)
    end
  elseif event == "SPAWN_SAILREN" then
    spawn_sailren()
  elseif event == "ANIMATION" then
    if npc ~= nil then
      npc:set_target(npc)
      npc:cast(skills.get(ANIMATION_SKILL, 1))
      s:start_quest_timer("ANIMATION", 2000, npc, nil)
    end
  elseif CAMERAS[event] ~= nil then
    if npc ~= nil then
      local c = CAMERAS[event]
      camera(npc, unpack(c[3]))
      if c[1] ~= nil then
        s:start_quest_timer(c[1], c[2], npc, nil)
      end
    end
  elseif event == "ATTACK" then
    if npc ~= nil and npc:id() == SAILREN then
      npc:set_invul(false)
      npc:set_immobilized(false)
    end
  elseif event == "CLEAR_STATUS" then
    status = ALIVE
    kill_count = 0
    last_attack = 0
  elseif event == "TIME_OUT" then
    timeout()
  elseif event == "CHECK_ATTACK" then
    check_inactivity()
  end
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if in_zone(attacker) then
    last_attack = time.now_ms()
  end
end)

s:on_kill(function(npc, killer, is_summon)
  if not in_zone(killer) then
    return
  end
  local id = npc:id()
  if id == SAILREN then
    status = DEAD
    s:add_spawn(TELEPORT_CUBE, CUBE_SPAWN, false, CUBE_LIFETIME_MS)
    globals.set(RESPAWN_VAR, time.now_ms() + RESPAWN_MS)
    s:cancel_quest_timer("CHECK_ATTACK", nil, nil)
    s:cancel_quest_timer("TIME_OUT", nil, nil)
    s:start_quest_timer("CLEAR_STATUS", RESPAWN_MS, nil, nil)
    s:start_quest_timer("TIME_OUT", CUBE_LIFETIME_MS, nil, nil)
  elseif id == VELOCIRAPTOR then
    kill_count = kill_count + 1
    if kill_count == 3 then
      local pterosaur = s:add_spawn(PTEROSAUR, DINO_SPAWN, false, 0)
      if pterosaur ~= nil then
        boss.attack_desire(pterosaur, killer)
      end
      kill_count = 0
    end
  elseif id == PTEROSAUR then
    local trex = s:add_spawn(TREX, DINO_SPAWN, false, 0)
    if trex ~= nil then
      boss.attack_desire(trex, killer)
    end
  elseif id == TREX then
    s:start_quest_timer("SPAWN_SAILREN", TREX_TO_SAILREN_MS, nil, nil)
  end
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)
