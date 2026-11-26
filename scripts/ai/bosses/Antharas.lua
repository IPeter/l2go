-- Antharas, ported from Mobius ai/bosses/Antharas (BazookaRpm).
--
-- Antharas lies asleep in his nest (zone 70050). The Heart of Warding
-- (13001) takes players with a Portal Stone (3865) in; the first one
-- starts the wait (status 1, Config.ANTHARAS_WAIT_TIME minutes), then he
-- rises with a scene and fights (status 2). He keeps three attackers with
-- their own hate and picks his skills by his HP and where they stand; his
-- regeneration grows as he weakens; every 5 minutes Behemoth Dragons (and
-- late Dragon Bombers) come. After his terror he moves to the middle of
-- the nest and calls invisible helpers that send bombers flying. After 15
-- minutes without a hit he goes back to sleep (status 0) and the nest is
-- emptied. Killed (status 3), he comes back after
-- Config.ANTHARAS_SPAWN_INTERVAL hours, give or take ANTHARAS_SPAWN_RANDOM;
-- a cube stands for 15 minutes, then the nest is emptied.
--
-- Deviations: the anti-strider skill is left out (the server has no mounts
-- yet); setOnKillDelay(0) is not ported (his kill handler runs 2.5 s after
-- his death, as for every NPC); the earthquake of his return reaches the
-- players around him, not every player online. Behemoths and bombers are
-- counted only while their killer is in the nest, as in Mobius.

local boss = require("lib.boss")

local s = script.new("Antharas")

local ANTHARAS = 29019
local BEHEMOTH = 29069
local BOMBER = 29070
local HEART = 13001
local TELEPORT_CUBE = 31859

-- The invisible helpers of the terror phase, by template.
local HELPERS = {
  { 29077, 177229, 113298 }, { 29078, 176707, 113585 }, { 29079, 176385, 113889 }, { 29080, 176082, 114241 },
  { 29081, 176066, 114802 }, { 29082, 176095, 115313 }, { 29083, 176425, 115829 }, { 29084, 176949, 116378 },
  { 29085, 177655, 116402 }, { 29086, 178248, 116395 }, { 29087, 178706, 115998 }, { 29088, 179208, 115452 },
  { 29089, 179191, 115079 }, { 29090, 179221, 114546 }, { 29091, 178916, 113925 }, { 29092, 178782, 113814 },
  { 29093, 178419, 113417 }, { 29094, 177855, 113282 },
}
local HELPER_Z = -7735

local PORTAL_STONE = 3865

local JUMP = { 4106, 1 }
local TAIL_ATTACK = { 4107, 1 }
local FEAR = { 4108, 1 }
local DEBUFF = { 4109, 1 }
local MOUTH_ATTACK = { 4110, 2 }
local BREATH_ATTACK = { 4111, 1 }
local NORMAL_ATTACK = { 4112, 1 }
local NORMAL_ATTACK_EX = { 4113, 1 }
local REGEN = { 4125, 4239, 4240, 4241 } -- by phase, level 1
local BOMBER_DISPEL = { 5042, 1 }
local FEAR_SHORT = { 5092, 1 }
local METEOR = { 5093, 1 }

local ZONE = 70050

local ALIVE = 0
local WAITING = 1
local IN_FIGHT = 2
local DEAD = 3

local IDLE_SPOT = { x = 185708, y = 114298, z = -8221 }
local SPAWN_SPOT = { x = 181323, y = 114850, z = -7623, heading = 32542 }
local MOVE_TARGET = { x = 179011, y = 114871, z = -7704 }
local FEAR_MOVE = { x = 177648, y = 114816, z = -7735 }
local EXIT_X, EXIT_Y, EXIT_Z = 79800, 151200, -3534

