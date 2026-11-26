-- Baium, ported from Mobius ai/bosses/Baium (BazookaRpm).
--
-- Baium sits as a stone statue (29025) on the top of the Tower of
-- Insolence (zone 70051). The Angelic Vortex (31862) takes a player with a
-- Blooded Fabric (4295) up; whoever wakes the statue starts the fight
-- (status 2): a scene, Baium's gift to the one who woke him (or another
-- player around), then five archangels and the fight. Baium keeps three
-- attackers with their own hate (NPC variables c_quest0..2 and
-- i_quest0..2) and picks his skill by his HP; after 30 minutes without a
-- hit he turns back to stone and the top is emptied. Killed (status 3), he
-- comes back after Config.BAIUM_SPAWN_INTERVAL hours, give or take
-- BAIUM_SPAWN_RANDOM; a cube stands for 15 minutes, then the top is
-- emptied.
--
-- A strider rider hitting Baium gets Hinder Strider (4258) unless it has it
-- or the skill is under reuse.
--
-- Deviations: Baium's wake-up social actions and the archangels' "first
-- playable in range" are seen and taken by the players around him instead
-- of those in the zone, and summons are not candidates. Mobius sets a
-- stored WAITING status back to ALIVE without spawning the statue; so does
-- this port.

local boss = require("lib.boss")

local s = script.new("Baium")

local BAIUM = 29020
local BAIUM_STONE = 29025
local ANGELIC_VORTEX = 31862
local ARCHANGEL = 29021
local TELEPORT_CUBE = 31842

local BLOODED_FABRIC = 4295

local SKILL_BAIUM_ATTACK = 4127
local SKILL_ENERGY_WAVE = 4128
local SKILL_EARTHQUAKE = 4129
local SKILL_THUNDERBOLT = 4130
local SKILL_GROUP_HOLD = 4131
local SKILL_SPEAR_ATTACK = 4132
local SKILL_ANGEL_HEAL = 4133
local SKILL_HEAL_OF_BAIUM = 4135
local SKILL_BAIUM_PRESENT = 4136
local SKILL_ANTI_STRIDER = 4258 -- Hinder Strider

local MOUNT_STRIDER = 1 -- player:mount_type()

local ZONE = 70051

local ALIVE = 0
local WAITING = 1
local IN_FIGHT = 2
local DEAD = 3

local CHECK_ATTACK_MS = 60000
local INACTIVITY_MS = 1800000
local CUBE_LIFETIME_MS = 900000
local SELECT_TARGET_MS = 5000
local PLAYER_PORT_RADIUS = 16000

local GIFT = { x = 115910, y = 17337, z = 10105 }
local BAIUM_SPAWN = { x = 116033, y = 17447, z = 10107, heading = 40188 }
local CUBE_SPOT = { x = 115017, y = 15549, z = 10090 }
local TELEPORT_IN = { x = 114077, y = 15882, z = 10078 }
local TELEPORT_OUT = {
  { x = 108784, y = 16000, z = -4928 },
  { x = 113824, y = 10448, z = -5164 },
  { x = 115488, y = 22096, z = -5168 },
}
local ARCHANGELS = {
  { x = 115792, y = 16608, z = 10136, heading = 0 },
  { x = 115168, y = 17200, z = 10136, heading = 0 },
  { x = 115780, y = 15564, z = 10136, heading = 13620 },
  { x = 114880, y = 16236, z = 10136, heading = 5400 },
  { x = 114239, y = 17168, z = 10136, heading = -1992 },
}

s:add_first_talk_id(ANGELIC_VORTEX)
s:add_talk_id(ANGELIC_VORTEX, TELEPORT_CUBE, BAIUM_STONE)
s:add_start_npc(ANGELIC_VORTEX, TELEPORT_CUBE, BAIUM_STONE)
s:add_attack_id(BAIUM, ARCHANGEL)
s:add_kill_id(BAIUM)
s:add_spell_finished_id(BAIUM)
s:add_creature_see_id(BAIUM)

local baium
local last_attack = 0

local function status()
  return grandboss.status(BAIUM)
end

local function spawn_stone()
  s:add_spawn(BAIUM_STONE, BAIUM_SPAWN, false, 0)
