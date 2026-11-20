-- Translated by questport from quests/Q00044_HelpTheSon (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(44, "Q00044_HelpTheSon", "Help the Son!")

-- NPCs
local LUNDY = 30827
local DRIKUS = 30505
-- Monsters
local MAILLE = 20919
local MAILLE_SCOUT = 20920
local MAILLE_GUARD = 20921
-- Items
local WORK_HAMMER = 168
local GEMSTONE_FRAGMENT = 7552
local GEMSTONE = 7553
local PET_TICKET = 7585

q:register_quest_items(GEMSTONE_FRAGMENT, GEMSTONE)
q:add_start_npc(LUNDY)
q:add_talk_id(LUNDY, DRIKUS)
q:add_kill_id(MAILLE, MAILLE_SCOUT, MAILLE_GUARD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30827-01.htm" then
    st:start_quest()
  elseif event == "30827-03.htm" then
    if q:has_quest_items(player, WORK_HAMMER) then
      st:set_cond(2, true)
      q:take_items(player, WORK_HAMMER, 1)
    end
  elseif event == "30827-05.htm" then
    st:set_cond(4, true)
    q:take_items(player, GEMSTONE_FRAGMENT, 30)
    q:give_items(player, GEMSTONE, 1)
  elseif event == "30505-06.htm" then
    st:set_cond(5, true)
    q:take_items(player, GEMSTONE, 1)
  elseif event == "30827-07.htm" then
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
    htmltext = player:level() < 24 and "30827-00a.htm" or "30827-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LUNDY then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, WORK_HAMMER) and "30827-01a.htm" or "30827-02.htm"
      elseif cond == 2 then
        htmltext = "30827-03a.htm"
      elseif cond == 3 then
        htmltext = "30827-04.htm"
      elseif cond == 4 then
        htmltext = "30827-05a.htm"
      elseif cond == 5 then
        htmltext = "30827-06.htm"
      end
    elseif sw2 == DRIKUS then
      if cond == 4 then
        htmltext = "30505-05.htm"
      elseif cond == 5 then
        htmltext = "30505-06a.htm"
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
    q:give_items(player, GEMSTONE_FRAGMENT, 1)
    if q:get_quest_items_count(player, GEMSTONE_FRAGMENT) == 30 then
      qs:set_cond(3, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
