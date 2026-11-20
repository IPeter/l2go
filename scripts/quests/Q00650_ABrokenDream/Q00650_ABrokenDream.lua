-- Translated by questport from quests/Q00650_ABrokenDream (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(650, "Q00650_ABrokenDream", "A Broken Dream")

-- NPC
local GHOST = 32054
-- Monsters
local CREWMAN = 22027
local VAGABOND = 22028
-- Item
local DREAM_FRAGMENT = 8514

q:register_quest_items(DREAM_FRAGMENT)
q:add_start_npc(GHOST)
q:add_talk_id(GHOST)
q:add_kill_id(CREWMAN, VAGABOND)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32054-01a.htm" then
    st:start_quest()
  elseif event == "32054-03.htm" then
    if not q:has_quest_items(player, DREAM_FRAGMENT) then
      htmltext = "32054-04.htm"
    end
  elseif event == "32054-05.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local st2 = player:get_quest_state("Q00117_TheOceanOfDistantStars")
    if st2 ~= nil and st2:is_completed() and player:level() >= 39 then
      htmltext = "32054-01.htm"
    else
      htmltext = "32054-00.htm"
      st:exit_quest(true)
    end
  elseif sw == State.STARTED then
    htmltext = "32054-02.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(100) < 25 then
    q:give_items(player, DREAM_FRAGMENT, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
