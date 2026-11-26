-- Zaken, ported from Mobius ai/bosses/Zaken/Zaken (BazookaRpm).
--
-- Zaken lives on his ship (boss zone 12016). Every 30 s he checks the time
-- of day: at night (game hour < 5) he wears his night face (4224) and
-- regenerates (4227), teleports away (4222) when everyone he fights is far
-- from his teleport point, and drops a target he chased for over five
-- checks; by day he wears his day face (4223) and stops regenerating. Now
-- and then he teleports to one of 15 places of the ship. Hit or
-- approached, he casts his teleport skills (4216 one player, 4217 the
-- players around) and his attacks (4218–4221); by day he teleports away
-- at each quarter of his HP lost. Spawned inside his zone, he calls seven
-- waves of pirates, 1.7 s apart; a dead pirate comes back after 30–89 s.
-- Killed, he comes back after Config.ZAKEN_SPAWN_INTERVAL hours, give or
-- take ZAKEN_SPAWN_RANDOM.
--
-- A mounted attacker gets his mount debuff (Hinder Strider, 4258) unless
-- it has it. The players his area teleport remembers are kept by object
-- ID.

local boss = require("lib.boss")

local s = script.new("Zaken")

local ZAKEN = 29022
local DOLL_BLADER = 29023
local VALE_MASTER = 29024
local ZOMBIE_CAPTAIN = 29026
local ZOMBIE = 29027

local ZONE = 12016

local UNLOCK = "zaken_unlock"
local TIMER_MAIN = "1001"
local TIMER_FACTION_TELEPORT = "1002"
local TIMER_MINION_WAVES = "1003"
local MINION_RESPAWN_PREFIX = "RESPAWN_MINION:"

local SKILL_TELEPORT_SELF = 4222
local SKILL_FACE_DAY = 4223
local SKILL_REGEN_NIGHT = 4227
local SKILL_TELEPORT_SINGLE = 4216
local SKILL_TELEPORT_AREA = 4217
local SKILL_4218 = 4218
local SKILL_4219 = 4219
local SKILL_4220 = 4220
local SKILL_4221 = 4221
local SKILL_REGEN_CANCEL = 4242
local SKILL_FACE_NIGHT = 4224
local SKILL_MOUNT_DEBUFF = 4258

local NIGHT_END_HOUR = 5
local TELEPORT_RANGE = 1500
local AREA_TELEPORT_RANGE = 250
local Z_TOLERANCE = 100
local FACTION_CALL_Z_TOLERANCE = 200
local SKILL_RANGE_CHECK = 100
local TELEPORT_OFFSET_MAX = 650
local TRACK_LIMIT = 5
local MAX_HEADING = 65536
local SKILL_ROLL_MAX = 15 * 15

local TELEPORTS = {
  { x = 53950, y = 219860, z = -3488 }, { x = 55980, y = 219820, z = -3488 }, { x = 54950, y = 218790, z = -3488 },
  { x = 55970, y = 217770, z = -3488 }, { x = 53930, y = 217760, z = -3488 }, { x = 55970, y = 217770, z = -3216 },
  { x = 55980, y = 219920, z = -3216 }, { x = 54960, y = 218790, z = -3216 }, { x = 53950, y = 219860, z = -3216 },
  { x = 53930, y = 217760, z = -3216 }, { x = 55970, y = 217770, z = -2944 }, { x = 55980, y = 219920, z = -2944 },
  { x = 54960, y = 218790, z = -2944 }, { x = 53950, y = 219860, z = -2944 }, { x = 53930, y = 217760, z = -2944 },
}

