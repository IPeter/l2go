-- Translated by questport from quests/Q00360_PlunderTheirSupplies (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(360, "Q00360_PlunderTheirSupplies", "Plunder Their Supplies")

-- Items
local SUPPLY_ITEM = 5872
local SUSPICIOUS_DOCUMENT = 5871
local RECIPE_OF_SUPPLY = 5870
-- Drops
local MONSTER_DROP_CHANCES = {}

MONSTER_DROP_CHANCES[20666] = 50 -- Taik Orc Seeker
MONSTER_DROP_CHANCES[20669] = 75 -- Taik Orc Supply Leader

q:register_quest_items(RECIPE_OF_SUPPLY, SUPPLY_ITEM, SUSPICIOUS_DOCUMENT)
q:add_start_npc(30873) -- Coleman
q:add_talk_id(30873)
q:add_kill_id(J.keys(MONSTER_DROP_CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  if event == "30873-2.htm" then
    st:start_quest()
  elseif event == "30873-6.htm" then
    q:take_items(player, SUPPLY_ITEM, -1)
    q:take_items(player, SUSPICIOUS_DOCUMENT, -1)
    q:take_items(player, RECIPE_OF_SUPPLY, -1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 52 and "30873-0a.htm" or "30873-0.htm"
  elseif sw == State.STARTED then
    local supplyItems = q:get_quest_items_count(player, SUPPLY_ITEM)
    if supplyItems == 0 then
      htmltext = "30873-3.htm"
    else
      local reward = 6000 + supplyItems * 100 + q:get_quest_items_count(player, RECIPE_OF_SUPPLY) * 6000
      htmltext = "30873-5.htm"
      q:take_items(player, SUPPLY_ITEM, -1)
      q:take_items(player, RECIPE_OF_SUPPLY, -1)
      q:give_adena(player, reward, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isPet)
  local qs = q:get_quest_state(killer, false)
  if qs == nil or not util.in_range(Config.ALT_PARTY_RANGE, npc, killer, false) then
    return
  end
  if rnd(100) < MONSTER_DROP_CHANCES[npc:id()] then
    q:give_items(killer, SUPPLY_ITEM, 1)
    q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
  if rnd(100) < 10 then
    if q:get_quest_items_count(killer, SUSPICIOUS_DOCUMENT) < 4 then
      q:give_items(killer, SUSPICIOUS_DOCUMENT, 1)
    else
      q:give_items(killer, RECIPE_OF_SUPPLY, 1)
      q:take_items(killer, SUSPICIOUS_DOCUMENT, -1)
    end
    q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
