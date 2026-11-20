-- Translated by questport from quests/Q00359_ForASleeplessDeadman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(359, "Q00359_ForASleeplessDeadman", "For Sleepless Deadmen")

-- Monsters
local DOOM_SERVANT = 21006
local DOOM_GUARD = 21007
local DOOM_ARCHER = 21008
-- Item
local REMAINS = 5869
-- Reward
local REWARD = {6341, 6342, 6343, 6344, 6345, 6346, 5494, 5495}
-- Drop chances
local CHANCES = {}

CHANCES[DOOM_SERVANT] = 320000
CHANCES[DOOM_GUARD] = 340000
CHANCES[DOOM_ARCHER] = 420000

q:register_quest_items(REMAINS)
q:add_start_npc(30857) -- Orven
q:add_talk_id(30857)
q:add_kill_id(DOOM_SERVANT, DOOM_GUARD, DOOM_ARCHER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30857-06.htm" then
    st:start_quest()
  elseif event == "30857-10.htm" then
    q:give_items(player, rnd.entry(REWARD), 4)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "30857-01.htm" or "30857-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30857-07.htm"
    elseif cond == 2 then
      htmltext = "30857-08.htm"
      st:set_cond(3, true)
      q:take_items(player, REMAINS, -1)
    elseif cond == 3 then
      htmltext = "30857-09.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, REMAINS, 1)
    if q:get_quest_items_count(player, REMAINS) < 60 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
