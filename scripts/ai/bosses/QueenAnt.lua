-- Queen Ant, ported from Mobius ai/bosses/QueenAnt (BazookaRpm).
--
-- The queen lives in her nest (boss zone 12012) with her larva, which
-- cannot move or die, her nurses, guards and royal guards. The nurses do
-- not fight: every second they heal the larva first, then the queen. Every
-- 10 s the queen may play a social action, and every 5 s she walks back
-- into the nest when she was dragged out. Players far above her level who
-- come near are cursed. Killed, she comes back after
-- Config.QUEEN_ANT_SPAWN_INTERVAL hours, give or take
-- QUEEN_ANT_SPAWN_RANDOM; the players in the nest are moved out when she
-- spawns.
--
-- Not ported: setIsRaidMinion(true) for the larva and the guards (the game
-- makes the queen's own minions raid minions; the larva is a plain spawn).
-- Mobius's isMuted and isParalyzed checks before the curse are the curse
-- skills' own effects here (Raid Curse 4215, 4515).

local boss = require("lib.boss")

local s = script.new("QueenAnt")

local QUEEN = 29001
local LARVA = 29002
local NURSE = 29003
local GUARD = 29004
local ROYAL = 29005
local IDS = { QUEEN, LARVA, NURSE, GUARD, ROYAL }

local ZONE = 12012
local NEST = { x = -21610, y = 181594, z = -5734 }
local LARVA_SPAWN = { x = -21600, y = 179482, z = -5846 }
local TELEPORTS = {
  { x = -19480, y = 187344, z = -5600 },
  { x = -17928, y = 180912, z = -5520 },
  { x = -23808, y = 182368, z = -5600 },
}

local HEAL = "heal"
local ACTION = "action"
local UNLOCK = "queen_unlock"
local DISTANCE_CHECK = "DISTANCE_CHECK"

local HEAL_MS = 1000
local ACTION_MS = 10000
local DISTANCE_CHECK_MS = 5000
local CURSE_LEVEL_DIFFERENCE = 8
local CURSE_CAST_MS = 300
local ROYAL_RESPAWN_S = 280
local ROYAL_RESPAWN_RANDOM_S = 40
local NURSE_RESPAWN_MS = 10000

local NURSE_HEAL_1 = 4020
local NURSE_HEAL_2 = 4024
local RAID_CURSE = 4215 -- mute, for mages
local RAID_CURSE2 = 4515 -- petrification, for the others

s:add_spawn_id(IDS)
s:add_kill_id(IDS)
s:add_aggro_range_enter_id(IDS)
s:add_faction_call_id(NURSE)

-- The live queen and larva, and the nurses by object ID. NPC handles last
-- across handlers; after a reload on_start finds them again.
local queen, larva
local nurses = {}

local function sorted_nurses()
  local ids = {}
  for id in pairs(nurses) do
    ids[#ids + 1] = id
  end
  table.sort(ids)
  return ids
end

-- spawn_boss moves the players out of the nest, starts the queen's timers
-- and spawns her larva (Mobius spawnBoss).
local function spawn_boss(npc)
  grandboss.add_boss(npc)
  if rnd(100) < 33 then
    zones.move_players(ZONE, TELEPORTS[1])
  elseif rnd(100) < 50 then
    zones.move_players(ZONE, TELEPORTS[2])
  else
    zones.move_players(ZONE, TELEPORTS[3])
  end
  s:start_quest_timer(ACTION, ACTION_MS, npc, nil, true)
  s:start_quest_timer(HEAL, HEAL_MS, nil, nil, true)
  boss.sound(npc, "BS01_A")
  queen = npc
  -- Mobius passes getRandom(360) as the heading.
  larva = s:add_spawn(LARVA, { x = LARVA_SPAWN.x, y = LARVA_SPAWN.y, z = LARVA_SPAWN.z, heading = rnd(360) }, false, 0)
end

local function unlock()
  local npc = s:add_spawn(QUEEN, { x = NEST.x, y = NEST.y, z = NEST.z, heading = 0 }, false, 0)
  grandboss.set_status(QUEEN, boss.ALIVE)
  spawn_boss(npc)
end

s:on_start(function(ev)
  boss.start(s, QUEEN, {
    unlock = UNLOCK,
    live = function(npc)
      -- A reload: the queen, larva and nurses are there, their timers not.
      queen = npc
      for _, n in ipairs(world.npcs_by_id(LARVA)) do
        if not n:is_dead() then
          larva = n
        end
      end
      for _, n in ipairs(world.npcs_by_id(NURSE)) do
        if not n:is_dead() then
          nurses[n:object_id()] = n
        end
      end
      s:start_quest_timer(ACTION, ACTION_MS, npc, nil, true)
      s:start_quest_timer(HEAL, HEAL_MS, nil, nil, true)
      s:start_quest_timer(DISTANCE_CHECK, DISTANCE_CHECK_MS, npc, nil, true)
    end,
    unlocked = unlock,
    restore = function(info)
      local loc = { x = info.x, y = info.y, z = info.z }
      if not zones.is_inside(ZONE, loc) then
        loc = NEST
      end
      spawn_boss(boss.spawn_saved(s, QUEEN, info, loc))
    end,
  })
end)

-- heal is the nurses' second: the larva first, then the queen (nurses of
-- the larva only heal the larva); idle nurses drop their target.
local function heal()
  local larva_hurt = boss.alive(larva) and larva:hp() < larva:max_hp()
  local queen_hurt = boss.alive(queen) and queen:hp() < queen:max_hp()
  for _, id in ipairs(sorted_nurses()) do
    local nurse = nurses[id]
    if not nurse:valid() then
      nurses[id] = nil
    elseif not nurse:is_dead() and not nurse:is_casting_now() then
      local not_casting = nurse:intention() ~= Intention.CAST
      if larva_hurt then
        if not boss.same(nurse:target(), larva) or not_casting then
          nurse:set_target(larva)
          local skill = rnd.bool() and skills.get(NURSE_HEAL_1, 1) or skills.get(NURSE_HEAL_2, 1)
          nurse:use_magic(skill)
        end
      elseif queen_hurt then
        if not boss.same(nurse:leader(), larva) and (not boss.same(nurse:target(), queen) or not_casting) then
          nurse:set_target(queen)
          nurse:use_magic(skills.get(NURSE_HEAL_1, 1))
        end
      elseif not_casting and nurse:target() ~= nil then
        nurse:set_target(nil)
      end
    end
  end
end

s:on_event(function(event, npc, player)
  if event == HEAL then
    heal()
  elseif event == ACTION then
    if npc ~= nil and rnd(3) == 0 then
      if rnd(2) == 0 then
        npc:social_action(3)
      else
        npc:social_action(4)
      end
    end
  elseif event == UNLOCK then
    unlock()
  elseif event == DISTANCE_CHECK then
    if not boss.alive(queen) then
      s:cancel_quest_timers(DISTANCE_CHECK)
    elseif not zones.is_inside(ZONE, queen) then
      queen:clear_aggro_list()
      queen:move_to(NEST)
    end
  end
end)

s:on_spawn(function(npc)
  local id = npc:id()
  if id == LARVA then
    npc:set_immobilized(true)
    npc:set_mortal(false)
  elseif id == NURSE then
    npc:disable_core_ai(true)
    nurses[npc:object_id()] = npc
  elseif id == QUEEN then
    s:cancel_quest_timer(DISTANCE_CHECK, npc, nil)
    s:start_quest_timer(DISTANCE_CHECK, DISTANCE_CHECK_MS, npc, nil, true)
  end
end)

-- A nurse called by a hurt ant heals it.
s:on_faction_call(function(npc, caller, attacker, is_summon)
  if caller == nil or npc == nil then
    return
  end
  if not npc:is_casting_now() and npc:intention() ~= Intention.CAST and caller:hp() < caller:max_hp() then
    npc:set_target(caller)
    npc:use_magic(skills.get(NURSE_HEAL_1, 1))
  end
end)

-- The raid curse on players (or their summons) more than 8 levels above
-- the ant that sees them: a 1 in 4 chance, and the ant stops hating them.
s:on_aggro_range_enter(function(npc, player, is_summon)
  if player:is_gm() and player:is_invisible() then
    return
  end
  local attacker, mage
  if is_summon then
    attacker, mage = player:summon(), false
  else
    attacker, mage = player, player:is_mage_class()
  end
  if attacker == nil then
    return
  end
  if not Config.RAID_DISABLE_CURSE and attacker:level() - npc:level() > CURSE_LEVEL_DIFFERENCE then
    local curse
    if mage then
      if not attacker:is_affected_by_skill(RAID_CURSE) and rnd(4) == 0 then
        curse = skills.get(RAID_CURSE, 1)
      end
    elseif not attacker:is_affected_by_skill(RAID_CURSE2) and rnd(4) == 0 then
      curse = skills.get(RAID_CURSE2, 1)
    end
    if curse ~= nil then
      npc:magic_skill_use(attacker, curse:id(), curse:level(), CURSE_CAST_MS, 0)
      skills.apply_effects(curse, npc, attacker)
    end
    npc:stop_hating(attacker)
  end
end)

s:on_kill(function(npc, killer, is_summon)
  local id = npc:id()
  if id == QUEEN then
    boss.sound(npc, "BS02_D")
    local delay = boss.respawn_ms(Config.QUEEN_ANT_SPAWN_INTERVAL, Config.QUEEN_ANT_SPAWN_RANDOM)
    s:cancel_quest_timer(ACTION, npc, nil)
    s:cancel_quest_timer(HEAL, nil, nil)
    boss.died(s, QUEEN, boss.DEAD, delay, UNLOCK, "Queen Ant")
    nurses = {}
    if larva ~= nil and larva:valid() then
      larva:delete()
    end
    larva, queen = nil, nil
    s:cancel_quest_timers(DISTANCE_CHECK)
  elseif boss.alive(queen) then
    if id == ROYAL then
      if npc:leader() ~= nil then
        npc:minion_respawn((ROYAL_RESPAWN_S + rnd(ROYAL_RESPAWN_RANDOM_S)) * 1000)
      end
    elseif id == NURSE then
      nurses[npc:object_id()] = nil
      if npc:leader() ~= nil then
        npc:minion_respawn(NURSE_RESPAWN_MS)
      end
    end
  end
end)