local CHECK_ATTACK_MS = 60000
local MINION_SPAWN_MS = 300000
local REGEN_MS = 60000
local INACTIVITY_MS = 900000
local CUBE_MS = 900000
local FEAR_COOLDOWN_MS = 300000
local MAX_MOVE_RETRY = 3
local MAX_THREAT_DISTANCE = 9000
local BOMBER_EXPLOSION_RANGE = 230

s:add_start_npc(HEART, TELEPORT_CUBE)
s:add_talk_id(HEART, TELEPORT_CUBE)
s:add_first_talk_id(HEART)
for _, h in ipairs(HELPERS) do
  s:add_spawn_id(h[1])
end
s:add_spawn_id(ANTHARAS)
s:add_move_finished_id(BOMBER)
s:add_aggro_range_enter_id(BOMBER)
s:add_spell_finished_id(ANTHARAS)
s:add_attack_id(ANTHARAS, BOMBER, BEHEMOTH)
s:add_kill_id(ANTHARAS, BEHEMOTH)

-- The fight's state (Mobius's static fields). The three attackers are
-- object IDs: players expire with the handler.
local antharas
local last_attack = 0
local minion_count = 0
local minion_multiplier = 0
local move_retry = 0
local sand_storm = 0
local attackers = { 0, 0, 0 }
local hates = { 0, 0, 0 }

local function skill(holder)
  return skills.get(holder[1], holder[2])
end

local function status()
  return grandboss.status(ANTHARAS)
end

local function reset_fight()
  last_attack, minion_count, minion_multiplier, move_retry, sand_storm = 0, 0, 0, 0, 0
  attackers = { 0, 0, 0 }
  hates = { 0, 0, 0 }
end

local function spawn_idle()
  antharas = s:add_spawn(ANTHARAS, IDLE_SPOT, false, 0)
  grandboss.add_boss(antharas)
  return antharas
end

local function exit_player(p)
  p:teleport(EXIT_X + rnd(600), EXIT_Y + rnd(1100), EXIT_Z)
end

s:on_start(function(ev)
  local live = boss.live_boss(ANTHARAS)
  if live ~= nil then
    -- A reload: Antharas is there; his timers by his status.
    antharas = live
    local st = status()
    if st == IN_FIGHT then
      last_attack = time.now_ms()
      s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, live, nil)
      s:start_quest_timer("SPAWN_MINION", MINION_SPAWN_MS, live, nil)
      s:start_quest_timer("SET_REGEN", REGEN_MS, live, nil)
    elseif st == WAITING then
      s:start_quest_timer("SPAWN_ANTHARAS", Config.ANTHARAS_WAIT_TIME * 60000, nil, nil)
    end
    return
  end
  local info = grandboss.info(ANTHARAS) or { respawn_time = 0, x = 0, y = 0, z = 0, heading = 0, hp = 0, mp = 0 }
  local st = status()
  if st == ALIVE or st == WAITING then
    reset_fight()
    spawn_idle()
    if antharas ~= nil and info.hp > 0 then
      antharas:set_hp_mp(info.hp, info.mp)
    end
    if st == WAITING then
      s:start_quest_timer("SPAWN_ANTHARAS", Config.ANTHARAS_WAIT_TIME * 60000, nil, nil)
    end
  elseif st == IN_FIGHT then
    antharas = boss.spawn_saved(s, ANTHARAS, info)
    grandboss.add_boss(antharas)
    last_attack = time.now_ms()
    s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, antharas, nil)
    s:start_quest_timer("SPAWN_MINION", MINION_SPAWN_MS, antharas, nil)
  elseif st == DEAD then
    local remaining = info.respawn_time - time.now_ms()
    if remaining > 0 then
      s:start_quest_timer("CLEAR_STATUS", remaining, nil, nil)
    else
      grandboss.set_status(ANTHARAS, ALIVE)
      reset_fight()
      spawn_idle()
    end
  end
end)

