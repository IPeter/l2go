-- Translated by questport from quests/Q00379_FantasyWine (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(379, "Q00379_FantasyWine", "Fantasy Wine")

-- NPCs
local HARLAN = 30074
-- Monsters
local ENKU_CHAMPION = 20291
local ENKU_SHAMAN = 20292
-- Items
local LEAF = 5893
local STONE = 5894

q:register_quest_items(LEAF, STONE)
q:add_start_npc(HARLAN)
q:add_talk_id(HARLAN)
q:add_kill_id(ENKU_CHAMPION, ENKU_SHAMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30074-3.htm" then
    st:start_quest()
  elseif event == "30074-6.htm" then
    q:take_items(player, LEAF, 80)
    q:take_items(player, STONE, 100)
    local rand = rnd(10)
    if rand < 3 then
      htmltext = "30074-6.htm"
      q:give_items(player, 5956, 1)
    elseif rand < 9 then
      htmltext = "30074-7.htm"
      q:give_items(player, 5957, 1)
    else
      htmltext = "30074-8.htm"
      q:give_items(player, 5958, 1)
    end
    st:exit_quest(true, true)
  elseif event == "30074-2a.htm" then
    st:exit_quest(true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "30074-0a.htm" or "30074-0.htm"
  elseif sw == State.STARTED then
    local leaf = q:get_quest_items_count(player, LEAF)
    local stone = q:get_quest_items_count(player, STONE)
    if leaf >= 80 and stone >= 100 then
      htmltext = "30074-5.htm"
    elseif leaf >= 80 then
      htmltext = "30074-4a.htm"
    elseif stone >= 100 then
      htmltext = "30074-4b.htm"
    else
      htmltext = "30074-4.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if npc:id() == ENKU_CHAMPION then
    -- Stop at the required amount, further kills would only bloat the inventory.
    if q:get_quest_items_count(player, LEAF) < 80 then
      q:give_items(player, LEAF, 1)
      if q:get_quest_items_count(player, LEAF) >= 80 and q:get_quest_items_count(player, STONE) >= 100 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  else
    -- Stop at the required amount, further kills would only bloat the inventory.
    if q:get_quest_items_count(player, STONE) < 100 then
      q:give_items(player, STONE, 1)
      if q:get_quest_items_count(player, STONE) >= 100 and q:get_quest_items_count(player, LEAF) >= 80 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
