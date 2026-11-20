-- Translated by questport from quests/Q00043_HelpTheSister (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(43, "Q00043_HelpTheSister", "Help the Sister!")

-- NPCs
local COOPER = 30829
local GALLADUCCI = 30097
-- Monsters
local SPECTER = 20171
local SORROW_MAIDEN = 20197
-- Items
local CRAFTED_DAGGER = 220
local MAP_PIECE = 7550
local MAP = 7551
local PET_TICKET = 7584

q:register_quest_items(MAP_PIECE, MAP)
q:add_start_npc(COOPER)
q:add_talk_id(COOPER, GALLADUCCI)
q:add_kill_id(SPECTER, SORROW_MAIDEN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30829-01.htm" then
    st:start_quest()
  elseif event == "30829-03.htm" then
    if q:has_quest_items(player, CRAFTED_DAGGER) then
      st:set_cond(2, true)
      q:take_items(player, CRAFTED_DAGGER, 1)
    end
  elseif event == "30829-05.htm" then
    st:set_cond(4, true)
    q:take_items(player, MAP_PIECE, 30)
    q:give_items(player, MAP, 1)
  elseif event == "30097-06.htm" then
    st:set_cond(5, true)
    q:take_items(player, MAP, 1)
  elseif event == "30829-07.htm" then
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
    htmltext = player:level() < 26 and "30829-00a.htm" or "30829-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == COOPER then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, CRAFTED_DAGGER) and "30829-01a.htm" or "30829-02.htm"
      elseif cond == 2 then
        htmltext = "30829-03a.htm"
      elseif cond == 3 then
        htmltext = "30829-04.htm"
      elseif cond == 4 then
        htmltext = "30829-05a.htm"
      elseif cond == 5 then
        htmltext = "30829-06.htm"
      end
    elseif sw2 == GALLADUCCI then
      if cond == 4 then
        htmltext = "30097-05.htm"
      elseif cond == 5 then
        htmltext = "30097-06a.htm"
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