-- refresh_ai_params keeps three attackers with their hate (Mobius
-- refreshAiParams).
local function refresh_ai_params(attacker, damage)
  local id = attacker:object_id()
  for i = 1, 3 do
    if attackers[i] ~= 0 and attackers[i] == id then
      if hates[i] < damage + 1000 then
        hates[i] = damage + rnd(3000)
      end
      return
    end
  end
  local min = math.min(hates[1], hates[2], hates[3])
  for i = 1, 3 do
    if hates[i] == min then
      hates[i] = damage + rnd(3000)
      attackers[i] = id
      return
    end
  end
end

local function cone(dist, dir, d1, lo1, hi1, d2, lo2, hi2)
  return (dist < d1 and dir < hi1 and dir > lo1) or (dist < d2 and dir < hi2 and dir > lo2)
end

-- choose_skill picks his skill by HP and where the target stands; the
-- second value says whether it is cast at the target (else at himself).
local function choose_skill(npc, dist, dir)
  local hp, max = npc:hp(), npc:max_hp()
  local function tail()
    return rnd(100) < 80 and cone(dist, dir, 1423, 172, 188, 802, 166, 194)
  end
  local function debuff()
    return rnd(100) < 40 and cone(dist, dir, 850, 150, 210, 425, 90, 270)
  end
  local function fear()
    return rnd.bool() and FEAR or FEAR_SHORT
  end
  if hp < max * 0.25 then
    if rnd(100) < 30 then
      return MOUTH_ATTACK, true
    elseif tail() then
      return TAIL_ATTACK, false
    elseif debuff() then
      return DEBUFF, false
    elseif rnd(100) < 10 and dist < 1100 then
      return JUMP, false
    elseif rnd(100) < 10 then
      return METEOR, true
    elseif rnd(100) < 6 then
      return BREATH_ATTACK, true
    elseif rnd.bool() then
      return NORMAL_ATTACK_EX, true
    elseif rnd(100) < 5 then
      return fear(), true
    end
    return NORMAL_ATTACK, true
  elseif hp < max * 0.5 then
    if tail() then
      return TAIL_ATTACK, false
    elseif debuff() then
      return DEBUFF, false
    elseif rnd(100) < 10 and dist < 1100 then
      return JUMP, false
    elseif rnd(100) < 7 then
      return METEOR, true
    elseif rnd(100) < 6 then
      return BREATH_ATTACK, true
    elseif rnd.bool() then
      return NORMAL_ATTACK_EX, true
    elseif rnd(100) < 5 then
      return fear(), true
    end
    return NORMAL_ATTACK, true
  elseif hp < max * 0.75 then
    if tail() then
      return TAIL_ATTACK, false
    elseif rnd(100) < 10 and dist < 1100 then
      return JUMP, false
    elseif rnd(100) < 5 then
      return METEOR, true
    elseif rnd(100) < 6 then
      return BREATH_ATTACK, true
    elseif rnd.bool() then
      return NORMAL_ATTACK_EX, true
    elseif rnd(100) < 5 then
      return fear(), true
    end
    return NORMAL_ATTACK, true
  elseif tail() then
    return TAIL_ATTACK, false
  elseif rnd(100) < 3 then
    return METEOR, true
  elseif rnd(100) < 6 then
    return BREATH_ATTACK, true
  elseif rnd.bool() then
    return NORMAL_ATTACK_EX, true
  elseif rnd(100) < 5 then
    return fear(), true
  end
  return NORMAL_ATTACK, true
end

