-- Translated by questport from quests/Q00341_HuntingForWildBeasts (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(341, "Q00341_HuntingForWildBeasts", "Hunting for Wild Beasts")

-- Item
local BEAR_SKIN = 4259
-- Drop chances
local CHANCES = {}

CHANCES[20021] = 500000 -- Red Bear
CHANCES[20203] = 900000 -- Dion Grizzly
CHANCES[20310] = 500000 -- Brown Bear
CHANCES[20335] = 700000 -- Grizzly Bear

q:register_quest_items(BEAR_SKIN)
q:add_start_npc(30078) -- Pano
q:add_talk_id(30078)
q:add_kill_id(20021, 20203, 20310, 20335)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30078-02.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "30078-00.htm" or "30078-01.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, BEAR_SKIN) < 20 then
      htmltext = "30078-03.htm"
    else
      htmltext = "30078-04.htm"
      q:take_items(player, BEAR_SKIN, -1)
      q:give_adena(player, 3710, true)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if q:get_quest_items_count(player, BEAR_SKIN) < 20 and rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, BEAR_SKIN, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
