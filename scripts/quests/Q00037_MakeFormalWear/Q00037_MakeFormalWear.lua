-- Translated by questport from quests/Q00037_MakeFormalWear (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(37, "Q00037_MakeFormalWear", "Make Formal Wear")

-- NPCs
local ALEXIS = 30842
local LEIKAR = 31520
local JEREMY = 31521
local MIST = 31627
-- Items
local MYSTERIOUS_CLOTH = 7076
local JEWEL_BOX = 7077
local SEWING_KIT = 7078
local DRESS_SHOES_BOX = 7113
local SIGNET_RING = 7164
local ICE_WINE = 7160
local BOX_OF_COOKIES = 7159
-- Reward
local FORMAL_WEAR = 6408

q:register_quest_items(SIGNET_RING, ICE_WINE, BOX_OF_COOKIES)
q:add_start_npc(ALEXIS)
q:add_talk_id(ALEXIS, LEIKAR, JEREMY, MIST)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30842-1.htm" then
    st:start_quest()
  elseif event == "31520-1.htm" then
    st:set_cond(2, true)
    q:give_items(player, SIGNET_RING, 1)
  elseif event == "31521-1.htm" then
    st:set_cond(3, true)
    q:take_items(player, SIGNET_RING, 1)
    q:give_items(player, ICE_WINE, 1)
  elseif event == "31627-1.htm" then
    st:set_cond(4, true)
    q:take_items(player, ICE_WINE, 1)
  elseif event == "31521-3.htm" then
    st:set_cond(5, true)
    q:give_items(player, BOX_OF_COOKIES, 1)
  elseif event == "31520-3.htm" then
    st:set_cond(6, true)
    q:take_items(player, BOX_OF_COOKIES, 1)
  elseif event == "31520-5.htm" then
    st:set_cond(7, true)
    q:take_items(player, JEWEL_BOX, 1)
    q:take_items(player, MYSTERIOUS_CLOTH, 1)
    q:take_items(player, SEWING_KIT, 1)
  elseif event == "31520-7.htm" then
    q:take_items(player, DRESS_SHOES_BOX, 1)
    q:give_items(player, FORMAL_WEAR, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "30842-0a.htm" or "30842-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ALEXIS then
      if cond == 1 then
        htmltext = "30842-2.htm"
      end
    elseif sw2 == LEIKAR then
      if cond == 1 then
        htmltext = "31520-0.htm"
      elseif cond == 2 then
        htmltext = "31520-1a.htm"
      elseif cond == 5 or cond == 6 then
        if q:has_quest_items(player, MYSTERIOUS_CLOTH, JEWEL_BOX, SEWING_KIT) then
          htmltext = "31520-4.htm"
        elseif q:has_quest_items(player, BOX_OF_COOKIES) then
          htmltext = "31520-2.htm"
        else
          htmltext = "31520-3a.htm"
        end
      elseif cond == 7 then
        htmltext = q:has_quest_items(player, DRESS_SHOES_BOX) and "31520-6.htm" or "31520-5a.htm"
      end
    elseif sw2 == JEREMY then
      if q:has_quest_items(player, SIGNET_RING) then
        htmltext = "31521-0.htm"
      elseif cond == 3 then
        htmltext = "31521-1a.htm"
      elseif cond == 4 then
        htmltext = "31521-2.htm"
      elseif cond > 4 then
        htmltext = "31521-3a.htm"
      end
    elseif sw2 == MIST then
      if cond == 3 then
        htmltext = "31627-0.htm"
      elseif cond > 3 then
        htmltext = "31627-2.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