-- The fixed pirate waves 5 to 7: {template, x, y, z}.
local WAVE5 = {
  { DOLL_BLADER, 52675, 219371, -3290 }, { DOLL_BLADER, 52687, 219596, -3368 }, { DOLL_BLADER, 52672, 219740, -3418 },
  { ZOMBIE, 52857, 219992, -3488 }, { ZOMBIE_CAPTAIN, 52959, 219997, -3488 }, { VALE_MASTER, 53381, 220151, -3488 },
  { ZOMBIE_CAPTAIN, 54236, 220948, -3488 }, { ZOMBIE, 54885, 220144, -3488 }, { ZOMBIE, 55264, 219860, -3488 },
  { ZOMBIE_CAPTAIN, 55399, 220263, -3488 }, { ZOMBIE, 55679, 220129, -3488 }, { VALE_MASTER, 56276, 220783, -3488 },
  { VALE_MASTER, 57173, 220234, -3488 }, { ZOMBIE, 56267, 218826, -3488 }, { DOLL_BLADER, 56294, 219482, -3488 },
  { ZOMBIE_CAPTAIN, 56094, 219113, -3488 }, { DOLL_BLADER, 56364, 218967, -3488 }, { ZOMBIE, 57113, 218079, -3488 },
  { DOLL_BLADER, 56186, 217153, -3488 }, { ZOMBIE, 55440, 218081, -3488 }, { ZOMBIE_CAPTAIN, 55202, 217940, -3488 },
  { ZOMBIE, 55225, 218236, -3488 }, { ZOMBIE, 54973, 218075, -3488 }, { ZOMBIE_CAPTAIN, 53412, 218077, -3488 },
  { VALE_MASTER, 54226, 218797, -3488 }, { VALE_MASTER, 54394, 219067, -3488 }, { ZOMBIE, 54139, 219253, -3488 },
  { DOLL_BLADER, 54262, 219480, -3488 },
}
local WAVE6 = {
  { ZOMBIE, 53412, 218077, -3488 }, { VALE_MASTER, 54413, 217132, -3488 }, { DOLL_BLADER, 54841, 217132, -3488 },
  { DOLL_BLADER, 55372, 217128, -3343 }, { DOLL_BLADER, 55893, 217122, -3488 }, { ZOMBIE_CAPTAIN, 56282, 217237, -3216 },
  { VALE_MASTER, 56963, 218080, -3216 }, { ZOMBIE, 56267, 218826, -3216 }, { DOLL_BLADER, 56294, 219482, -3216 },
  { ZOMBIE_CAPTAIN, 56094, 219113, -3216 }, { DOLL_BLADER, 56364, 218967, -3216 }, { VALE_MASTER, 56276, 220783, -3216 },
  { VALE_MASTER, 57173, 220234, -3216 }, { ZOMBIE, 54885, 220144, -3216 }, { ZOMBIE, 55264, 219860, -3216 },
  { ZOMBIE_CAPTAIN, 55399, 220263, -3216 }, { ZOMBIE, 55679, 220129, -3216 }, { ZOMBIE_CAPTAIN, 54236, 220948, -3216 },
  { ZOMBIE_CAPTAIN, 54464, 219095, -3216 }, { VALE_MASTER, 54226, 218797, -3216 }, { VALE_MASTER, 54394, 219067, -3216 },
  { ZOMBIE, 54139, 219253, -3216 }, { DOLL_BLADER, 54262, 219480, -3216 }, { ZOMBIE_CAPTAIN, 53412, 218077, -3216 },
  { ZOMBIE, 55440, 218081, -3216 }, { ZOMBIE_CAPTAIN, 55202, 217940, -3216 }, { ZOMBIE, 55225, 218236, -3216 },
  { ZOMBIE, 54973, 218075, -3216 },
}
local WAVE7 = {
  { ZOMBIE, 54228, 217504, -3216 }, { VALE_MASTER, 54181, 217168, -3216 }, { DOLL_BLADER, 54714, 217123, -3168 },
  { DOLL_BLADER, 55298, 217127, -3073 }, { DOLL_BLADER, 55787, 217130, -2993 }, { ZOMBIE_CAPTAIN, 56284, 217216, -2944 },
  { VALE_MASTER, 56963, 218080, -2944 }, { ZOMBIE, 56267, 218826, -2944 }, { DOLL_BLADER, 56294, 219482, -2944 },
  { ZOMBIE_CAPTAIN, 56094, 219113, -2944 }, { DOLL_BLADER, 56364, 218967, -2944 }, { VALE_MASTER, 56276, 220783, -2944 },
  { VALE_MASTER, 57173, 220234, -2944 }, { ZOMBIE, 54885, 220144, -2944 }, { ZOMBIE, 55264, 219860, -2944 },
  { ZOMBIE_CAPTAIN, 55399, 220263, -2944 }, { ZOMBIE, 55679, 220129, -2944 }, { ZOMBIE_CAPTAIN, 54236, 220948, -2944 },
  { ZOMBIE_CAPTAIN, 54464, 219095, -2944 }, { VALE_MASTER, 54226, 218797, -2944 }, { VALE_MASTER, 54394, 219067, -2944 },
  { ZOMBIE, 54139, 219253, -2944 }, { DOLL_BLADER, 54262, 219480, -2944 }, { ZOMBIE_CAPTAIN, 53412, 218077, -2944 },
  { ZOMBIE_CAPTAIN, 54280, 217200, -2944 }, { ZOMBIE, 55440, 218081, -2944 }, { ZOMBIE_CAPTAIN, 55202, 217940, -2944 },
  { ZOMBIE, 55225, 218236, -2944 }, { ZOMBIE, 54973, 218075, -2944 },
}

