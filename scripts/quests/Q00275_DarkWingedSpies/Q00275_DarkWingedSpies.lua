-- Translated by questport from quests/Q00275_DarkWingedSpies (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(275, "Q00275_DarkWingedSpies", "Dark Winged Spies")

-- Monsters
local DARKWING_BAT = 20316
local VARANGKA_TRACKER = 27043
-- Items
local DARKWING_BAT_FANG = 1478
local VARANGKA_PARASITE = 1479

q:register_quest_items(DARKWING_BAT_FANG, VARANGKA_PARASITE)
q:add_start_npc(30567) -- Tantus
q:add_talk_id(30567)
q:add_kill_id(DARKWING_BAT, VARANGKA_TRACKER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30567-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30567-00.htm"
    elseif player:level() < 11 then
      htmltext = "30567-01.htm"
    else
      htmltext = "30567-02.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30567-04.htm"
    else
      htmltext = "30567-05.htm"
      q:take_items(player, DARKWING_BAT_FANG, -1)
      q:take_items(player, VARANGKA_PARASITE, -1)
      q:give_adena(player, 4200, true)
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
  local sw = npc:id()
  if sw == DARKWING_BAT then
    q:give_items(player, DARKWING_BAT_FANG, 1)
    if q:get_quest_items_count(player, DARKWING_BAT_FANG) < 70 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
    if rnd(100) < 10 and q:get_quest_items_count(player, DARKWING_BAT_FANG) > 10 and q:get_quest_items_count(player, DARKWING_BAT_FANG) < 66 then
      -- Spawn of Varangka Tracker on the npc position.
      q:add_spawn(VARANGKA_TRACKER, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
      q:give_items(player, VARANGKA_PARASITE, 1)
    end
  elseif sw == VARANGKA_TRACKER then
    if q:has_quest_items(player, VARANGKA_PARASITE) then
      q:take_items(player, VARANGKA_PARASITE, -1)
      if q:get_quest_items_count(player, DARKWING_BAT_FANG) < 70 then
        q:give_items(player, DARKWING_BAT_FANG, 5)
        if q:get_quest_items_count(player, DARKWING_BAT_FANG) < 70 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:set_cond(2, true)
        end
      end
    end
  end
end)