-- manage_skills casts at the most hated of the three (Mobius manageSkills).
local function manage_skills(npc)
  if npc:is_casting_now() or npc:is_core_ai_disabled() or not npc:is_in_combat() then
    return
  end
  local players = {}
  for i = 1, 3 do
    local p = attackers[i] ~= 0 and world.get_player(attackers[i]) or nil
    players[i] = p
    if p == nil or npc:calculate_distance_3d(p) > MAX_THREAT_DISTANCE or p:is_dead() then
      hates[i] = 0
    end
  end
  local slot, hate, target = 0, 0, nil
  if hates[1] > hates[2] then
    slot, hate, target = 1, hates[1], players[1]
  elseif hates[2] > 0 then
    slot, hate, target = 2, hates[2], players[2]
  end
  if hates[3] > hate then
    slot, hate, target = 3, hates[3], players[3]
  end
  if hate <= 0 then
    return
  end
  if rnd(100) < 70 then
    hates[slot] = 500
  end
  local dist = npc:calculate_distance_3d(target)
  local dir = util.calculate_angle_from(npc, target)
  local holder, on_target = choose_skill(npc, dist, dir)
  local sk = skill(holder)
  if sk ~= nil and npc:check_do_cast_conditions(sk) then
    if on_target then
      boss.cast_desire(npc, target, sk, 100)
    else
      npc:set_intention_cast(sk, npc)
    end
  end
end

-- clear_nest empties the nest: NPCs go (Antharas goes back to sleep when
-- keep_antharas), players go out.
local function clear_nest(keep_antharas, spare_gms)
  for _, n in ipairs(zones.npcs_inside(ZONE)) do
    if keep_antharas and n:id() == ANTHARAS then
      n:teleport(IDLE_SPOT)
    else
      n:delete()
    end
  end
  for _, p in ipairs(zones.players_inside(ZONE)) do
    if not (spare_gms and p:is_gm()) then
      exit_player(p)
    end
  end
end

local function cancel_fight_timers(npc)
  for _, name in ipairs({ "CHECK_ATTACK", "SPAWN_MINION", "SET_REGEN", "TID_USED_FEAR", "TID_FEAR_MOVE_TIMEOVER", "TID_FEAR_COOLTIME" }) do
    s:cancel_quest_timer(name, npc, nil)
  end
end

-- The intro cameras: {next event, delay, camera arguments}.
local CAMERAS = {
  CAMERA_1 = { "CAMERA_2", 3000, { 700, 13, -19, 0, 10000, 20000, 0, 0, 0, 0, 0 } },
  CAMERA_2 = { "CAMERA_3", 10000, { 700, 13, 0, 6000, 10000, 20000, 0, 0, 0, 0, 0 } },
  CAMERA_4 = { "CAMERA_5", 10800, { 1100, 0, -3, 22000, 10000, 30000, 0, 0, 0, 0, 0 } },
  CAMERA_5 = { "START_MOVE", 1900, { 1100, 0, -3, 300, 10000, 7000, 0, 0, 0, 0, 0 } },
}

local function despawn_minions()
  minion_count = 0
  for _, n in ipairs(zones.npcs_inside(ZONE)) do
    if n:id() == BEHEMOTH or n:id() == BOMBER then
      n:delete()
    end
  end
end

