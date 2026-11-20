-- Translated by questport from quests/Q00153_DeliverGoods (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(153, "Q00153_DeliverGoods", "Deliver Goods")

-- NPCs
local JACKSON = 30002
local SILVIA = 30003
local ARNOLD = 30041
local RANT = 30054
-- Items
local DELIVERY_LIST = 1012
local HEAVY_WOOD_BOX = 1013
local CLOTH_BUNDLE = 1014
local CLAY_POT = 1015
local JACKSON_RECEIPT = 1016
local SILVIA_RECEIPT = 1017
local RANT_RECEIPT = 1018
-- Rewards
local SOULSHOT_NO_GRADE = 1835
local RING_OF_KNOWLEDGE = 875

q:register_quest_items(DELIVERY_LIST, HEAVY_WOOD_BOX, CLOTH_BUNDLE, CLAY_POT, JACKSON_RECEIPT, SILVIA_RECEIPT, RANT_RECEIPT)
q:add_start_npc(ARNOLD)
q:add_talk_id(JACKSON, SILVIA, ARNOLD, RANT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30041-02.htm" then
    st:start_quest()
    q:give_items(player, DELIVERY_LIST, 1)
    q:give_items(player, CLAY_POT, 1)
    q:give_items(player, CLOTH_BUNDLE, 1)
    q:give_items(player, HEAVY_WOOD_BOX, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 2 and "30041-00.htm" or "30041-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == ARNOLD then
      if st:is_cond(1) then
        htmltext = "30041-03.htm"
      elseif st:is_cond(2) then
        htmltext = "30041-04.htm"
        q:take_items(player, DELIVERY_LIST, 1)
        q:take_items(player, JACKSON_RECEIPT, 1)
        q:take_items(player, SILVIA_RECEIPT, 1)
        q:take_items(player, RANT_RECEIPT, 1)
        q:give_items(player, RING_OF_KNOWLEDGE, 1)
        q:give_items(player, RING_OF_KNOWLEDGE, 1)
        q:add_exp_and_sp(player, 600, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == JACKSON then
      if q:has_quest_items(player, HEAVY_WOOD_BOX) then
        htmltext = "30002-01.htm"
        q:take_items(player, HEAVY_WOOD_BOX, 1)
        q:give_items(player, JACKSON_RECEIPT, 1)
        if q:has_quest_items(player, SILVIA_RECEIPT, RANT_RECEIPT) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30002-02.htm"
      end
    elseif sw2 == SILVIA then
      if q:has_quest_items(player, CLOTH_BUNDLE) then
        htmltext = "30003-01.htm"
        q:take_items(player, CLOTH_BUNDLE, 1)
        q:give_items(player, SILVIA_RECEIPT, 1)
        q:give_items(player, SOULSHOT_NO_GRADE, 3)
        if q:has_quest_items(player, JACKSON_RECEIPT, RANT_RECEIPT) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30003-02.htm"
      end
    elseif sw2 == RANT then
      if q:has_quest_items(player, CLAY_POT) then
        htmltext = "30054-01.htm"
        q:take_items(player, CLAY_POT, 1)
        q:give_items(player, RANT_RECEIPT, 1)
        if q:has_quest_items(player, JACKSON_RECEIPT, SILVIA_RECEIPT) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30054-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
