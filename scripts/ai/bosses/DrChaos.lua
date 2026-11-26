-- Dr. Chaos, ported from Mobius ai/bosses/DrChaos (BazookaRpm).
--
-- Doctor Chaos (32033) lives in Pavel's Ruins and trusts no one: every
-- second each living player within 500 of him and every talk to him wears
-- his patience down (30 at his spawn). At 0 he walks to his box, and after
-- a short scene the Chaos Golem (25512, the grand boss) appears. A golem
-- left alone for 30 minutes turns back into the doctor. Killed, the doctor
-- comes back 12–60 hours later. The store's status is 0 for the doctor, 1
-- for the golem and 2 for dead.

local boss = require("lib.boss")

local s = script.new("DrChaos")

local DOCTOR_CHAOS = 32033
local CHAOS_GOLEM = 25512

local NORMAL = 0
local CRAZY = 1
local DEAD = 2

local SPAWN = { x = 96320, y = -110912, z = -3328, heading = 8191 }
local RETURN = { x = 96320, y = -110912, z = -3328 }
local MOVE = { x = 95928, y = -110671, z = -3340 }
local BOX = { x = 96323, y = -110914, z = -3328 }
local GOLEM_SPAWN = { x = 96080, y = -110822, z = -3343 }

local GOLEM_DESPAWN_CHECK_MS = 60000
local GOLEM_INACTIVITY_MS = 1800000
local PARANOIA_TICK_MS = 1000
local DISTANCE_CHECK_MS = 10000

local PARANOIA_START = 30
local PARANOIA_FIRST_MESSAGE = 15
local PARANOIA_SECOND_UPPER = 20
local PARANOIA_THIRD_UPPER = 10
local PARANOIA_RANGE = 500

local RETURN_DISTANCE = 2000
local DISTANCE_LIMIT = 10000

s:add_first_talk_id(DOCTOR_CHAOS)
s:add_spawn_id(DOCTOR_CHAOS)
s:add_kill_id(CHAOS_GOLEM)
s:add_attack_id(CHAOS_GOLEM)

local last_golem_attack = 0
local paranoia = PARANOIA_START

local function status()
  return grandboss.status(CHAOS_GOLEM)
end

local function spawn_doctor()
  return s:add_spawn(DOCTOR_CHAOS, SPAWN, false, 0, false)
end

local function golem_timers(golem)
  last_golem_attack = time.now_ms()
  s:start_quest_timer("golem_despawn", GOLEM_DESPAWN_CHECK_MS, golem, nil, true)
end

s:on_start(function(ev)
  -- A reload: the doctor or the golem is there.
  local golem = boss.live_boss(CHAOS_GOLEM)
  if golem ~= nil then
    golem_timers(golem)
    s:start_quest_timer("DISTANCE_CHECK", DISTANCE_CHECK_MS, golem, nil, true)
    return
  end
  for _, n in ipairs(world.npcs_by_id(DOCTOR_CHAOS)) do
    if not n:is_dead() then
      s:start_quest_timer("paranoia_activity", PARANOIA_TICK_MS, n, nil, true)
      s:start_quest_timer("DISTANCE_CHECK", DISTANCE_CHECK_MS, n, nil, true)
      return
    end
  end

  local info = grandboss.info(CHAOS_GOLEM)
  local st = status()
  if st == DEAD then
    if info ~= nil then
      local remaining = info.respawn_time - time.now_ms()
      if remaining > 0 then
        s:start_quest_timer("reset_drchaos", remaining, nil, nil, false)
        return
      end
    end
    spawn_doctor()
    grandboss.set_status(CHAOS_GOLEM, NORMAL)
  elseif st == CRAZY then
    if info ~= nil then
      local g = s:add_spawn(CHAOS_GOLEM, { x = info.x, y = info.y, z = info.z, heading = info.heading }, false, 0, false)
      grandboss.add_boss(g)
      g:set_hp_mp(math.floor(info.hp), math.floor(info.mp))
      g:set_running()
      golem_timers(g)
    else
      spawn_doctor()
      grandboss.set_status(CHAOS_GOLEM, NORMAL)
    end
  else
    spawn_doctor()
  end
end)

-- becomes_angry starts the scene that turns the doctor into the golem.
local function becomes_angry(npc)
  if status() ~= NORMAL then
    return
  end
  grandboss.set_status(CHAOS_GOLEM, CRAZY)
  s:cancel_quest_timer("paranoia_activity", npc, nil)
  npc:move_to(BOX)
  npc:broadcast_say(ChatType.NPC_GENERAL, "Fools! Why haven't you fled yet? Prepare to learn a lesson!")
  s:start_quest_timer("1", 2000, npc, nil, false)
  s:start_quest_timer("2", 4000, npc, nil, false)
  s:start_quest_timer("3", 6500, npc, nil, false)
  s:start_quest_timer("4", 12500, npc, nil, false)
  s:start_quest_timer("5", 17000, npc, nil, false)
end

