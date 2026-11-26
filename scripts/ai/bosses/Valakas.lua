-- Valakas, ported from Mobius ai/bosses/Valakas (BazookaRpm).
--
-- Valakas sleeps hidden and invulnerable (status 0). The first player let
-- into his lair (ValakasTeleporters.lua) wakes him: after
-- Config.VALAKAS_WAIT_TIME minutes (status 1) he flies into the lair with
-- a 26 s scene and fights (status 2). Every minute he picks his
-- regeneration by HP and, after 15 minutes without a hit, goes back to
-- sleep and the lair is emptied. Every 2 s he picks a target and a skill.
-- Players hitting him from outside the lair die. Killed (status 3), he
-- comes back after Config.VALAKAS_SPAWN_INTERVAL hours, give or take
-- VALAKAS_SPAWN_RANDOM; teleport cubes stand for 15 minutes, then the lair
-- is emptied.
--
-- A strider rider hitting him gets Hinder Strider (4258) unless it has it.
--
-- Not ported: setCastingNow(true) before a cast. The surrounding-region
-- test of his target is "still among his visible players".

local boss = require("lib.boss")

local s = script.new("Valakas")

local VALAKAS = 29028
local TELEPORT_CUBE = 31759

local SKILL_REGEN = 4691
local SKILL_LAVA_SKIN = 4680
local SKILL_ANTI_STRIDER = 4258 -- Hinder Strider
local MOUNT_STRIDER = 1 -- player:mount_type()
local SKILLS_REGULAR = { 4681, 4682, 4683, 4689 }
local SKILLS_LOW_HP = { 4681, 4682, 4683, 4689, 4690 }
local SKILLS_AOE = { 4683, 4684, 4685, 4686, 4688, 4689, 4690 }

local DORMANT = 0
local WAITING = 1
local FIGHTING = 2
local DEAD = 3

local LAIR = 12010
local GROUND = 13010

local CUBES = {
  { x = 214880, y = -116144, z = -1644 }, { x = 213696, y = -116592, z = -1644 }, { x = 212112, y = -116688, z = -1644 },
  { x = 211184, y = -115472, z = -1664 }, { x = 210336, y = -114592, z = -1644 }, { x = 211360, y = -113904, z = -1644 },
  { x = 213152, y = -112352, z = -1644 }, { x = 214032, y = -113232, z = -1644 }, { x = 214752, y = -114592, z = -1644 },
  { x = 209824, y = -115568, z = -1421 }, { x = 210528, y = -112192, z = -1403 }, { x = 213120, y = -111136, z = -1408 },
  { x = 215184, y = -111504, z = -1392 }, { x = 215456, y = -117328, z = -1392 }, { x = 213200, y = -118160, z = -1424 },
}

local HIDE = { x = 220963, y = -104895, z = -1620 }
local EJECT = { x = 150037, y = -57255, z = -2976 }
local LAIR_SPOT = { x = 212852, y = -114842, z = -1632 }
local REGEN_SPOT = { x = -105200, y = -253104, z = -15264 }

local EVT_START = "VALAKAS_START_ENCOUNTER"
local EVT_REGEN_TICK = "VALAKAS_REGEN_TICK"
local EVT_SKILL_TICK = "VALAKAS_SKILL_TICK"
local EVT_BROADCAST_SPAWN = "VALAKAS_BROADCAST_SPAWN"
local EVT_UNLOCK = "VALAKAS_UNLOCK"
local EVT_REMOVE_PLAYERS = "VALAKAS_REMOVE_PLAYERS"

local INACTIVITY_MS = 900000
local REGEN_TICK_MS = 60000
local SKILL_TICK_MS = 2000
local CUBE_LIFETIME_MS = 900000

