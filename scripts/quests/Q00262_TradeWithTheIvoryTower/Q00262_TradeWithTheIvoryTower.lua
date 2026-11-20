-- Translated by questport from quests/Q00262_TradeWithTheIvoryTower (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(262, "Q00262_TradeWithTheIvoryTower", "Trade with the Ivory Tower")

-- Item
local FUNGUS_SAC = 707

q:register_quest_items(FUNGUS_SAC)
q:add_start_npc(30137) -- Vollodos
q:add_talk_id(30137)
q:add_kill_id(20400, 20007)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30137-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 8 and "30137-01.htm" or "30137-02.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, FUNGUS_SAC) < 10 then
      htmltext = "30137-04.htm"
    else
      htmltext = "30137-05.htm"
      q:take_items(player, FUNGUS_SAC, -1)
      q:give_adena(player, 3000, true)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(10) < (npc:id() == 20400 and 4 or 3) then
    q:give_items(player, FUNGUS_SAC, 1)
    if q:get_quest_items_count(player, FUNGUS_SAC) >= 10 then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