s:on_event(function(event, npc, player)
  if event == "ENTER" then
    local st = status()
    if st == DEAD then
      return "13001-01.html"
    elseif st == IN_FIGHT then
      return "13001-02.html"
    elseif not s:has_quest_items(player, PORTAL_STONE) then
      return "13001-03.html"
    end
    s:take_items(player, PORTAL_STONE, 1)
    player:teleport(179700 + rnd(700), 113800 + rnd(2100), -7709)
    if st ~= WAITING then
      grandboss.set_status(ANTHARAS, WAITING)
      s:start_quest_timer("SPAWN_ANTHARAS", Config.ANTHARAS_WAIT_TIME * 60000, nil, nil)
    end
    return nil
  elseif event == "EXIT" then
    exit_player(player)
  elseif event == "SPAWN_ANTHARAS" then
    if not boss.alive(antharas) then
      return
    end
    antharas:disable_core_ai(true)
    antharas:set_random_walking(false)
    antharas:teleport(SPAWN_SPOT)
    grandboss.set_status(ANTHARAS, IN_FIGHT)
    last_attack = time.now_ms()
    cinematics.sound(ZONE, "BS02_A")
    s:start_quest_timer("CAMERA_1", 23, antharas, nil)
  elseif CAMERAS[event] ~= nil then
    if npc ~= nil then
      local c = CAMERAS[event]
      cinematics.special_camera(ZONE, npc, unpack(c[3]))
      s:start_quest_timer(c[1], c[2], npc, nil)
    end
  elseif event == "CAMERA_3" then
    if npc ~= nil then
      cinematics.special_camera(ZONE, npc, 3700, 0, -3, 0, 10000, 10000, 0, 0, 0, 0, 0)
      npc:social_action(1)
      s:start_quest_timer("CAMERA_4", 200, npc, nil)
      s:start_quest_timer("SOCIAL", 5200, npc, nil)
    end
  elseif event == "SOCIAL" then
    if npc ~= nil then
      npc:social_action(2)
    end
  elseif event == "START_MOVE" then
    if npc ~= nil then
      npc:disable_core_ai(false)
      npc:set_random_walking(true)
      for _, p in ipairs(world.players_around(npc, 4000)) do
        if p:is_hero() and Config.ANTHARAS_RECOGNIZE_HERO then
          cinematics.screen_message(ZONE, p:name() .. "!!!! You cannot hope to defeat me with your meager strength.", 4000, 2)
          break
        end
      end
      npc:move_to(MOVE_TARGET)
      s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, npc, nil)
      s:start_quest_timer("SPAWN_MINION", MINION_SPAWN_MS, npc, nil)
    end
  elseif event == "SET_REGEN" then
    if npc ~= nil then
      local hp, max = npc:hp(), npc:max_hp()
      local phase = 1
      if hp < max * 0.25 then
        phase = 4
      elseif hp < max * 0.5 then
        phase = 3
      elseif hp < max * 0.75 then
        phase = 2
      end
      if not npc:is_affected_by_skill(REGEN[phase]) then
        npc:set_intention_cast(skills.get(REGEN[phase], 1), npc)
      end
      s:start_quest_timer("SET_REGEN", REGEN_MS, npc, nil)
    end
  elseif event == "CHECK_ATTACK" then
    if npc == nil then
      return
    end
    if last_attack + INACTIVITY_MS < time.now_ms() then
      grandboss.set_status(ANTHARAS, ALIVE)
      clear_nest(true, false)
      reset_fight()
      cancel_fight_timers(npc)
    else
      for i = 1, 3 do
        if hates[i] > 10 then
          hates[i] = hates[i] - rnd(10)
        end
      end
      manage_skills(npc)
      s:start_quest_timer("CHECK_ATTACK", CHECK_ATTACK_MS, npc, nil)
    end
  elseif event == "SPAWN_MINION" then
    if npc == nil then
      return
    end
    if minion_multiplier > 1 and minion_count < 100 - minion_multiplier * 2 then
      for _ = 1, minion_multiplier do
        s:add_spawn(BEHEMOTH, npc, true)
        s:add_spawn(BEHEMOTH, npc, true)
      end
      minion_count = minion_count + minion_multiplier * 2
    elseif minion_count < 98 then
      s:add_spawn(BEHEMOTH, npc, true)
      s:add_spawn(BEHEMOTH, npc, true)
      minion_count = minion_count + 2
    elseif minion_count < 99 then
      s:add_spawn(rnd.bool() and BEHEMOTH or BOMBER, npc, true)
      minion_count = minion_count + 1
    end
    if rnd(100) > 10 and minion_multiplier < 4 then
      minion_multiplier = minion_multiplier + 1
    end
    s:start_quest_timer("SPAWN_MINION", MINION_SPAWN_MS, npc, nil)
  elseif event == "CLEAR_ZONE" then
    clear_nest(false, false)
  elseif event == "TID_USED_FEAR" then
    if npc ~= nil and sand_storm == 0 then
      sand_storm = 1
      npc:disable_core_ai(true)
      npc:move_to(FEAR_MOVE)
      s:start_quest_timer("TID_FEAR_MOVE_TIMEOVER", 2000, npc, nil)
      s:start_quest_timer("TID_FEAR_COOLTIME", FEAR_COOLDOWN_MS, npc, nil)
    end
  elseif event == "TID_FEAR_COOLTIME" then
    sand_storm = 0
  elseif event == "TID_FEAR_MOVE_TIMEOVER" then
    if npc == nil then
      return
    end
    if sand_storm == 1 and npc:x() == FEAR_MOVE.x and npc:y() == FEAR_MOVE.y then
      sand_storm = 2
      move_retry = 0
      npc:disable_core_ai(false)
      for _, h in ipairs(HELPERS) do
        s:add_spawn(h[1], { x = h[2], y = h[3], z = HELPER_Z })
      end
    elseif sand_storm == 1 then
      if move_retry <= MAX_MOVE_RETRY then
        move_retry = move_retry + 1
        npc:move_to(FEAR_MOVE)
        s:start_quest_timer("TID_FEAR_MOVE_TIMEOVER", 5000, npc, nil)
      else
        npc:teleport({ x = FEAR_MOVE.x, y = FEAR_MOVE.y, z = FEAR_MOVE.z, heading = npc:heading() })
        s:start_quest_timer("TID_FEAR_MOVE_TIMEOVER", 1000, npc, nil)
      end
    end
  elseif event == "CLEAR_STATUS" then
    local a = spawn_idle()
    cinematics.earthquake(a, IDLE_SPOT, 20, 10)
    grandboss.set_status(ANTHARAS, ALIVE)
    reset_fight()
  elseif event == "SKIP_WAITING" then
    if status() == WAITING then
      s:cancel_quest_timer("SPAWN_ANTHARAS", nil, nil)
      s:start_quest_timer("SPAWN_ANTHARAS", 0, nil, nil)
      if player ~= nil then
        player:send_message("Antharas: Skipping waiting time ...")
      end
    elseif player ~= nil then
      player:send_message("Antharas: You can't skip waiting time right now!")
    end
  elseif event == "RESPAWN_ANTHARAS" then
    if status() == DEAD then
      s:cancel_quest_timer("CLEAR_STATUS", nil, nil)
      s:start_quest_timer("CLEAR_STATUS", 0, nil, nil)
      if player ~= nil then
        player:send_message("Antharas: Antharas has been respawned.")
      end
    elseif player ~= nil then
      player:send_message("Antharas: You can't respawn antharas while antharas is alive!")
    end
  elseif event == "DESPAWN_MINIONS" then
    if status() == IN_FIGHT then
      despawn_minions()
      if player ~= nil then
        player:send_message("Antharas: All minions have been deleted!")
      end
    elseif player ~= nil then
      player:send_message("Antharas: You can't despawn minions right now!")
    end
  elseif event == "ABORT_FIGHT" then
    if status() == IN_FIGHT then
      grandboss.set_status(ANTHARAS, ALIVE)
      if boss.alive(antharas) then
        cancel_fight_timers(antharas)
      end
      clear_nest(true, true)
      reset_fight()
      if player ~= nil then
        player:send_message("Antharas: Fight has been aborted!")
      end
    elseif player ~= nil then
      player:send_message("Antharas: You can't abort fight right now!")
    end
  elseif event == "MANAGE_SKILL" then
    if npc ~= nil then
      manage_skills(npc)
    end
  end