end

local function in_zone(obj)
  return obj ~= nil and zones.is_inside(ZONE, obj)
end

local function spawn_archangels()
  for _, loc in ipairs(ARCHANGELS) do
    local a = s:add_spawn(ARCHANGEL, loc, false, 0, true)
    if a ~= nil then
      s:start_quest_timer("SELECT_TARGET", SELECT_TARGET_MS, a, nil)
    end
  end
end

-- random_player is the first living player within 2,000 of npc on the top.
local function random_player(npc)
  for _, p in ipairs(world.players_around(npc, 2000)) do
    if in_zone(p) and not p:is_dead() then
      return p
    end
  end
  return nil
end

s:on_start(function(ev)
  local live = boss.live_boss(BAIUM)
  if live ~= nil then
    -- A reload during the fight: Baium and the archangels are there.
    baium = live
    last_attack = time.now_ms()
    for _, a in ipairs(world.npcs_by_id(ARCHANGEL)) do
      s:start_quest_timer("SELECT_TARGET", SELECT_TARGET_MS, a, nil)
    end
    s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, live, nil)
    return
  end
  local info = grandboss.info(BAIUM) or { respawn_time = 0, x = 0, y = 0, z = 0, heading = 0, hp = 0, mp = 0 }
  local st = status()
  if st == WAITING then
    grandboss.set_status(BAIUM, ALIVE)
  elseif st == ALIVE then
    if #world.npcs_by_id(BAIUM_STONE) == 0 then
      spawn_stone()
    end
  elseif st == IN_FIGHT then
    baium = boss.spawn_saved(s, BAIUM, info)
    last_attack = time.now_ms()
    grandboss.add_boss(baium)
    spawn_archangels()
    s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, baium, nil)
  elseif st == DEAD then
    local remaining = info.respawn_time - time.now_ms()
    if remaining > 0 then
      s:start_quest_timer("CLEAR_STATUS", remaining, nil, nil)
    else
      grandboss.set_status(BAIUM, ALIVE)
      spawn_stone()
    end
  end
end)

-- index_of returns the first index (0–2) of the smallest (or largest)
-- value (Mobius MathUtil.getIndexOfMinValue, getIndexOfMaxValue).
local function index_of(values, smallest)
  local best = 0
  for i = 1, 2 do
    if (smallest and values[i + 1] < values[best + 1]) or (not smallest and values[i + 1] > values[best + 1]) then
      best = i
    end
  end
  return best
end

local function slots(npc)
  return { npc:var_int("i_quest0"), npc:var_int("i_quest1"), npc:var_int("i_quest2") }
end

-- refresh_ai_params keeps three attackers with their hate: a known one
-- gets new hate under its threshold, a new one replaces the least hated.
local function refresh_ai_params(attacker, npc, damage, aggro)
  aggro = aggro or damage
  local value = damage + rnd(3000)
  local threshold = aggro + 1000
  for i = 0, 2 do
    if npc:var_int("c_quest" .. i, -1) == attacker:object_id() then
      if npc:var_int("i_quest" .. i) < threshold then
        npc:set_var("i_quest" .. i, value)
      end
      return
    end
  end
  local i = index_of(slots(npc), true)
  npc:set_var("i_quest" .. i, value)
  npc:set_var("c_quest" .. i, attacker:object_id())
end

