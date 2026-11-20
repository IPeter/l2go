-- Translated by questport from quests/Q00020_BringUpWithLove (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Cond 2 and the Jewel of Innocence come from Mobius's FeedableBeasts AI
-- (taming a beast at the Beast Farm), which L2Go does not have yet.
local J = require("lib.java")

local q = quest.new(20, "Q00020_BringUpWithLove", "Bring Up With Love")

-- Item
local JEWEL_OF_INNOCENCE = 7185

q:register_quest_items(JEWEL_OF_INNOCENCE)
q:add_start_npc(31537) -- Tunatun
q:add_talk_id(31537)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31537-09.htm" then
    st:start_quest()
  elseif event == "31537-12.htm" then
    q:take_items(player, JEWEL_OF_INNOCENCE, -1)
    q:give_adena(player, 68500, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 65 and "31537-02.htm" or "31537-01.htm"
  elseif sw == State.STARTED then
    if st:is_cond(2) then
      htmltext = "31537-11.htm"
    else
      htmltext = "31537-10.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
