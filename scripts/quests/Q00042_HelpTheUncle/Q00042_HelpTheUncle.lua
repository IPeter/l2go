-- Translated by questport from quests/Q00042_HelpTheUncle (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(42, "Q00042_HelpTheUncle", "Help the Uncle!")

-- NPCs
local WATERS = 30828
local SOPHYA = 30735
-- Monsters
local MONSTER_EYE_DESTROYER = 20068
local MONSTER_EYE_GAZER = 20266
-- Items
local TRIDENT = 291
local MAP_PIECE = 7548
local MAP = 7549
local PET_TICKET = 7583

q:register_quest_items(MAP_PIECE, MAP)
q:add_start_npc(WATERS)
q:add_talk_id(WATERS, SOPHYA)
q:add_kill_id(MONSTER_EYE_DESTROYER, MONSTER_EYE_GAZER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30828-01.htm" then
    st:start_quest()
  elseif event == "30828-03.htm" then
    if q:has_quest_items(player, TRIDENT) then
      st:set_cond(2, true)
      q:take_items(player, TRIDENT, 1)
    end
  elseif event == "30828-05.htm" then
    st:set_cond(4, true)
    q:take_items(player, MAP_PIECE, 30)
    q:give_items(player, MAP, 1)
  elseif event == "30735-06.htm" then
    st:set_cond(5, true)
    q:take_items(player, MAP, 1)
  elseif event == "30828-07.htm" then
    q:give_items(player, PET_TICKET, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 25 and "30828-00a.htm" or "30828-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == WATERS then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, TRIDENT) and "30828-01a.htm" or "30828-02.htm"
      elseif cond == 2 then
        htmltext = "30828-03a.htm"
      elseif cond == 3 then
        htmltext = "30828-04.htm"
      elseif cond == 4 then
        htmltext = "30828-05a.htm"
      elseif cond == 5 then
        htmltext = "30828-06.htm"
      end
    elseif sw2 == SOPHYA then
      if cond == 4 then
        htmltext = "30735-05.htm"
      elseif cond == 5 then
        htmltext = "30735-06a.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(2) then
    q:give_items(player, MAP_PIECE, 1)
    if q:get_quest_items_count(player, MAP_PIECE) == 30 then
      qs:set_cond(3, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
