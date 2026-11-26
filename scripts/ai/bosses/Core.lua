-- Core, ported from Mobius ai/bosses/Core (BazookaRpm).
--
-- Core sits immobile on the lowest floor of Cruma Tower among 19 guards
-- spawned at fixed spots. A dead guard comes back where it died after a
-- minute while Core lives. The first hit makes Core announce the
-- intruders (remembered across restarts in the global variable
-- Core_Attacked). Killed, Core shuts down, two teleport cubes appear for
-- 15 minutes, his guards vanish 20 s later, and he comes back after
-- Config.CORE_SPAWN_INTERVAL hours, give or take CORE_SPAWN_RANDOM.
--
-- Deviations: Mobius saves Core_Attacked when the server saves; here it is
-- saved when it changes. The teleport cube (899099) is a Mobius custom NPC
-- that the NPC data does not have yet, and its page links to this script's
-- TELEPORT event instead of the cube's custom teleport list.
-- setIsRaidMinion(true) is not ported (the guards are plain spawns).

local boss = require("lib.boss")

local s = script.new("Core")

local CORE = 29006
local DEATH_KNIGHT = 29007
local DOOM_WRAITH = 29008
local SUSCEPTOR = 29011
local TELEPORT_CUBE = 899099

local UNLOCK = "core_unlock"
local SPAWN_MINION_LEGACY = "spawn_minion"
local DESPAWN_MINIONS = "despawn_minions"
local TELEPORT = "TELEPORT"
local SPAWN_MINION_PREFIX = "spawn_minion_"

local MINION_RESPAWN_MS = 60000
local MINION_DESPAWN_MS = 20000
local CUBE_DESPAWN_MS = 900000

local CRUMA_ENTRANCE = { x = 17253, y = 114232, z = -3440 }
local CRUMA_FLOOR3 = { x = 17719, y = 115590, z = -6584 }
local CORE_SPAWN = { x = 17726, y = 108915, z = -6480 }

-- The guards by template, in the order Mobius's HashMap gives them
-- (29008, 29011, 29007).
local MINION_SPAWNS = {
  { DOOM_WRAITH, {
    { x = 17113, y = 110970, z = -6648 }, { x = 17496, y = 110880, z = -6648 }, { x = 18061, y = 110990, z = -6648 },
    { x = 18384, y = 110698, z = -6648 }, { x = 17993, y = 111458, z = -6584 },
  } },
  { SUSCEPTOR, {
    { x = 17297, y = 111470, z = -6584 }, { x = 17893, y = 110198, z = -6648 }, { x = 17706, y = 109423, z = -6488 },
    { x = 17849, y = 109388, z = -6480 },
  } },
  { DEATH_KNIGHT, {
    { x = 17191, y = 109298, z = -6488 }, { x = 17564, y = 109548, z = -6488 }, { x = 17855, y = 109552, z = -6488 },
    { x = 18280, y = 109202, z = -6488 }, { x = 18784, y = 109253, z = -6488 }, { x = 18059, y = 108314, z = -6488 },
    { x = 17300, y = 108444, z = -6488 }, { x = 17148, y = 110071, z = -6648 }, { x = 18318, y = 110077, z = -6648 },
    { x = 17726, y = 110391, z = -6648 },
  } },
}

s:add_kill_id(CORE, DEATH_KNIGHT, DOOM_WRAITH, SUSCEPTOR)
s:add_attack_id(CORE)
s:add_spawn_id(CORE)
s:add_first_talk_id(TELEPORT_CUBE)

-- The guards by object ID, the guards waiting to come back by respawn ID,
-- and whether Core was attacked since he spawned.
local minions = {}
local pending = {}
local next_respawn = 0
local first_attacked = false

local function set_attacked(value)
  first_attacked = value
  globals.set("Core_Attacked", value)
end

local function spawn_minion(id, loc)
  local m = s:add_spawn(id, loc, false, 0)
  if m ~= nil then
    minions[m:object_id()] = m
  end
end

-- spawn_boss registers Core and spawns his guards (Mobius spawnBoss).
local function spawn_boss(npc)
  if npc == nil then
    return
  end
  grandboss.add_boss(npc)
  boss.sound(npc, "BS01_A")
  for _, group in ipairs(MINION_SPAWNS) do
    for _, loc in ipairs(group[2]) do
      spawn_minion(group[1], loc)
    end
  end
  pending = {}
  next_respawn = 0
end