-- The intro (VALAKAS_SPAWN_1..9) and death (VALAKAS_DIE_1..8) cameras:
-- {delay, distance, yaw, pitch, time, duration, unk, rel_yaw, rel_pitch,
-- wide, unk2, unk3}.
local SPAWN_STEPS = {
  { 1700, 1800, 180, -1, 1500, 15000, 10000, 0, 0, 1, 0, 0 },
  { 3200, 1300, 180, -5, 3000, 15000, 10000, 0, -5, 1, 0, 0 },
  { 6500, 500, 180, -8, 600, 15000, 10000, 0, 60, 1, 0, 0 },
  { 9400, 800, 180, -8, 2700, 15000, 10000, 0, 30, 1, 0, 0 },
  { 12100, 200, 250, 70, 0, 15000, 10000, 30, 80, 1, 0, 0 },
  { 12430, 1100, 250, 70, 2500, 15000, 10000, 30, 80, 1, 0, 0 },
  { 15430, 700, 150, 30, 0, 15000, 10000, -10, 60, 1, 0, 0 },
  { 16830, 1200, 150, 20, 2900, 15000, 10000, -10, 30, 1, 0, 0 },
  { 23530, 750, 170, -10, 3400, 15000, 4000, 10, -15, 1, 0, 0 },
}
local SPAWN_STEP_10_MS = 26000
local DIE_STEPS = {
  { 300, 2000, 130, -1, 0, 15000, 10000, 0, 0, 1, 1, 0 },
  { 600, 1100, 210, -5, 3000, 15000, 10000, -13, 0, 1, 1, 0 },
  { 3800, 1300, 200, -8, 3000, 15000, 10000, 0, 15, 1, 1, 0 },
  { 8200, 1000, 190, 0, 500, 15000, 10000, 0, 10, 1, 1, 0 },
  { 8700, 1700, 120, 0, 2500, 15000, 10000, 12, 40, 1, 1, 0 },
  { 13300, 1700, 20, 0, 700, 15000, 10000, 10, 10, 1, 1, 0 },
  { 14000, 1700, 10, 0, 1000, 15000, 10000, 20, 70, 1, 1, 0 },
  { 16500, 1700, 10, 0, 300, 15000, 250, 20, -20, 1, 1, 0 },
}

s:add_attack_id(VALAKAS)
s:add_kill_id(VALAKAS)
s:add_spawn_id(VALAKAS)
s:add_spell_finished_id(VALAKAS)
s:add_event_received_id(VALAKAS)

local valakas
local current_target
local last_attack = 0

local function camera(npc, c)
  cinematics.special_camera(LAIR, npc, c[2], c[3], c[4], c[5], c[6], c[7], c[8], c[9], c[10], c[11], c[12])
end

local function fight_timers(npc)
  s:start_quest_timer(EVT_REGEN_TICK, REGEN_TICK_MS, npc, nil, true)
  s:start_quest_timer(EVT_SKILL_TICK, SKILL_TICK_MS, npc, nil, true)
end

local function hide(npc)
  npc:teleport(HIDE)
  npc:set_invul(true)
  npc:set_intention(Intention.IDLE)
end

local function spawn_dormant()
  valakas = s:add_spawn(VALAKAS, REGEN_SPOT, false, 0)
  valakas:teleport(HIDE)
  valakas:set_invul(true)
  valakas:set_running()
  valakas:set_intention(Intention.IDLE)
  grandboss.add_boss(valakas)
  grandboss.set_status(VALAKAS, DORMANT)
end

s:on_start(function(ev)
  boss.start(s, VALAKAS, {
    dead = DEAD,
    unlock = EVT_UNLOCK,
    live = function(npc)
      -- A reload: Valakas is there; his timers by his status.
      valakas = npc
      local st = grandboss.status(VALAKAS)
      if st == FIGHTING then
        last_attack = time.now_ms()
        fight_timers(npc)
      elseif st == WAITING then
        s:start_quest_timer(EVT_START, Config.VALAKAS_WAIT_TIME * 60000, npc, nil)
      end
    end,
    unlocked = spawn_dormant,
    restore = function(info)
      valakas = boss.spawn_saved(s, VALAKAS, info)
      grandboss.add_boss(valakas)
      valakas:set_running()
      local st = grandboss.status(VALAKAS)
      if st == FIGHTING then
        last_attack = time.now_ms()
        fight_timers(valakas)
      else
        hide(valakas)
        if st == WAITING then
          s:start_quest_timer(EVT_START, Config.VALAKAS_WAIT_TIME * 60000, valakas, nil)
        end
      end
    end,
  })
end)

