-- Translated by questport from quests/Q00007_ATripBegins (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(7, "Q00007_ATripBegins", "A Trip Begins")

-- NPCs
local MIRABEL = 30146
local ARIEL = 30148
local ASTERIOS = 30154
-- Items
local ARIEL_RECO = 7572
-- Rewards
local MARK_TRAVELER = 7570
local SOE_GIRAN = 7559

q:register_quest_items(ARIEL_RECO)
q:add_start_npc(MIRABEL)
q:add_talk_id(MIRABEL, ARIEL, ASTERIOS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30146-03.htm" then
    st:start_quest()
  elseif event == "30148-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, ARIEL_RECO, 1)
  elseif event == "30154-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, ARIEL_RECO, 1)
  elseif event == "30146-06.htm" then
    q:give_items(player, MARK_TRAVELER, 1)
    q:reward_items(player, SOE_GIRAN, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30146-01.htm"
    elseif player:level() < 3 then
      htmltext = "30146-01a.htm"
    else
      htmltext = "30146-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MIRABEL then
      if cond == 1 or cond == 2 then
        htmltext = "30146-04.htm"
      elseif cond == 3 then
        htmltext = "30146-05.htm"
      end
    elseif sw2 == ARIEL then
      if cond == 1 then
        htmltext = "30148-01.htm"
      elseif cond == 2 then
        htmltext = "30148-03.htm"
      end
    elseif sw2 == ASTERIOS then
      if cond == 2 then
        htmltext = "30154-01.htm"
      elseif cond == 3 then
        htmltext = "30154-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
