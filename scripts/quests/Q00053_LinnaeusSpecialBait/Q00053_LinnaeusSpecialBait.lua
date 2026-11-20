-- Translated by questport from quests/Q00053_LinnaeusSpecialBait (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(53, "Q00053_LinnaeusSpecialBait", "Linnaeus' Special Bait")
-- The reward is a fishing lure: offered once fishing exists (Plan 11).
q:requires("fishing")

-- Item
local CRIMSON_DRAKE_HEART = 7624
-- Reward
local FLAMING_FISHING_LURE = 7613

q:register_quest_items(CRIMSON_DRAKE_HEART)
q:add_start_npc(31577) -- Linnaeus
q:add_talk_id(31577)
q:add_kill_id(20670) -- Crimson Drake

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31577-03.htm" then
    st:start_quest()
  elseif event == "31577-07.htm" then
    htmltext = "31577-06.htm"
    q:take_items(player, CRIMSON_DRAKE_HEART, -1)
    q:reward_items(player, FLAMING_FISHING_LURE, 4)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "31577-02.htm" or "31577-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, CRIMSON_DRAKE_HEART) == 100 and "31577-04.htm" or "31577-05.htm"
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local partyMember = q:get_random_party_member(player, 1)
  if partyMember == nil then
    return
  end
  local qs = q:get_quest_state(partyMember, false)
  if q:get_quest_items_count(partyMember, CRIMSON_DRAKE_HEART) < 100 then
    if rnd(100) < 33 then
      q:give_items(partyMember, CRIMSON_DRAKE_HEART, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
  if q:get_quest_items_count(partyMember, CRIMSON_DRAKE_HEART) >= 100 then
    qs:set_cond(2, true)
  end
end)