-- The first player let into the lair (ValakasTeleporters.lua) starts the
-- wait.
s:on_event_received(function(event, sender, receiver, reference)
  if event == "VALAKAS_WAIT" then
    s:start_quest_timer(EVT_START, Config.VALAKAS_WAIT_TIME * 60000, receiver, nil)
  end
end)

local function encounter_start(npc)
  last_attack = time.now_ms()
  npc:teleport(LAIR_SPOT)
  s:start_quest_timer(EVT_BROADCAST_SPAWN, 100, npc, nil)
  for i, step in ipairs(SPAWN_STEPS) do
    s:start_quest_timer("VALAKAS_SPAWN_" .. i, step[1], npc, nil)
  end
  s:start_quest_timer("VALAKAS_SPAWN_10", SPAWN_STEP_10_MS, npc, nil)
end

local function regen_tick(npc)
  if grandboss.status(VALAKAS) == FIGHTING and last_attack + INACTIVITY_MS < time.now_ms() then
    npc:set_intention(Intention.IDLE)
    npc:teleport(REGEN_SPOT)
    grandboss.set_status(VALAKAS, DORMANT)
    npc:set_hp_mp(npc:max_hp(), npc:max_mp())
    zones.oust_all(LAIR)
    s:cancel_quest_timer(EVT_REGEN_TICK, npc, nil)
    s:cancel_quest_timer(EVT_SKILL_TICK, npc, nil)
    return
  end
  local level = npc:effect_level(SKILL_REGEN) or 0
  local max, hp = npc:max_hp(), npc:hp()
  local want
  if hp < max / 4.0 and level ~= 4 then
    want = 4
  elseif hp < max * 2.0 / 4.0 and level ~= 3 then
    want = 3
  elseif hp < max * 3.0 / 4.0 and level ~= 2 then
    want = 2
  elseif level ~= 1 then
    want = 1
  end
  if want ~= nil then
    npc:set_target(npc)
    npc:cast(skills.get(SKILL_REGEN, want))
  end
end