end)

-- A bomber explodes on the first player in its range.
s:on_aggro_range_enter(function(npc, player, is_summon)
  if npc:id() ~= BOMBER then
    return
  end
  npc:cast(skill(BOMBER_DISPEL))
  npc:die(player)
end)

s:on_attack(function(npc, attacker, damage, is_summon, sk)
  local id = npc:id()
  if id == BOMBER then
    if npc:calculate_distance_3d(attacker) < BOMBER_EXPLOSION_RANGE then
      npc:cast(skill(BOMBER_DISPEL))
      npc:die(attacker)
    end
  elseif id == ANTHARAS then
    last_attack = time.now_ms()
    if not zones.is_inside(ZONE, attacker) or status() ~= IN_FIGHT then
      log.warn("Antharas: a player attacked Antharas in invalid conditions", "player", attacker:name())
      attacker:teleport(80464, 152294, -3534)
    end
    local hp, max = npc:hp(), npc:max_hp()
    if sk == nil then
      refresh_ai_params(attacker, damage * 1000)
    elseif hp < max * 0.25 then
      refresh_ai_params(attacker, math.floor(damage / 3) * 100)
    elseif hp < max * 0.5 then
      refresh_ai_params(attacker, damage * 20)
    elseif hp < max * 0.75 then
      refresh_ai_params(attacker, damage * 10)
    else
      refresh_ai_params(attacker, math.floor(damage / 3) * 20)
    end
    manage_skills(npc)
  end
end)

