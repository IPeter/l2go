-- Translated by questport from quests/Q00610_MagicalPowerOfWaterPart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Magical Power of Water - Part 2 (610)
-- @author Joxit
local J = require("lib.java")

local q = quest.new(610, "Q00610_MagicalPowerOfWaterPart2", "Magical Power of Water - Part 2")

-- NPCs
local ASEFA = 31372
local VARKA_TOTEM = 31560
-- Monster
local ASHUTAR = 25316
-- Items
local GREEN_TOTEM = 7238
local ASHUTAR_HEART = 7239
-- Misc
local MIN_LEVEL = 75

local function spawnAshutar(npc, qs)
  if q:get_quest_timer("spawn_npc", nil, nil) ~= nil then
    return "31560-03.html"
  end
  if qs:is_cond(1) then
    q:take_items(qs:get_player(), GREEN_TOTEM, 1)
    qs:set_cond(2, true)
  end
  npc:delete()
  local ashutar = q:add_spawn(ASHUTAR, {x = 104825, y = -36926, z = -1136, heading = 0}, false, 0)
  ashutar:broadcast_say(ChatType.NPC_GENERAL, "The magical power of water comes from the power of storm and hail! If you dare to confront it, only death will await you!")
  q:start_quest_timer("despawn_ashutar", 1200000, ashutar, nil)
  return "31560-02.html"
end

q:add_start_npc(ASEFA)
q:add_talk_id(ASEFA, VARKA_TOTEM)
q:add_kill_id(ASHUTAR)
q:register_quest_items(GREEN_TOTEM, ASHUTAR_HEART)

-- Mobius does this in the constructor: the totem, or the timer that brings
-- it back after Ashutar's respawn delay.
q:on_start(function()
  local test = globals.get_long("Q00610_respawn", 0)
  local remain = test ~= 0 and test - time.now_ms() or 0
  if remain > 0 then
    q:start_quest_timer("spawn_npc", remain, nil, nil)
  else
    q:add_spawn(VARKA_TOTEM, {x = 105452, y = -36775, z = -1050, heading = 34000}, false, 0, true)
  end
end)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    if npc:id() == ASHUTAR then
      -- Mobius: case 1 takes the totem and falls through into case 2.
      local sw = qs:get_cond()
      if sw == 1 then
        q:take_items(player, GREEN_TOTEM, 1)
        if not q:has_quest_items(player, ASHUTAR_HEART) then
          q:give_items(player, ASHUTAR_HEART, 1)
        end
        qs:set_cond(3, true)
      elseif sw == 2 then
        if not q:has_quest_items(player, ASHUTAR_HEART) then
          q:give_items(player, ASHUTAR_HEART, 1)
        end
        qs:set_cond(3, true)
      end
    end
  end
end)

q:on_event(function(event, npc, player)
  local htmltext = nil
  if player ~= nil then
    local qs = q:get_quest_state(player, false)
    if qs == nil then
      return nil
    end
    if event == "31372-02.html" then
      qs:start_quest()
      htmltext = event
    elseif event == "give_heart" then
      if q:has_quest_items(player, ASHUTAR_HEART) then
        q:add_exp_and_sp(player, 10000, 0)
        qs:exit_quest(true, true)
        htmltext = "31372-06.html"
      else
        htmltext = "31372-07.html"
      end
    elseif event == "spawn_totem" then
      if q:has_quest_items(player, GREEN_TOTEM) then
        htmltext = spawnAshutar(npc, qs)
      else
        htmltext = "31560-04.html"
      end
    end
  else
    if event == "despawn_ashutar" then
      npc:broadcast_say(ChatType.NPC_GENERAL, "The power of constraint is getting weaker. Your ritual has failed!")
      npc:delete()
      q:add_spawn(VARKA_TOTEM, {x = 105452, y = -36775, z = -1050, heading = 34000}, false, 0, true)
    elseif event == "spawn_npc" then
      q:add_spawn(VARKA_TOTEM, {x = 105452, y = -36775, z = -1050, heading = 34000}, false, 0, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local respawnMinDelay = J.int(43200000 * Config.RAID_MIN_RESPAWN_MULTIPLIER)
  local respawnMaxDelay = J.int(129600000 * Config.RAID_MAX_RESPAWN_MULTIPLIER)
  local respawnDelay = rnd(respawnMinDelay, respawnMaxDelay)
  q:cancel_quest_timer("despawn_ashutar", npc, nil)
  globals.set("Q00610_respawn", tostring(time.now_ms() + respawnDelay))
  q:start_quest_timer("spawn_npc", respawnDelay, nil, nil)
  q:execute_for_each_player(killer, npc, isSummon, true, false)
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == ASEFA then
    local sw2 = qs:get_state()
    if sw2 == State.CREATED then
      htmltext = player:level() >= MIN_LEVEL and (q:has_quest_items(player, GREEN_TOTEM) and "31372-01.htm" or "31372-00a.html") or "31372-00b.html"
    elseif sw2 == State.STARTED then
      htmltext = qs:is_cond(1) and "31372-03.html" or q:has_quest_items(player, ASHUTAR_HEART) and "31372-04.html" or "31372-05.html"
    end
  elseif sw == VARKA_TOTEM then
    if qs:is_started() then
      local sw3 = qs:get_cond()
      if sw3 == 1 then
        htmltext = "31560-01.html"
      elseif sw3 == 2 then
        htmltext = spawnAshutar(npc, qs)
      elseif sw3 == 3 then
        htmltext = "31560-05.html"
      end
    end
  end
  return htmltext
end)
