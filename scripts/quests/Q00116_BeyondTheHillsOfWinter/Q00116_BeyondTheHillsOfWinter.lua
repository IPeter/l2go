-- Translated by questport from quests/Q00116_BeyondTheHillsOfWinter (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(116, "Q00116_BeyondTheHillsOfWinter", "Beyond the Hills of Winter")

-- NPCs
local FILAUR = 30535
local OBI = 32052
-- Items
local BANDAGE = 1833
local ENERGY_STONE = 5589
local THIEF_KEY = 1661
local GOODS = 8098
-- Reward
local SSD = 1463

q:register_quest_items(GOODS)
q:add_start_npc(FILAUR)
q:add_talk_id(FILAUR, OBI)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30535-02.htm" then
    st:start_quest()
  elseif event == "30535-05.htm" then
    st:set_cond(2, true)
    q:give_items(player, GOODS, 1)
  elseif event == "materials" then
    htmltext = "32052-02.htm"
    q:take_items(player, GOODS, -1)
    q:reward_items(player, SSD, 1650)
    st:exit_quest(false, true)
  elseif event == "adena" then
    htmltext = "32052-02.htm"
    q:take_items(player, GOODS, -1)
    q:give_adena(player, 16500, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (player:level() < 30 or player:race() ~= Race.DWARF) and "30535-00.htm" or "30535-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == FILAUR then
      if cond == 1 then
        if q:get_quest_items_count(player, BANDAGE) >= 20 and q:get_quest_items_count(player, ENERGY_STONE) >= 5 and q:get_quest_items_count(player, THIEF_KEY) >= 10 then
          htmltext = "30535-03.htm"
          q:take_items(player, BANDAGE, 20)
          q:take_items(player, ENERGY_STONE, 5)
          q:take_items(player, THIEF_KEY, 10)
        else
          htmltext = "30535-04.htm"
        end
      elseif cond == 2 then
        htmltext = "30535-05.htm"
      end
    elseif sw2 == OBI then
      if cond == 2 then
        htmltext = "32052-00.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
