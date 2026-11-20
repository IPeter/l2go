-- Translated by questport from quests/Q00362_BardsMandolin (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(362, "Q00362_BardsMandolin", "Bard's Mandolin")

-- NPCs
local SWAN = 30957
local NANARIN = 30956
local GALION = 30958
local WOODROW = 30837
-- Items
local SWAN_FLUTE = 4316
local SWAN_LETTER = 4317

q:register_quest_items(SWAN_FLUTE, SWAN_LETTER)
q:add_start_npc(SWAN)
q:add_talk_id(SWAN, NANARIN, GALION, WOODROW)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30957-3.htm" then
    st:start_quest()
  elseif event == "30957-7.htm" or event == "30957-8.htm" then
    q:give_adena(player, 10000, true)
    q:give_items(player, 4410, 1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30957-2.htm" or "30957-1.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SWAN then
      if cond == 1 or cond == 2 then
        htmltext = "30957-4.htm"
      elseif cond == 3 then
        htmltext = "30957-5.htm"
        st:set_cond(4, true)
        q:give_items(player, SWAN_LETTER, 1)
      elseif cond == 4 then
        htmltext = "30957-5a.htm"
      elseif cond == 5 then
        htmltext = "30957-6.htm"
      end
    elseif sw2 == WOODROW then
      if cond == 1 then
        htmltext = "30837-1.htm"
        st:set_cond(2, true)
      elseif cond == 2 then
        htmltext = "30837-2.htm"
      elseif cond > 2 then
        htmltext = "30837-3.htm"
      end
    elseif sw2 == GALION then
      if cond == 2 then
        htmltext = "30958-1.htm"
        st:set_cond(3)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, SWAN_FLUTE, 1)
      elseif cond > 2 then
        htmltext = "30958-2.htm"
      end
    elseif sw2 == NANARIN then
      if cond == 4 then
        htmltext = "30956-1.htm"
        st:set_cond(5, true)
        q:take_items(player, SWAN_FLUTE, 1)
        q:take_items(player, SWAN_LETTER, 1)
      elseif cond == 5 then
        htmltext = "30956-2.htm"
      end
    end
  end
  return htmltext
end)