-- pick returns the first skill whose 1 in 10 roll wins, else the last.
local function pick(list)
  for i = 1, #list - 1 do
    if rnd(100) < 10 then
      return list[i]
    end
  end
  return list[#list]
end

-- manage_skills casts at the most hated of the three (Mobius manageSkills).
local function manage_skills(npc)
  if npc:is_casting_now() or npc:is_core_ai_disabled() or not npc:is_in_combat() then
    return
  end
  for i = 0, 2 do
    local id = npc:var_int("c_quest" .. i, -1)
    local attacker = id >= 0 and world.find_object(id) or nil
    if attacker == nil or npc:calculate_distance_3d(attacker) > 9000 or attacker:is_dead() then
      npc:set_var("i_quest" .. i, 0)
    end
  end
  local max = index_of(slots(npc), false)
  local id = npc:var_int("c_quest" .. max, -1)
  local target = id >= 0 and world.find_object(id) or nil
  if npc:var_int("i_quest" .. max) > 0 and rnd(100) < 70 then
    npc:set_var("i_quest" .. max, 500)
  end
  local skill
  if target ~= nil and not target:is_dead() then
    local ratio = npc:hp() / npc:max_hp()
    if ratio > 0.75 then
      skill = pick({ SKILL_ENERGY_WAVE, SKILL_EARTHQUAKE, SKILL_BAIUM_ATTACK })
    elseif ratio > 0.5 then
      skill = pick({ SKILL_GROUP_HOLD, SKILL_ENERGY_WAVE, SKILL_EARTHQUAKE, SKILL_BAIUM_ATTACK })
    else
      skill = pick({ SKILL_THUNDERBOLT, SKILL_GROUP_HOLD, SKILL_ENERGY_WAVE, SKILL_EARTHQUAKE, SKILL_BAIUM_ATTACK })
    end
  end
  local sk = skill and skills.get(skill, 1)
  if sk ~= nil and npc:check_do_cast_conditions(sk) then
    npc:set_target(target)
    npc:cast(sk)
  end
end

local function exit(player)
  local dest = rnd.entry(TELEPORT_OUT)
  local dx, dy = rnd(100), rnd(100)
  player:teleport(dest.x + dx, dest.y + dy, dest.z)
end

local function clear_zone()
  for _, n in ipairs(zones.npcs_inside(ZONE)) do
    n:delete()
  end
  for _, p in ipairs(zones.players_inside(ZONE)) do
    exit(p)
  end
end

-- select_target is an archangel's choice every 5 s (Mobius SELECT_TARGET).
local function select_target(npc)
  if not boss.alive(baium) then
    npc:delete()
    return
  end
  local most = npc:most_hated()
  local function go(target)
    if not boss.same(npc:target(), target) then
      npc:clear_aggro_list()
    end
    boss.attack_desire(npc, target)
  end
  if most ~= nil and most:is_player() and in_zone(most) then
    go(most)
  else
    local candidate
    for _, p in ipairs(world.players_around(npc, 1000)) do
      if in_zone(p) and not p:is_dead() then
        candidate = p
        break
      end
    end
    if candidate ~= nil then
      go(candidate)
    elseif npc:is_inside_radius_3d(baium, 40) then
      go(baium)
    else
      npc:follow(baium)
    end
  end
  s:start_quest_timer("SELECT_TARGET", SELECT_TARGET_MS, npc, nil)
end

s:on_event(function(event, npc, player)
  if event == "31862-04.html" then
    return event
  elseif event == "ENTER" then
    local st = status()
    if st == DEAD then
      return "31862-03.html"
    elseif st == IN_FIGHT then
      return "31862-02.html"
    elseif not s:has_quest_items(player, BLOODED_FABRIC) then
      return "31862-01.html"
    end
    s:take_items(player, BLOODED_FABRIC, 1)
    player:teleport(TELEPORT_IN)
    return nil
  elseif event == "EXIT" then
    exit(player)
  elseif event == "WAKE_UP" then
    if status() == ALIVE then
      grandboss.set_status(BAIUM, IN_FIGHT)
      baium = s:add_spawn(BAIUM, BAIUM_SPAWN, false, 0)
      baium:disable_core_ai(true)
      baium:set_random_walking(false)
      grandboss.add_boss(baium)
      last_attack = time.now_ms()
      s:start_quest_timer("WAKEUP_ACTION", 2500, baium, nil)
      s:start_quest_timer("MANAGE_EARTHQUAKE", 6000, baium, nil)
      s:start_quest_timer("SOCIAL_ACTION", 20000, baium, player)
      s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, baium, nil)
      npc:delete()
    end
  elseif event == "WAKEUP_ACTION" then
    if npc ~= nil and boss.alive(baium) then
      baium:social_action(2)
    end
  elseif event == "MANAGE_EARTHQUAKE" then
    if npc ~= nil then
      cinematics.earthquake(ZONE, npc, 40, 10)
      cinematics.sound(ZONE, "BS02_A")
    end
  elseif event == "SOCIAL_ACTION" then
    if npc ~= nil then
      npc:social_action(3)
      s:start_quest_timer("PLAYER_PORT", 6000, npc, player)
    end
  elseif event == "PLAYER_PORT" then
    if npc ~= nil then
      local target = player
      if target == nil or not target:is_inside_radius_3d(npc, PLAYER_PORT_RADIUS) then
        target = random_player(npc)
      end
      if target ~= nil then
        target:teleport(GIFT)
      end
      s:start_quest_timer("PLAYER_KILL", 3000, npc, target)
    end
  elseif event == "PLAYER_KILL" then
    if npc ~= nil and player ~= nil and player:is_inside_radius_3d(npc, PLAYER_PORT_RADIUS) then
      npc:social_action(1)
      npc:broadcast_say(ChatType.NPC_GENERAL, "How dare you wake me! Now you shall die!")
      npc:set_target(player)
      npc:cast(skills.get(SKILL_BAIUM_PRESENT, 1))
    end
    for _, p in ipairs(zones.players_inside(ZONE)) do
      if p:is_hero() and Config.BAIUM_RECOGNIZE_HERO then
        cinematics.screen_message(ZONE, "Not even the gods themselves could touch me. But you, " .. p:name()
          .. ", you dare challenge me?! Ignorant mortal!", 4000, 2)
        break
      end
    end
    s:start_quest_timer("SPAWN_ARCHANGEL", 8000, npc, player)
  elseif event == "SPAWN_ARCHANGEL" then
    if not boss.alive(baium) or status() ~= IN_FIGHT then
      return
    end
    baium:disable_core_ai(false)
    baium:set_random_walking(true)
    spawn_archangels()
    if player ~= nil and not player:is_dead() then
      boss.attack_desire(npc, player)
    else
      local target = random_player(npc)
      if target ~= nil then
        boss.attack_desire(npc, target)
      end
    end
  elseif event == "SELECT_TARGET" then
    if npc ~= nil then
      select_target(npc)
    end
  elseif event == "CHECK_ATTACK" then
    if npc ~= nil and last_attack + INACTIVITY_MS < time.now_ms() then
      s:cancel_quest_timers("SELECT_TARGET")
      clear_zone()
      spawn_stone()
      grandboss.set_status(BAIUM, ALIVE)
    elseif npc ~= nil then
      if last_attack + 300000 < time.now_ms() and npc:hp() < npc:max_hp() * 0.75 then
        npc:set_target(npc)
        npc:cast(skills.get(SKILL_HEAL_OF_BAIUM, 1))
      end
      s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, npc, nil)
    end
  elseif event == "CLEAR_STATUS" then
    grandboss.set_status(BAIUM, ALIVE)
    spawn_stone()
  elseif event == "CLEAR_ZONE" then
    clear_zone()
  elseif event == "RESPAWN_BAIUM" then
    if status() == DEAD then
      local info = grandboss.info(BAIUM)
      if info ~= nil then
        info.respawn_time = time.now_ms()
        grandboss.set_info(BAIUM, info)
      end
      s:cancel_quest_timer("CLEAR_STATUS", nil, nil)
      s:cancel_quest_timers("CLEAR_ZONE")
      grandboss.set_status(BAIUM, ALIVE)
      spawn_stone()
    elseif player ~= nil then
      player:send_message("Baium: You cannot respawn Baium while Baium is alive!")
    end
  elseif event == "ABORT_FIGHT" then
    if status() == IN_FIGHT then
      baium = nil
      clear_zone()
      grandboss.set_status(BAIUM, ALIVE)
      spawn_stone()
      if player ~= nil then
        player:send_message("Baium: Aborting fight!")
      end
    elseif player ~= nil then
      player:send_message("Baium: You cannot abort attack right now!")
    end
    s:cancel_quest_timers("CHECK_ATTACK")
    s:cancel_quest_timers("SELECT_TARGET")
  elseif event == "DESPAWN_MINIONS" then
    if status() == IN_FIGHT then
      for _, n in ipairs(zones.npcs_inside(ZONE)) do
        if n:id() == ARCHANGEL then
          n:delete()
        end
      end
      if player ~= nil then
        player:send_message("Baium: All archangels has been deleted!")
      end
    elseif player ~= nil then
      player:send_message("Baium: You cannot despawn archangels right now!")
    end
  elseif event == "MANAGE_SKILLS" then
    if npc ~= nil then
      manage_skills(npc)
    end
  end