s:on_kill(function(npc, killer, is_summon)
  local id = npc:id()
  if id == ANTHARAS then
    if killer == nil or not zones.is_inside(ZONE, killer) then
      log.warn("Antharas was killed outside of the nest; forcing the death handling")
    end
    antharas = nil
    despawn_minions()
    cinematics.special_camera(ZONE, npc, 1200, 20, -10, 0, 10000, 13000, 0, 0, 0, 0, 0)
    cinematics.sound(ZONE, "BS01_D")
    s:add_spawn(TELEPORT_CUBE, { x = 177615, y = 114941, z = -7709, heading = 0 }, false, CUBE_MS)
    local delay = boss.respawn_ms(Config.ANTHARAS_SPAWN_INTERVAL, Config.ANTHARAS_SPAWN_RANDOM)
    s:cancel_quest_timer("SET_REGEN", npc, nil)
    s:cancel_quest_timer("CHECK_ATTACK", npc, nil)
    s:cancel_quest_timer("SPAWN_MINION", npc, nil)
    s:start_quest_timer("CLEAR_ZONE", CUBE_MS, nil, nil)
    boss.died(s, ANTHARAS, DEAD, delay, "CLEAR_STATUS", "Antharas")
    reset_fight()
  elseif id == BEHEMOTH and killer ~= nil and zones.is_inside(ZONE, killer) then
    minion_count = minion_count - 1
  end
end)

-- A bomber reached its spot and explodes.
s:on_move_finished(function(npc)
  npc:cast(skill(BOMBER_DISPEL))
  npc:die()
end)

s:on_spawn(function(npc)
  if npc:id() == ANTHARAS then
    npc:set_can_return_to_spawn_point(false)
    npc:set_random_walking(false)
    s:cancel_quest_timer("SET_REGEN", npc, nil)
    s:start_quest_timer("SET_REGEN", REGEN_MS, npc, nil)
    return
  end
  -- An invisible helper sends six bombers to its suicide spots and goes.
  for i = 1, 6 do
    local x = npc:param_int("suicide" .. i .. "_x")
    local y = npc:param_int("suicide" .. i .. "_y")
    local bomber = s:add_spawn(BOMBER, { x = npc:x(), y = npc:y(), z = npc:z(), heading = 0 }, true, 15000, true)
    if bomber ~= nil then
      bomber:move_to(x, y, npc:z())
    end
  end
  npc:delete()
end)

s:on_spell_finished(function(npc, player, sk)
  if sk ~= nil and (sk:id() == FEAR[1] or sk:id() == FEAR_SHORT[1]) then
    s:start_quest_timer("TID_USED_FEAR", 7000, npc, nil)
  end
  s:start_quest_timer("MANAGE_SKILL", 1000, npc, nil)
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)