s:on_event(function(event, npc, player)
  if event == "reset_drchaos" then
    grandboss.set_status(CHAOS_GOLEM, NORMAL)
    spawn_doctor()
  elseif event == "golem_despawn" then
    if npc ~= nil and npc:id() == CHAOS_GOLEM and last_golem_attack + GOLEM_INACTIVITY_MS < time.now_ms() then
      local chaos = spawn_doctor()
      grandboss.set_status(CHAOS_GOLEM, NORMAL)
      s:cancel_quest_timer("golem_despawn", chaos, nil)
      s:cancel_quest_timers("DISTANCE_CHECK")
      npc:delete()
    end
  elseif event == "1" then
    npc:social_action(2)
    cinematics.special_camera(npc, npc, 1, -200, 15, 5500, 1000, 13500, 0, 0, 0, 0, 0)
  elseif event == "2" then
    npc:social_action(3)
  elseif event == "3" then
    npc:social_action(1)
  elseif event == "4" then
    cinematics.special_camera(npc, npc, 1, -150, 10, 3500, 1000, 5000, 0, 0, 0, 0, 0)
    npc:move_to(MOVE)
  elseif event == "5" then
    local golem = s:add_spawn(CHAOS_GOLEM, GOLEM_SPAWN, false, 0, false)
    grandboss.add_boss(golem)
    s:start_quest_timer("DISTANCE_CHECK", DISTANCE_CHECK_MS, golem, nil, true)
    cinematics.special_camera(golem, npc, 30, 200, 20, 6000, 700, 8000, 0, 0, 0, 0, 0)
    npc:social_action(1)
    cinematics.sound(golem, "Rm03_A")
    golem_timers(golem)
    npc:delete()
  elseif event == "paranoia_activity" then
    if status() == NORMAL then
      for _, p in ipairs(world.players_around(npc, PARANOIA_RANGE)) do
        if not p:is_dead() then
          paranoia = paranoia - 1
          if paranoia == PARANOIA_FIRST_MESSAGE then
            npc:broadcast_say(ChatType.NPC_GENERAL, "How dare you trespass into my territory! Have you no fear?")
          end
          if paranoia <= 0 then
            becomes_angry(npc)
          end
        end
      end
    end
    if npc:calculate_distance_2d(RETURN) > RETURN_DISTANCE then
      npc:clear_aggro_list()
      npc:move_to(RETURN)
    end
  elseif event == "DISTANCE_CHECK" then
    if npc == nil or npc:is_dead() then
      s:cancel_quest_timers("DISTANCE_CHECK")
    else
      local home = npc:spawn_location()
      if home ~= nil and npc:calculate_distance_2d(home) > DISTANCE_LIMIT then
        npc:clear_aggro_list()
        npc:teleport(home)
      end
    end
  end
end)

s:on_first_talk(function(npc, player)
  local html = ""
  if status() == NORMAL then
    paranoia = paranoia - (1 + rnd(5))
    if paranoia > PARANOIA_SECOND_UPPER and paranoia <= PARANOIA_START then
      html = "<html><body>Doctor Chaos:<br>What?! Who are you? How did you come here?<br>You really look suspicious... Aren't those filthy members of Black Anvil guild send you? No? Mhhhhh... I don't trust you!</body></html>"
    elseif paranoia > PARANOIA_THIRD_UPPER and paranoia <= PARANOIA_SECOND_UPPER then
      html = "<html><body>Doctor Chaos:<br>Why are you standing here? Don't you see it's a private propertie? Don't look at him with those eyes... Did you smile?! Don't make fun of me! He will ... destroy ... you ... if you continue!</body></html>"
    elseif paranoia > 0 and paranoia <= PARANOIA_THIRD_UPPER then
      html = "<html><body>Doctor Chaos:<br>I know why you are here, traitor! He discovered your plans! You are assassin ... sent by the Black Anvil guild! But you won't kill the Emperor of Evil!</body></html>"
    elseif paranoia <= 0 then
      becomes_angry(npc)
    end
  end
  return html
end)

s:on_spawn(function(npc)
  paranoia = PARANOIA_START
  s:start_quest_timer("paranoia_activity", PARANOIA_TICK_MS, npc, nil, true)
  s:cancel_quest_timer("DISTANCE_CHECK", npc, nil)
  s:start_quest_timer("DISTANCE_CHECK", DISTANCE_CHECK_MS, npc, nil, true)
end)

s:on_kill(function(npc, killer, is_summon)
  s:cancel_quest_timer("golem_despawn", npc, nil)
  npc:broadcast_say(ChatType.NPC_GENERAL, "Urggh! You will pay dearly for this insult.")
  -- Mobius draws whole hours: 36 ± 24.
  local delay = (36 + rnd(-24, 24)) * boss.HOUR_MS
  grandboss.set_status(CHAOS_GOLEM, DEAD)
  s:start_quest_timer("reset_drchaos", delay, nil, nil, false)
  local info = grandboss.info(CHAOS_GOLEM) or { x = 0, y = 0, z = 0, heading = 0, hp = 0, mp = 0 }
  info.respawn_time = time.now_ms() + delay
  grandboss.set_info(CHAOS_GOLEM, info)
  s:cancel_quest_timers("DISTANCE_CHECK")
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  last_golem_attack = time.now_ms()
  local chance = rnd(300)
  if chance < 3 then
    local message
    if chance == 0 then
      message = "Bwah-ha-ha! Your doom is at hand! Behold the Ultra Secret Super Weapon!"
    elseif chance == 1 then
      message = "Foolish, insignificant creatures! How dare you challenge me!"
    else
      message = "I see that none will challenge me now!"
    end
    npc:broadcast_say(ChatType.NPC_GENERAL, message)
  end
end)