local function unlock()
  local npc = s:add_spawn(CORE, CORE_SPAWN, false, 0)
  grandboss.set_status(CORE, boss.ALIVE)
  spawn_boss(npc)
end

s:on_start(function(ev)
  boss.start(s, CORE, {
    unlock = UNLOCK,
    live = function()
      -- A reload: Core and his guards are there.
      first_attacked = globals.get_bool("Core_Attacked", false)
      for _, group in ipairs(MINION_SPAWNS) do
        for _, n in ipairs(world.npcs_by_id(group[1])) do
          if not n:is_dead() then
            minions[n:object_id()] = n
          end
        end
      end
    end,
    unlocked = unlock,
    restore = function(info)
      first_attacked = globals.get_bool("Core_Attacked", false)
      spawn_boss(boss.spawn_saved(s, CORE, info))
    end,
  })
end)

s:on_event(function(event, npc, player)
  if event:sub(1, #SPAWN_MINION_PREFIX) == SPAWN_MINION_PREFIX then
    local id = tonumber(event:sub(#SPAWN_MINION_PREFIX + 1))
    if id == nil then
      log.warn("Core: a minion respawn event without an ID", "event", event)
      return
    end
    local data = pending[id]
    pending[id] = nil
    if data ~= nil and grandboss.status(CORE) == boss.ALIVE then
      spawn_minion(data.id, data.loc)
    end
  elseif event == UNLOCK then
    unlock()
  elseif event == SPAWN_MINION_LEGACY then
    if npc ~= nil then
      spawn_minion(npc:id(), { x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading() })
    end
  elseif event == DESPAWN_MINIONS then
    for _, m in pairs(minions) do
      if m:valid() then
        m:delete()
      end
    end
    minions = {}
    pending = {}
    next_respawn = 0
  elseif event:sub(1, #TELEPORT) == TELEPORT then
    if player ~= nil then
      local destination = event:match("^%S+%s+(%S+)")
      if destination == "floor3" then
        player:teleport(CRUMA_FLOOR3)
      elseif destination == "ground" then
        player:teleport(CRUMA_ENTRANCE)
      else
        log.warn("Core: unknown teleport destination", "event", event)
      end
    end
  end
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if first_attacked then
    if rnd(100) == 0 then
      npc:broadcast_say(ChatType.NPC_GENERAL, "Removing intruders.")
    end
  else
    set_attacked(true)
    npc:broadcast_say(ChatType.NPC_GENERAL, "A non-permitted target has been discovered.")
    npc:broadcast_say(ChatType.NPC_GENERAL, "Intruder removal system initiated.")
  end
end)

s:on_kill(function(npc, killer, is_summon)
  local id = npc:id()
  if id == CORE then
    boss.sound(npc, "BS02_D")
    npc:broadcast_say(ChatType.NPC_GENERAL, "A fatal error has occurred.")
    npc:broadcast_say(ChatType.NPC_GENERAL, "System is being shut down...")
    npc:broadcast_say(ChatType.NPC_GENERAL, "......")
    set_attacked(false)
    s:add_spawn(TELEPORT_CUBE, { x = 16502, y = 110165, z = -6394, heading = 0 }, false, CUBE_DESPAWN_MS)
    s:add_spawn(TELEPORT_CUBE, { x = 18948, y = 110166, z = -6397, heading = 0 }, false, CUBE_DESPAWN_MS)
    local delay = boss.respawn_ms(Config.CORE_SPAWN_INTERVAL, Config.CORE_SPAWN_RANDOM)
    boss.died(s, CORE, boss.DEAD, delay, UNLOCK, "Core")
    s:start_quest_timer(DESPAWN_MINIONS, MINION_DESPAWN_MS, nil, nil)
    for i = 0, next_respawn - 1 do
      s:cancel_quest_timer(SPAWN_MINION_PREFIX .. i, nil, nil)
    end
    pending = {}
    next_respawn = 0
    s:cancel_quest_timers(SPAWN_MINION_LEGACY)
  elseif grandboss.status(CORE) == boss.ALIVE and minions[npc:object_id()] ~= nil then
    minions[npc:object_id()] = nil
    local respawn = next_respawn
    next_respawn = next_respawn + 1
    pending[respawn] = { id = id, loc = { x = npc:x(), y = npc:y(), z = npc:z() } }
    s:start_quest_timer(SPAWN_MINION_PREFIX .. respawn, MINION_RESPAWN_MS, nil, nil)
  end
end)

s:on_spawn(function(npc)
  npc:set_immobilized(true)
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)
