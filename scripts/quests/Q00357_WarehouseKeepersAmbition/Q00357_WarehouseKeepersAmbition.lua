-- Translated by questport from quests/Q00357_WarehouseKeepersAmbition (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(357, "Q00357_WarehouseKeepersAmbition", "Warehouse Keeper's Ambition")

-- Monsters
local FOREST_RUNNER = 20594
local FLINE_ELDER = 20595
local LIELE_ELDER = 20596
local VALLEY_TREANT_ELDER = 20597
-- Item
local JADE_CRYSTAL = 5867
-- Drop chances
local DROP_DATA = {}

DROP_DATA[FOREST_RUNNER] = 0.577 -- Forest Runner
DROP_DATA[FLINE_ELDER] = 0.6 -- Fline Elder
DROP_DATA[LIELE_ELDER] = 0.638 -- Liele Elder
DROP_DATA[VALLEY_TREANT_ELDER] = 0.062 -- Valley Treant Elder

q:register_quest_items(JADE_CRYSTAL)
q:add_start_npc(30686) -- Silva
q:add_talk_id(30686)
q:add_kill_id(FOREST_RUNNER, FLINE_ELDER, LIELE_ELDER, VALLEY_TREANT_ELDER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30686-2.htm" then
    st:start_quest()
  elseif event == "30686-7.htm" then
    local count = q:get_quest_items_count(player, JADE_CRYSTAL)
    if count == 0 then
      htmltext = "30686-4.htm"
    else
      local reward = count * 425 + 3500
      if count >= 100 then
        reward = reward + 7400
      end
      q:take_items(player, JADE_CRYSTAL, -1)
      q:give_adena(player, reward, true)
    end
  elseif event == "30686-8.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 47 and "30686-0a.htm" or "30686-0.htm"
  elseif sw == State.STARTED then
    htmltext = not q:has_quest_items(player, JADE_CRYSTAL) and "30686-4.htm" or "30686-6.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, -1, 3, npc)
  if qs ~= nil then
    q:give_item_randomly(qs:get_player(), npc, JADE_CRYSTAL, 1, 1, 0, DROP_DATA[npc:id()], true)
  end
end)
