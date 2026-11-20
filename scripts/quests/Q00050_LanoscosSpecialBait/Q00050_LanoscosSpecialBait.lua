-- Translated by questport from quests/Q00050_LanoscosSpecialBait (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(50, "Q00050_LanoscosSpecialBait", "Lanosco's Special Bait")
-- The reward is a fishing lure: offered once fishing exists (Plan 11).
q:requires("fishing")

-- Item
local ESSENCE_OF_WIND = 7621
-- Reward
local WIND_FISHING_LURE = 7610

q:register_quest_items(ESSENCE_OF_WIND)
q:add_start_npc(31570) -- Lanosco
q:add_talk_id(31570)
q:add_kill_id(21026) -- Singing wind

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31570-03.htm" then
    st:start_quest()
  elseif event == "31570-07.htm" then
    htmltext = "31570-06.htm"
    q:take_items(player, ESSENCE_OF_WIND, -1)
    q:reward_items(player, WIND_FISHING_LURE, 4)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 27 and "31570-02.htm" or "31570-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, ESSENCE_OF_WIND) == 100 and "31570-04.htm" or "31570-05.htm"
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
  if q:get_quest_items_count(partyMember, ESSENCE_OF_WIND) < 100 then
    if rnd(100) < 33 then
      q:give_items(partyMember, ESSENCE_OF_WIND, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
  if q:get_quest_items_count(partyMember, ESSENCE_OF_WIND) >= 100 then
    qs:set_cond(2, true)
  end
end)