end)

s:on_attack(function(npc, attacker, damage, is_summon, skill)
  last_attack = time.now_ms()
  local id = npc:id()
  if id == BAIUM then
    local hinder = skills.get(SKILL_ANTI_STRIDER, 1)
    if attacker:mount_type() == MOUNT_STRIDER and not attacker:is_affected_by_skill(SKILL_ANTI_STRIDER)
      and not npc:is_skill_disabled(hinder) then
      npc:set_target(attacker)
      npc:cast(hinder)
    end
    local ratio = npc:hp() / npc:max_hp()
    if skill == nil then
      refresh_ai_params(attacker, npc, damage * 1000)
    elseif ratio < 0.25 then
      refresh_ai_params(attacker, npc, math.floor(damage / 3) * 100)
    elseif ratio < 0.5 then
      refresh_ai_params(attacker, npc, damage * 20)
    elseif ratio < 0.75 then
      refresh_ai_params(attacker, npc, damage * 10)
    else
      refresh_ai_params(attacker, npc, math.floor(damage / 3) * 20)
    end
    manage_skills(npc)
  elseif id == ARCHANGEL then
    local most = npc:most_hated()
    local spear = skills.get(SKILL_SPEAR_ATTACK, 1)
    if rnd(100) < 10 and spear ~= nil and npc:check_do_cast_conditions(spear) then
      if most ~= nil and npc:calculate_distance_3d(most) < 1000 and in_zone(most) then
        npc:set_target(most)
        npc:cast(spear)
      elseif in_zone(attacker) then
        npc:set_target(attacker)
        npc:cast(spear)
      end
    end
    local heal = skills.get(SKILL_ANGEL_HEAL, 1)
    if rnd(100) < 5 and npc:hp() / npc:max_hp() < 0.5 and heal ~= nil and npc:check_do_cast_conditions(heal) then
      npc:set_target(npc)
      npc:cast(heal)
    end
  end
end)

