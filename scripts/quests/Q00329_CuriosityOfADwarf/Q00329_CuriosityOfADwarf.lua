-- Translated by questport from quests/Q00329_CuriosityOfADwarf (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(329, "Q00329_CuriosityOfADwarf", "Curiosity of a Dwarf")

-- Items
local GOLEM_HEARTSTONE = 1346
local BROKEN_HEARTSTONE = 1365

q:add_start_npc(30437) -- Rolento
q:add_talk_id(30437)
q:add_kill_id(20083, 20085) -- Granite golem, Puncher

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30437-03.htm" then
    st:start_quest()
  elseif event == "30437-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 33 and "30437-01.htm" or "30437-02.htm"
  elseif sw == State.STARTED then
    local golem = q:get_quest_items_count(player, GOLEM_HEARTSTONE)
    local broken = q:get_quest_items_count(player, BROKEN_HEARTSTONE)
    if golem + broken == 0 then
      htmltext = "30437-04.htm"
    else
      htmltext = "30437-05.htm"
      q:take_items(player, GOLEM_HEARTSTONE, -1)
      q:take_items(player, BROKEN_HEARTSTONE, -1)
      q:give_adena(player, broken * 50 + golem * 1000 + (golem + broken > 10 and 1183 or 0), true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local chance = rnd(100)
  if chance < 2 then
    q:give_items(player, GOLEM_HEARTSTONE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif chance < (npc:id() == 20083 and 44 or 50) then
    q:give_items(player, BROKEN_HEARTSTONE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
