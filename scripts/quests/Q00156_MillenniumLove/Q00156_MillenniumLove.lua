-- Translated by questport from quests/Q00156_MillenniumLove (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(156, "Q00156_MillenniumLove", "Millennium Love")

-- NPCs
local LILITH = 30368
local BAENEDES = 30369
-- Items
local LILITH_LETTER = 1022
local THEON_DIARY = 1023

q:register_quest_items(LILITH_LETTER, THEON_DIARY)
q:add_start_npc(LILITH)
q:add_talk_id(LILITH, BAENEDES)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30368-04.htm" then
    st:start_quest()
    q:give_items(player, LILITH_LETTER, 1)
  elseif event == "30369-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, LILITH_LETTER, 1)
    q:give_items(player, THEON_DIARY, 1)
  elseif event == "30369-03.htm" then
    q:take_items(player, LILITH_LETTER, 1)
    q:add_exp_and_sp(player, 3000, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30368-00.htm" or "30368-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == LILITH then
      if q:has_quest_items(player, LILITH_LETTER) then
        htmltext = "30368-05.htm"
      elseif q:has_quest_items(player, THEON_DIARY) then
        htmltext = "30368-06.htm"
        q:take_items(player, THEON_DIARY, 1)
        q:give_items(player, 5250, 1)
        q:add_exp_and_sp(player, 3000, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == BAENEDES then
      if q:has_quest_items(player, LILITH_LETTER) then
        htmltext = "30369-01.htm"
      elseif q:has_quest_items(player, THEON_DIARY) then
        htmltext = "30369-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