s:on_kill(function(npc, killer, is_summon)
  if not in_zone(killer) then
    return
  end
  grandboss.set_status(BAIUM, DEAD)
  s:add_spawn(TELEPORT_CUBE, CUBE_SPOT, false, CUBE_LIFETIME_MS)
  cinematics.sound(ZONE, "BS01_D")
  local delay = boss.respawn_ms(Config.BAIUM_SPAWN_INTERVAL, Config.BAIUM_SPAWN_RANDOM)
  boss.died(s, BAIUM, DEAD, delay, "CLEAR_STATUS", "Baium")
  s:start_quest_timer("CLEAR_ZONE", CUBE_LIFETIME_MS, nil, nil)
  s:cancel_quest_timer("CHECK_ATTACK", npc, nil)
  s:cancel_quest_timers("SELECT_TARGET")
end)

-- Baium hates the creatures he sees on the top, healers more as he weakens.
s:on_creature_see(function(npc, creature)
  if not in_zone(creature) or (not creature:is_player() and creature:id() == BAIUM_STONE) then
    return
  end
  local ratio = npc:hp() / npc:max_hp()
  if creature:is_player() and creature:is_in_category(Category.CLERIC_GROUP) then
    if ratio < 0.25 then
      refresh_ai_params(creature, npc, 10000)
    elseif ratio < 0.5 then
      refresh_ai_params(creature, npc, 10000, 6000)
    elseif ratio < 0.75 then
      refresh_ai_params(creature, npc, 10000, 3000)
    else
      refresh_ai_params(creature, npc, 10000, 2000)
    end
  else
    refresh_ai_params(creature, npc, 10000, 1000)
  end
  manage_skills(npc)
end)

s:on_spell_finished(function(npc, player, skill)
  s:start_quest_timer("MANAGE_SKILLS", 1000, npc, nil)
  if boss.alive(baium) and not in_zone(npc) then
    baium:teleport(BAIUM_SPAWN)
  end
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)