s:add_attack_id(ZAKEN)
s:add_kill_id(ZAKEN, DOLL_BLADER, VALE_MASTER, ZOMBIE_CAPTAIN, ZOMBIE)
s:add_spell_finished_id(ZAKEN)
s:add_aggro_range_enter_id(ZAKEN)
s:add_faction_call_id(ZAKEN)

-- The AI state (Mobius's fields). tracked holds the object IDs of up to
-- five players the area teleport takes along.
local zaken
local pending = {}
local next_respawn = 0
local teleport_in_progress = 0
local tp = { x = 0, y = 0, z = 0 }
local wave = 0
local tracked = {}
local tracked_count = 0
local hated_ticks = 0
local hated_target
local day_hp_stage = 3

local function hour()
  return time.game_hour() % 24
end

local function cast_self(npc, id)
  npc:set_target(npc)
  npc:cast(skills.get(id, 1))
end

local function cast_at(npc, target, id)
  npc:set_target(target)
  npc:cast(skills.get(id, 1))
end

-- random_spot is one of the 15 places with a random offset.
local function random_spot()
  local loc = TELEPORTS[rnd(#TELEPORTS) + 1]
  local x = loc.x + rnd(TELEPORT_OFFSET_MAX)
  local y = loc.y + rnd(TELEPORT_OFFSET_MAX)
  return { x = x, y = y, z = loc.z }
end

local function set_tp(loc)
  tp.x, tp.y, tp.z = loc.x, loc.y, loc.z
end

local function dist2(ax, ay, bx, by)
  local dx, dy = ax - bx, ay - by
  return dx * dx + dy * dy
end

local function spawn_pirate(id, x, y, z)
  return s:add_spawn(id, { x = x, y = y, z = z, heading = rnd(MAX_HEADING) }, false, 0)
end

local function spawn_random_pirate(id)
  local spot = random_spot()
  spawn_pirate(id, spot.x, spot.y, spot.z)
end

-- spawn_boss registers Zaken and starts his timers (Mobius spawnBoss).
local function spawn_boss(npc)
  if npc == nil then
    log.warn("Zaken AI failed to load: no Zaken spawned")
    return
  end
  grandboss.add_boss(npc)
  zaken = npc
  pending = {}
  boss.sound(npc, "BS01_A")
  teleport_in_progress = 0
  set_tp({ x = npc:x(), y = npc:y(), z = npc:z() })
  wave = 0
  tracked_count = 0
  hated_ticks = 0
  day_hp_stage = 3
  hated_target = nil
  tracked = {}
  if zones.is_inside(ZONE, npc) then
    wave = 1
    s:start_quest_timer(TIMER_MINION_WAVES, 1700, npc, nil, true)
  end
  s:start_quest_timer(TIMER_MAIN, 1000, npc, nil)
end

local function unlock()
  local npc = s:add_spawn(ZAKEN, { x = 55312, y = 219168, z = -3223, heading = 0 }, false, 0)
  grandboss.set_status(ZAKEN, boss.ALIVE)
  spawn_boss(npc)
end

s:on_start(function(ev)
  boss.start(s, ZAKEN, {
    unlock = UNLOCK,
    live = function(npc)
      zaken = npc
      set_tp({ x = npc:x(), y = npc:y(), z = npc:z() })
      s:start_quest_timer(TIMER_MAIN, 1000, npc, nil)
    end,
    unlocked = unlock,
    restore = function(info)
      info.hp, info.mp = math.floor(info.hp), math.floor(info.mp)
      spawn_boss(boss.spawn_saved(s, ZAKEN, info))
    end,
  })
end)

-- out_of_range reports whether a tracked player (by object ID) is more
-- than 1,500 from the teleport point; a missing one counts as in range.
local function out_of_range(id)
  local p = id and world.get_player(id)
  if p == nil then
    return false
  end
  return dist2(p:x(), p:y(), tp.x, tp.y) > TELEPORT_RANGE * TELEPORT_RANGE
end

-- main is Zaken's 30 s check.
local function main(npc)
  local day_face = npc:is_affected_by_skill(SKILL_FACE_DAY)
  local night_regen = npc:is_affected_by_skill(SKILL_REGEN_NIGHT)
  if hour() < NIGHT_END_HOUR then
    if day_face then
      cast_self(npc, SKILL_FACE_NIGHT)
      set_tp({ x = npc:x(), y = npc:y(), z = npc:z() })
    end
    if not night_regen then
      cast_self(npc, SKILL_REGEN_NIGHT)
    end
    if npc:intention() == Intention.ATTACK and teleport_in_progress == 0 then
      local most = npc:most_hated()
      if most ~= nil then
        local all_out = dist2(most:x(), most:y(), tp.x, tp.y) > TELEPORT_RANGE * TELEPORT_RANGE
        for i = 1, math.min(tracked_count, TRACK_LIMIT) do
          if not out_of_range(tracked[i]) then
            all_out = false
          end
        end
        if all_out then
          tracked_count = 0
          set_tp(random_spot())
          cast_self(npc, SKILL_TELEPORT_SELF)
        end
      end
    end
    if rnd(20) < 1 and teleport_in_progress == 0 then
      set_tp({ x = npc:x(), y = npc:y(), z = npc:z() })
    end
    -- hated_target is an object ID: players expire with the handler.
    local most = npc:most_hated()
    local attacking = npc:intention() == Intention.ATTACK
    if attacking and hated_ticks == 0 and most ~= nil then
      hated_target, hated_ticks = most:object_id(), 1
    elseif attacking and hated_ticks ~= 0 and most ~= nil then
      if most:object_id() == hated_target then
        hated_ticks = hated_ticks + 1
      else
        hated_target, hated_ticks = most:object_id(), 1
      end
    end
    if npc:intention() == Intention.IDLE then
      hated_ticks, hated_target = 0, nil
    end
    if hated_ticks > 5 then
      local target = hated_target and world.find_object(hated_target)
      if target ~= nil then
        npc:stop_hating(target)
      end
      local next_target = npc:most_hated()
      if next_target ~= nil then
        npc:attack(next_target)
      end
      hated_ticks, hated_target = 0, nil
    end
  elseif not day_face then
    cast_self(npc, SKILL_FACE_DAY)
    day_hp_stage = 3
  end
  if hour() >= NIGHT_END_HOUR and night_regen then
    cast_self(npc, SKILL_REGEN_CANCEL)
  end
  if rnd(40) < 1 then
    set_tp(random_spot())
    cast_self(npc, SKILL_TELEPORT_SELF)
  end
  s:start_quest_timer(TIMER_MAIN, 30000, npc, nil)
end

-- next_wave spawns the next pirate wave.
local function next_wave()
  if wave == 1 then
    spawn_random_pirate(ZOMBIE_CAPTAIN)
  elseif wave == 2 then
    spawn_random_pirate(DOLL_BLADER)
  elseif wave == 3 then
    spawn_random_pirate(VALE_MASTER)
    spawn_random_pirate(VALE_MASTER)
  elseif wave == 4 then
    for _ = 1, 5 do
      spawn_random_pirate(ZOMBIE)
    end
  elseif wave >= 5 and wave <= 7 then
    for _, p in ipairs(({ WAVE5, WAVE6, WAVE7 })[wave - 4]) do
      spawn_pirate(p[1], p[2], p[3], p[4])
    end
  else
    return
  end
  wave = wave + 1
  if wave == 8 then
    s:cancel_quest_timer(TIMER_MINION_WAVES, nil, nil)
  end
end

s:on_event(function(event, npc, player)
  if event:sub(1, #MINION_RESPAWN_PREFIX) == MINION_RESPAWN_PREFIX then
    local id = tonumber(event:sub(#MINION_RESPAWN_PREFIX + 1))
    if id == nil then
      log.warn("Zaken minion respawn event parse failed", "event", event)
      return
    end
    local data = pending[id]
    pending[id] = nil
    if data ~= nil and grandboss.status(ZAKEN) == boss.ALIVE then
      s:add_spawn(data.id, { x = data.x, y = data.y, z = data.z, heading = data.heading }, false, 0)
    end
  elseif event == TIMER_MAIN then
    main(npc)
  elseif event == TIMER_FACTION_TELEPORT then
    tracked_count = 0
    npc:cast(skills.get(SKILL_TELEPORT_SELF, 1))
    teleport_in_progress = 0
  elseif event == TIMER_MINION_WAVES then
    next_wave()
  elseif event == UNLOCK then
    if grandboss.status(ZAKEN) == boss.DEAD then
      unlock()
    end
  end
end)

-- original returns the summon for a summon's action, else the player.
local function original(player, is_summon)
  if is_summon then
    return player:summon() or player
  end
  return player
end

local function in_hate_list(npc, target)
  for _, row in ipairs(npc:hate_list()) do
    if boss.same(row.target, target) then
      return true
    end
  end
  return false
end

-- At night, a pirate fighting near him may make Zaken teleport to it.
s:on_faction_call(function(npc, caller, attacker, is_summon)
  if caller == nil or npc == nil or attacker == nil then
    return
  end
  if hour() < NIGHT_END_HOUR and caller:id() ~= ZAKEN and npc:id() == ZAKEN then
    local target = original(attacker, is_summon)
    if not in_hate_list(caller, target) then
      return
    end
    local dz = math.abs(target:z() - caller:z())
    if dz <= FACTION_CALL_Z_TOLERANCE and dist2(target:x(), target:y(), caller:x(), caller:y()) <= TELEPORT_RANGE * TELEPORT_RANGE then
      if npc:intention() == Intention.IDLE and teleport_in_progress == 0 and rnd(30 * 15) < 1 then
        teleport_in_progress = 1
        set_tp({ x = caller:x(), y = caller:y(), z = caller:z() })
        s:start_quest_timer(TIMER_FACTION_TELEPORT, 300, npc, nil)
      end
    end
  end
end)

-- teleport_away sends a player to a random place and Zaken stops hating
-- them.
local function teleport_away(npc, p)
  local spot = random_spot()
  p:teleport(spot.x, spot.y, spot.z)
  npc:stop_hating(p)
end

local function attack_most_hated(npc)
  local next_target = npc:most_hated()
  if next_target ~= nil then
    npc:attack(next_target)
  end
end

s:on_spell_finished(function(npc, player, sk)
  local id = sk:id()
  if id == SKILL_TELEPORT_SELF then
    npc:teleport(tp.x, tp.y, tp.z)
    npc:set_intention(Intention.IDLE)
  elseif id == SKILL_TELEPORT_SINGLE then
    teleport_away(npc, player)
    attack_most_hated(npc)
  elseif id == SKILL_TELEPORT_AREA then
    teleport_away(npc, player)
    -- The tracked players on the same floor within 250 go along.
    for i = 1, math.min(tracked_count, TRACK_LIMIT) do
      local p = tracked[i] and world.get_player(tracked[i])
      if p ~= nil and p:object_id() ~= player:object_id() and p:z() > player:z() - Z_TOLERANCE and p:z() < player:z() + Z_TOLERANCE
        and dist2(p:x(), p:y(), player:x(), player:y()) <= AREA_TELEPORT_RANGE * AREA_TELEPORT_RANGE then
        teleport_away(npc, p)
      end
    end
    attack_most_hated(npc)
  end
end)

-- skill_roll is the 1 in 225 draw of his skills at target; within_range
-- checks the distance for 4221 (3D for attackers, 2D for newcomers).
local function skill_roll(npc, target, within_range)
  local roll = rnd(SKILL_ROLL_MAX)
  if roll < 1 then
    cast_at(npc, target, SKILL_TELEPORT_SINGLE)
  elseif roll < 2 then
    cast_at(npc, target, SKILL_TELEPORT_AREA)
  elseif roll < 4 then
    cast_at(npc, target, SKILL_4219)
  elseif roll < 8 then
    cast_at(npc, target, SKILL_4218)
  elseif roll < 15 then
    if not boss.same(target, npc:most_hated()) and within_range() then
      cast_at(npc, target, SKILL_4221)
    end
  end
  if rnd.bool() and boss.same(target, npc:most_hated()) then
    cast_at(npc, target, SKILL_4220)
  end
end

s:on_attack(function(npc, attacker, damage, is_summon)
  if npc:id() == ZAKEN and attacker:is_mounted() and not attacker:is_affected_by_skill(SKILL_MOUNT_DEBUFF) then
    cast_at(npc, attacker, SKILL_MOUNT_DEBUFF)
  end
  local target = original(attacker, is_summon)
  npc:add_damage_hate(target, 0, math.floor(((damage / npc:max_hp()) / 0.05) * 20000))
  if rnd(10) < 1 then
    skill_roll(npc, attacker, function()
      local dx, dy, dz = attacker:x() - npc:x(), attacker:y() - npc:y(), attacker:z() - npc:z()
      return dx * dx + dy * dy + dz * dz <= SKILL_RANGE_CHECK * SKILL_RANGE_CHECK
    end)
  end
  if hour() >= NIGHT_END_HOUR and npc:hp() < npc:max_hp() * day_hp_stage / 4.0 then
    day_hp_stage = day_hp_stage - 1
    set_tp(random_spot())
    cast_self(npc, SKILL_TELEPORT_SELF)
  end
end)

s:on_kill(function(npc, killer, is_summon)
  local id = npc:id()
  if id == ZAKEN then
    boss.sound(npc, "BS02_D")
    s:cancel_quest_timer(TIMER_MAIN, npc, nil)
    s:cancel_quest_timer(TIMER_MINION_WAVES, npc, nil)
    for pid in pairs(pending) do
      s:cancel_quest_timer(MINION_RESPAWN_PREFIX .. pid, npc, nil)
    end
    pending = {}
    zaken = nil
    local delay = boss.respawn_ms(Config.ZAKEN_SPAWN_INTERVAL, Config.ZAKEN_SPAWN_RANDOM)
    boss.died(s, ZAKEN, boss.DEAD, delay, UNLOCK, "Zaken")
    return
  end
  if grandboss.status(ZAKEN) == boss.ALIVE then
    if not boss.alive(zaken) then
      log.warn("Zaken minion respawn skipped: no live Zaken while his status is alive", "minion", id)
      return
    end
    next_respawn = next_respawn + 1
    pending[next_respawn] = { id = id, x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading() }
    s:start_quest_timer(MINION_RESPAWN_PREFIX .. next_respawn, (30 + rnd(60)) * 1000, zaken, nil)
  end
end)

s:on_aggro_range_enter(function(npc, player, is_summon)
  if zones.is_inside(ZONE, npc) then
    npc:add_damage_hate(original(player, is_summon), 1, 200)
  end
  if player:z() > npc:z() - Z_TOLERANCE and player:z() < npc:z() + Z_TOLERANCE then
    if tracked_count < TRACK_LIMIT and rnd(3) < 1 then
      tracked[tracked_count + 1] = player:object_id()
      tracked_count = tracked_count + 1
    end
    if rnd(15) < 1 then
      skill_roll(npc, player, function()
        return dist2(player:x(), player:y(), npc:x(), npc:y()) <= SKILL_RANGE_CHECK * SKILL_RANGE_CHECK
      end)
    end
  end
end)