-- random_target is a living, visible player around Valakas, or nil.
local function random_target(npc)
  local candidates = {}
  for _, p in ipairs(world.visible_objects(npc, "Playable")) do
    if not p:is_dead() and not p:is_invisible() then
      candidates[#candidates + 1] = p
    end
  end
  return rnd.entry(candidates)
end

local function visible(npc, target)
  for _, p in ipairs(world.visible_objects(npc, "Playable")) do
    if p:object_id() == target:object_id() then
      return true
    end
  end
  return false
end

local function random_skill(npc)
  local hp_percent = math.floor(npc:hp() / npc:max_hp() * 100)
  if hp_percent < 75 and rnd(150) == 0 and not npc:is_affected_by_skill(SKILL_LAVA_SKIN) then
    return SKILL_LAVA_SKIN
  end
  if #world.players_around(npc, 1200) >= 20 then
    return rnd.entry(SKILLS_AOE)
  end
  if hp_percent > 50 then
    return rnd.entry(SKILLS_REGULAR)
  end
  return rnd.entry(SKILLS_LOW_HP)
end

-- skill_ai picks a target and a skill, and casts or follows (Mobius
-- callSkillAI).
local function skill_ai(npc)
  if npc == nil or npc:is_invul() or npc:is_casting_now() then
    return
  end
  -- current_target is an object ID: players expire with the handler.
  local target = current_target and world.get_player(current_target)
  if target == nil or target:is_dead() or not visible(npc, target) or rnd(10) == 0 then
    target = random_target(npc)
    current_target = target and target:object_id()
  end
  if target == nil then
    if rnd(10) == 0 then
      local from = { x = npc:x(), y = npc:y(), z = npc:z() }
      local to = { x = from.x + rnd(-1400, 1400), y = from.y + rnd(-1400, 1400), z = from.z }
      local valid = world.valid_location(from, to)
      if valid.x == to.x and valid.y == to.y then
        npc:move_to(to)
      end
    end
    return
  end
  local skill = skills.get(random_skill(npc), 1)
  if skill == nil then
    return
  end
  local range = math.max(skill:cast_range(), 600)
  if util.in_range(range, npc, target, true) then
    npc:set_intention(Intention.IDLE)
    npc:set_target(target)
    npc:cast(skill)
  else
    npc:follow(target)
  end
end

local function spawn_step_10(npc)
  grandboss.set_status(VALAKAS, FIGHTING)
  npc:set_invul(false)
  fight_timers(npc)
  if Config.VALAKAS_RECOGNIZE_HERO then
    for _, p in ipairs(zones.players_inside(LAIR)) do
      if p:is_hero() then
        cinematics.screen_message(LAIR, p:name() .. "!!!! You cannot hope to defeat me with your meager strength.", 4000, 2)
        break
      end
    end
  end
end

s:on_event(function(event, npc, player)
  if npc == nil then
    if event == EVT_UNLOCK then
      spawn_dormant()
    elseif event == EVT_REMOVE_PLAYERS then
      zones.oust_all(LAIR)
    end
    return
  end
  if event == EVT_START then
    encounter_start(npc)
  elseif event == EVT_REGEN_TICK then
    regen_tick(npc)
  elseif event == EVT_SKILL_TICK then
    skill_ai(npc)
  elseif event == EVT_BROADCAST_SPAWN then
    for _, p in ipairs(zones.players_inside(LAIR)) do
      cinematics.sound(p, "BS03_A")
    end
    npc:social_action(3)
  elseif event == "VALAKAS_SPAWN_10" then
    spawn_step_10(npc)
  else
    local m = event:match("^VALAKAS_SPAWN_(%d+)$")
    local step = m and tonumber(m)
    if step ~= nil and SPAWN_STEPS[step] ~= nil then
      camera(npc, SPAWN_STEPS[step])
      return
    end
    m = event:match("^VALAKAS_DIE_(%d+)$")
    step = m and tonumber(m)
    if step ~= nil and DIE_STEPS[step] ~= nil then
      camera(npc, DIE_STEPS[step])
      if step == #DIE_STEPS then
        for _, loc in ipairs(CUBES) do
          s:add_spawn(TELEPORT_CUBE, loc, false, CUBE_LIFETIME_MS)
        end
        s:start_quest_timer(EVT_REMOVE_PLAYERS, CUBE_LIFETIME_MS, nil, nil)
      end
    end
  end
end)

s:on_spawn(function(npc)
  npc:set_can_return_to_spawn_point(false)
  npc:set_random_walking(false)
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if not zones.is_inside(LAIR, attacker) then
    attacker:die(attacker)
    return
  end
  if npc:is_invul() then
    return
  end
  if grandboss.status(VALAKAS) ~= FIGHTING then
    attacker:teleport(EJECT)
    return
  end
  if attacker:mount_type() == MOUNT_STRIDER and not attacker:is_affected_by_skill(SKILL_ANTI_STRIDER) then
    npc:set_target(attacker)
    npc:cast(skills.get(SKILL_ANTI_STRIDER, 1))
  end
  last_attack = time.now_ms()
end)

s:on_kill(function(npc, killer, is_summon)
  s:cancel_quest_timer(EVT_REGEN_TICK, npc, nil)
  s:cancel_quest_timer(EVT_SKILL_TICK, npc, nil)
  cinematics.sound(LAIR, "B03_D")
  cinematics.special_camera(LAIR, npc, 1200, 20, -10, 0, 10000, 13000, 0, 0, 0, 0, 0)
  for i, step in ipairs(DIE_STEPS) do
    s:start_quest_timer("VALAKAS_DIE_" .. i, step[1], npc, nil)
  end
  local delay = boss.respawn_ms(Config.VALAKAS_SPAWN_INTERVAL, Config.VALAKAS_SPAWN_RANDOM)
  boss.died(s, VALAKAS, DEAD, delay, EVT_UNLOCK, "Valakas")
end)

s:on_spell_finished(function(npc, player, skill)
  s:start_quest_timer(EVT_SKILL_TICK, 1000, npc, nil)
  if not zones.is_inside(GROUND, npc) and valakas ~= nil and valakas:valid() then
    valakas:teleport(LAIR_SPOT)
  end
end)
