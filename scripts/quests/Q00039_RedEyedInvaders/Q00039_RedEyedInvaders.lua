-- Translated by questport from quests/Q00039_RedEyedInvaders (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(39, "Q00039_RedEyedInvaders", "Red-Eyed Invaders")

-- NPCs
local BABENCO = 30334
local BATHIS = 30332
-- Monsters
local MAILLE_LIZARDMAN = 20919
local MAILLE_LIZARDMAN_SCOUT = 20920
local MAILLE_LIZARDMAN_GUARD = 20921
local ARANEID = 20925
-- Items
local BLACK_BONE_NECKLACE = 7178
local RED_BONE_NECKLACE = 7179
local INCENSE_POUCH = 7180
local GEM_OF_MAILLE = 7181
-- First droplist
local FIRST_DP = {}
-- Second droplist
local SECOND_DP = {}
-- Rewards
local GREEN_COLORED_LURE_HG = 6521
local BABY_DUCK_RODE = 6529
local FISHING_SHOT_NG = 6535

FIRST_DP[MAILLE_LIZARDMAN_GUARD] = {RED_BONE_NECKLACE, BLACK_BONE_NECKLACE, 1000000}
FIRST_DP[MAILLE_LIZARDMAN] = {BLACK_BONE_NECKLACE, RED_BONE_NECKLACE, 1000000}
FIRST_DP[MAILLE_LIZARDMAN_SCOUT] = {BLACK_BONE_NECKLACE, RED_BONE_NECKLACE, 1000000}
SECOND_DP[ARANEID] = {GEM_OF_MAILLE, INCENSE_POUCH, 500000}
SECOND_DP[MAILLE_LIZARDMAN_GUARD] = {INCENSE_POUCH, GEM_OF_MAILLE, 300000}
SECOND_DP[MAILLE_LIZARDMAN_SCOUT] = {INCENSE_POUCH, GEM_OF_MAILLE, 250000}

q:register_quest_items(BLACK_BONE_NECKLACE, RED_BONE_NECKLACE, INCENSE_POUCH, GEM_OF_MAILLE)
q:add_start_npc(BABENCO)
q:add_talk_id(BABENCO, BATHIS)
q:add_kill_id(MAILLE_LIZARDMAN, MAILLE_LIZARDMAN_SCOUT, MAILLE_LIZARDMAN_GUARD, ARANEID)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30334-1.htm" then
    st:start_quest()
  elseif event == "30332-1.htm" then
    st:set_cond(2, true)
  elseif event == "30332-3.htm" then
    st:set_cond(4, true)
    q:take_items(player, BLACK_BONE_NECKLACE, -1)
    q:take_items(player, RED_BONE_NECKLACE, -1)
  elseif event == "30332-5.htm" then
    q:take_items(player, INCENSE_POUCH, -1)
    q:take_items(player, GEM_OF_MAILLE, -1)
    q:give_items(player, GREEN_COLORED_LURE_HG, 60)
    q:give_items(player, BABY_DUCK_RODE, 1)
    q:give_items(player, FISHING_SHOT_NG, 500)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "30334-2.htm" or "30334-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BABENCO then
      htmltext = "30334-3.htm"
    elseif sw2 == BATHIS then
      if cond == 1 then
        htmltext = "30332-0.htm"
      elseif cond == 2 then
        htmltext = "30332-2a.htm"
      elseif cond == 3 then
        htmltext = "30332-2.htm"
      elseif cond == 4 then
        htmltext = "30332-3a.htm"
      elseif cond == 5 then
        htmltext = "30332-4.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local npcId = npc:id()
  local qs = q:get_random_party_member_state(player, 2, 3, npc)
  if qs ~= nil and npcId ~= ARANEID then
    local list = FIRST_DP[npcId]
    if q:give_item_randomly(qs:get_player(), npc, list[1], 1, 1, 100, list[3] / 1e+06, true) and q:get_quest_items_count(qs:get_player(), list[2]) == 100 then
      qs:set_cond(3, true)
    end
  else
    qs = q:get_random_party_member_state(player, 4, 3, npc)
    if qs ~= nil and npcId ~= MAILLE_LIZARDMAN then
      local list = SECOND_DP[npcId]
      if q:give_item_randomly(qs:get_player(), npc, list[1], 1, 1, 30, list[3] / 1e+06, true) and q:get_quest_items_count(qs:get_player(), list[2]) == 30 then
        qs:set_cond(5, true)
      end
    end
  end
end)
