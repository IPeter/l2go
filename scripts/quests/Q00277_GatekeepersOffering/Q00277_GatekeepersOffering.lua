-- Translated by questport from quests/Q00277_GatekeepersOffering (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(277, "Q00277_GatekeepersOffering", "Gatekeeper's Offering")

-- Item
local STARSTONE = 1572
-- Reward
local GATEKEEPER_CHARM = 1658

q:add_start_npc(30576) -- Tamil
q:add_talk_id(30576)
q:add_kill_id(20333) -- Graystone Golem

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30576-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30576-01.htm" or "30576-02.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30576-04.htm"
    else
      htmltext = "30576-05.htm"
      q:take_items(player, STARSTONE, -1)
      q:reward_items(player, GATEKEEPER_CHARM, 2)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd.bool() then
    q:give_items(player, STARSTONE, 1)
    if q:get_quest_items_count(player, STARSTONE) < 20 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
