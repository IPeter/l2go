-- Orfen, ported from Mobius ai/bosses/Orfen (BazookaRpm).
--
-- Orfen spawns at one of three places in the Sea of Spores with four Raikel
-- Leos. Below half HP she flees to her nest (the first place) and stays
-- there until she healed above 95 %; outside her zone (12013) she goes back
-- to a place. Every 10 s her Raikel Leos that are 3,000 away are pulled
-- back to her. She pulls attackers from 300–1,000 away to her and
-- paralyzes them, and does the same to players buffing or healing near
-- her. Riba Iren heal Orfen and each other, Raikel Leos strike the
-- attackers of their group. Killed, she comes back after
-- Config.ORFEN_SPAWN_INTERVAL hours, give or take ORFEN_SPAWN_RANDOM; her
-- Raikel Leos vanish 20 s after her.
--
-- Not ported: Mobius's kill handler for Raikel Leos and its "spawn_minion"
-- event never run (only Orfen's kill is registered), and
-- setIsRaidMinion(true) (the Raikel Leos are plain spawns). Whether she
-- fled is kept in her NPC variable "teleported" so a reload keeps it.

local boss = require("lib.boss")

local s = script.new("Orfen")

local ORFEN = 29014
local RAIKEL_LEOS = 29016
local RIBA_IREN = 29018

local ZONE = 12013

local UNLOCK = "orfen_unlock"
local CHECK_POSITION = "check_orfen_pos"
local CHECK_MINIONS = "check_minion_loc"
local DESPAWN_MINIONS = "despawn_minions"

local CHECK_POSITION_MS = 10000
local CHECK_MINIONS_MS = 10000
local MINION_DESPAWN_MS = 20000

local MINION_FOLLOW_RADIUS = 3000
local TELEPORT_MAX_RANGE = 1000
local TELEPORT_MIN_RANGE = 300

local PARALYSIS = { 4064, 1 }
local BLOW = { 4067, 4 }
local ORFEN_HEAL = { 4516, 1 }

-- The nest (1) and the three places she roams (2–4).
local SPAWNS = {
  { x = 43728, y = 17220, z = -4342 },
  { x = 55024, y = 17368, z = -5412 },
  { x = 53504, y = 21248, z = -5486 },
  { x = 53248, y = 24576, z = -5262 },
}

local MESSAGES = {
  "$s1. Stop kidding yourself about your own powerlessness!",
  "$s1. I'll make you feel what true fear is!",
  "You're really stupid to have challenged me. $s1! Get ready!",
  "$s1. Do you think that's going to work?!",
}

s:add_skill_see_id(ORFEN)
s:add_faction_call_id(ORFEN, RAIKEL_LEOS, RIBA_IREN)
s:add_attack_id(ORFEN, RAIKEL_LEOS, RIBA_IREN)
s:add_kill_id(ORFEN)

-- Orfen's Raikel Leos by object ID.
local minions = {}

local function skill(holder)
  return skills.get(holder[1], holder[2])
end

local function has_teleported(npc)
  return npc:var_bool("teleported", false)
end

local function set_teleported(npc, value)
  npc:set_var("teleported", value)
end

-- set_spawn_point moves Orfen to a place, which becomes her spawn point
-- (Mobius setSpawnPoint).
local function set_spawn_point(npc, index)
  npc:clear_aggro_list()
  npc:set_intention(Intention.IDLE)
  npc:set_spawn_location(SPAWNS[index])
  npc:teleport(SPAWNS[index])
end

-- random_place is one of the three roaming places: 40 %, 30 %, 30 %.
local function random_place()
  local roll = rnd(10)
  if roll < 4 then
    return SPAWNS[2]
  elseif roll < 7 then
    return SPAWNS[3]
  end
  return SPAWNS[4]
end

-- spawn_boss registers Orfen, starts her checks and spawns her Raikel
-- Leos (Mobius spawnBoss).
local function spawn_boss(npc)
  grandboss.add_boss(npc)
  boss.sound(npc, "BS01_A")
  s:start_quest_timer(CHECK_POSITION, CHECK_POSITION_MS, npc, nil, true)
  local x, y, z = npc:x(), npc:y(), npc:z()
  for _, d in ipairs({ { 100, 100 }, { 100, -100 }, { -100, 100 }, { -100, -100 } }) do
    local m = s:add_spawn(RAIKEL_LEOS, { x = x + d[1], y = y + d[2], z = z, heading = 0 }, false, 0)
    if m ~= nil then
      minions[m:object_id()] = m
    end
  end
  s:start_quest_timer(CHECK_MINIONS, CHECK_MINIONS_MS, npc, nil, true)
end

local function unlock()
  local npc = s:add_spawn(ORFEN, random_place(), false, 0)
  grandboss.set_status(ORFEN, boss.ALIVE)
  spawn_boss(npc)
end

s:on_start(function(ev)
  boss.start(s, ORFEN, {
    unlock = UNLOCK,
    live = function(npc)
      for _, m in ipairs(world.npcs_by_id(RAIKEL_LEOS)) do
        minions[m:object_id()] = m
      end
      s:start_quest_timer(CHECK_POSITION, CHECK_POSITION_MS, npc, nil, true)
      s:start_quest_timer(CHECK_MINIONS, CHECK_MINIONS_MS, npc, nil, true)
    end,
    unlocked = unlock,
    restore = function(info)
      spawn_boss(boss.spawn_saved(s, ORFEN, info))
    end,
  })
end)

local function sorted_minions()
  local ids = {}
  for id in pairs(minions) do
    ids[#ids + 1] = id
  end
  table.sort(ids)
  return ids
end

s:on_event(function(event, npc, player)
  if event == UNLOCK then
    unlock()
  elseif event == CHECK_POSITION then
    local teleported = has_teleported(npc)
    local inside = zones.is_inside(ZONE, npc)
    if (teleported and npc:hp() > npc:max_hp() * 0.95) or (not inside and not teleported) then
      set_spawn_point(npc, rnd(3) + 2)
      set_teleported(npc, false)
    elseif teleported and not inside then
      set_spawn_point(npc, 1)
    end
  elseif event == CHECK_MINIONS then
    for _, id in ipairs(sorted_minions()) do
      local m = minions[id]
      if not m:valid() then
        minions[id] = nil
      elseif not npc:is_inside_radius_2d(m, MINION_FOLLOW_RADIUS) then
        m:teleport(npc:x(), npc:y(), npc:z())
        m:clear_aggro_list()
      end
    end
  elseif event == DESPAWN_MINIONS then
    for _, m in pairs(minions) do
      if m:valid() then
        m:delete()
      end
    end
    minions = {}
  end
end)

-- pull says a message to target, brings it to Orfen and paralyzes it.
local function pull(npc, target, name, count)
  local text = MESSAGES[rnd(count) + 1]:gsub("%$s1", name)
  npc:broadcast_say(ChatType.NPC_GENERAL, text)
  target:teleport(npc:x(), npc:y(), npc:z())
  npc:set_target(target)
  npc:cast(skill(PARALYSIS))
end

-- Helpful skills (buffs, heals) seen near Orfen.
s:on_skill_see(function(npc, caster, sk, targets, is_summon)
  if npc:is_dead() then
    return
  end
  local original = caster
  if is_summon then
    original = caster:summon()
  end
  if sk:effect_point() > 0 and rnd(5) == 0 and npc:is_inside_radius_2d(original, TELEPORT_MAX_RANGE) then
    pull(npc, original, caster:name(), 4)
  end
end)

s:on_faction_call(function(npc, caller, attacker, is_summon)
  if caller == nil or npc == nil or npc:is_casting_now() then
    return
  end
  local id, caller_id = npc:id(), caller:id()
  if id == RAIKEL_LEOS and rnd(20) == 0 then
    npc:set_target(attacker)
    npc:cast(skill(BLOW))
  elseif id == RIBA_IREN then
    local chance = 1
    if caller_id == ORFEN then
      chance = 9
    end
    if caller_id ~= RIBA_IREN and caller:hp() < caller:max_hp() / 2.0 and rnd(10) < chance then
      npc:set_intention(Intention.IDLE)
      npc:set_target(caller)
      npc:cast(skill(ORFEN_HEAL))
    end
  end
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if npc:is_dead() then
    return
  end
  local id = npc:id()
  if id == ORFEN then
    if not has_teleported(npc) and npc:hp() - damage < npc:max_hp() * 0.5 then
      set_teleported(npc, true)
      set_spawn_point(npc, 1)
    elseif npc:is_inside_radius_2d(attacker, TELEPORT_MAX_RANGE) and not npc:is_inside_radius_2d(attacker, TELEPORT_MIN_RANGE)
      and rnd(10) == 0 then
      pull(npc, attacker, attacker:name(), 3)
    end
  elseif id == RIBA_IREN and not npc:is_casting_now() and npc:hp() - damage < npc:max_hp() * 0.5 then
    -- Mobius targets the attacker with the heal.
    npc:set_target(attacker)
    npc:cast(skill(ORFEN_HEAL))
  end
end)

s:on_kill(function(npc, killer, is_summon)
  if npc:id() ~= ORFEN then
    return
  end
  boss.sound(npc, "BS02_D")
  local delay = boss.respawn_ms(Config.ORFEN_SPAWN_INTERVAL, Config.ORFEN_SPAWN_RANDOM)
  boss.died(s, ORFEN, boss.DEAD, delay, UNLOCK, "Orfen")
  s:cancel_quest_timer(CHECK_MINIONS, npc, nil)
  s:cancel_quest_timer(CHECK_POSITION, npc, nil)
  s:start_quest_timer(DESPAWN_MINIONS, MINION_DESPAWN_MS, nil, nil)
end)
