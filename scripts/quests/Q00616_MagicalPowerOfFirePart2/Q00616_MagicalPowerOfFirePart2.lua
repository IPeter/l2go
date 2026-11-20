-- Translated by questport from quests/Q00616_MagicalPowerOfFirePart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Magical Power of Fire - Part 2 (616)
-- @author Joxit
local J = require("lib.java")

local q = quest.new(616, "Q00616_MagicalPowerOfFirePart2", "Magical Power of Fire - Part 2")

-- NPCs
local UDAN = 31379
local KETRA_TOTEM = 31558
-- Monster
local NASTRON = 25306
-- Items
local RED_TOTEM = 7243
local NASTRON_HEART = 7244
-- Misc
local MIN_LEVEL = 75

local function spawnNastron(npc, qs)
  if q:get_quest_timer("spawn_npc", nil, nil) ~= nil then
    return "31558-03.html"
  end
  if qs:is_cond(1) then
    q:take_items(qs:get_player(), RED_TOTEM, 1)
    qs:set_cond(2, true)
  end
  npc:delete()
  local nastron = q:add_spawn(NASTRON, {x = 142528, y = -82528, z = -6496, heading = 0}, false, 0)
  nastron:broadcast_say(ChatType.NPC_GENERAL, "The magical power of fire is also the power of flames and lava! If you dare to confront it, only death will await you!")
  q:start_quest_timer("despawn_nastron", 1200000, nastron, nil)
  return "31558-02.html"
end

q:add_start_npc(UDAN)
q:add_talk_id(UDAN, KETRA_TOTEM)
q:add_kill_id(NASTRON)
q:register_quest_items(RED_TOTEM, NASTRON_HEART)

-- Mobius does this in the constructor: the totem, or the timer that brings
-- it back after Nastron's respawn delay.
q:on_start(function()
  local test = globals.get_long("Q00616_respawn", 0)
  local remain = test ~= 0 and test - time.now_ms() or 0
  if remain > 0 then
    q:start_quest_timer("spawn_npc", remain, nil, nil)
  else
    q:add_spawn(KETRA_TOTEM, {x = 142368, y = -82512, z = -6487, heading = 58000}, false, 0, true)
  end
end)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    if npc:id() == NASTRON then
      -- Mobius: case 1 takes the totem and falls through into case 2.
      local sw = qs:get_cond()
      if sw == 1 then
        q:take_items(player, RED_TOTEM, 1)
        if not q:has_quest_items(player, NASTRON_HEART) then
          q:give_items(player, NASTRON_HEART, 1)
        end
        qs:set_cond(3, true)
      elseif sw == 2 then
        if not q:has_quest_items(player, NASTRON_HEART) then
          q:give_items(player, NASTRON_HEART, 1)
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
    if event == "31379-02.html" then
      qs:start_quest()
      htmltext = event
    elseif event == "give_heart" then
      if q:has_quest_items(player, NASTRON_HEART) then
        q:add_exp_and_sp(player, 10000, 0)
        qs:exit_quest(true, true)
        htmltext = "31379-06.html"
      else
        htmltext = "31379-07.html"
      end
    elseif event == "spawn_totem" then
      if q:has_quest_items(player, RED_TOTEM) then
        htmltext = spawnNastron(npc, qs)
      else
        htmltext = "31558-04.html"
      end
    end
  else
    if event == "despawn_nastron" then
      npc:broadcast_say(ChatType.NPC_GENERAL, "The power of constraint is getting weaker. Your ritual has failed!")
      npc:delete()
      q:add_spawn(KETRA_TOTEM, {x = 142368, y = -82512, z = -6487, heading = 58000}, false, 0, true)
    elseif event == "spawn_npc" then
      q:add_spawn(KETRA_TOTEM, {x = 142368, y = -82512, z = -6487, heading = 58000}, false, 0, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local respawnMinDelay = J.int(43200000 * Config.RAID_MIN_RESPAWN_MULTIPLIER)
  local respawnMaxDelay = J.int(129600000 * Config.RAID_MAX_RESPAWN_MULTIPLIER)
  local respawnDelay = rnd(respawnMinDelay, respawnMaxDelay)
  q:cancel_quest_timer("despawn_nastron", npc, nil)
  globals.set("Q00616_respawn", tostring(time.now_ms() + respawnDelay))
  q:start_quest_timer("spawn_npc", respawnDelay, nil, nil)
  q:execute_for_each_player(killer, npc, isSummon, true, false)
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == UDAN then
    local sw2 = qs:get_state()
    if sw2 == State.CREATED then
      htmltext = player:level() >= MIN_LEVEL and (q:has_quest_items(player, RED_TOTEM) and "31379-01.htm" or "31379-00a.html") or "31379-00b.html"
    elseif sw2 == State.STARTED then
      htmltext = qs:is_cond(1) and "31379-03.html" or q:has_quest_items(player, NASTRON_HEART) and "31379-04.html" or "31379-05.html"
    end
  elseif sw == KETRA_TOTEM then
    if qs:is_started() then
      local sw3 = qs:get_cond()
      if sw3 == 1 then
        htmltext = "31558-01.html"
      elseif sw3 == 2 then
        htmltext = spawnNastron(npc, qs)
      elseif sw3 == 3 then
        htmltext = "31558-05.html"
      end
    end
  end
  return htmltext
end)
