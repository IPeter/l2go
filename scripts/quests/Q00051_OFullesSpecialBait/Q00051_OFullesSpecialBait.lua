-- Translated by questport from quests/Q00051_OFullesSpecialBait (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(51, "Q00051_OFullesSpecialBait", "O'Fulle's Special Bait")
-- The reward is a fishing lure: offered once fishing exists (Plan 11).
q:requires("fishing")

-- Item
local LOST_BAIT = 7622
-- Reward
local ICY_AIR_LURE = 7611

q:register_quest_items(LOST_BAIT)
q:add_start_npc(31572) -- O'Fulle
q:add_talk_id(31572)
q:add_kill_id(20552) -- Fettered Soul

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31572-03.htm" then
    st:start_quest()
  elseif event == "31572-07.htm" then
    htmltext = "31572-06.htm"
    q:take_items(player, LOST_BAIT, -1)
    q:reward_items(player, ICY_AIR_LURE, 4)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 36 and "31572-02.htm" or "31572-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, LOST_BAIT) == 100 and "31572-04.htm" or "31572-05.htm"
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
  if q:get_quest_items_count(partyMember, LOST_BAIT) < 100 then
    if rnd(100) < 33 then
      q:give_items(partyMember, LOST_BAIT, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
  if q:get_quest_items_count(partyMember, LOST_BAIT) >= 100 then
    qs:set_cond(2, true)
  end
end)
