-- Translated by questport from quests/Q00161_FruitOfTheMotherTree (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(161, "Q00161_FruitOfTheMotherTree", "Fruit of the Mothertree")

-- NPCs
local ANDELLIA = 30362
local THALIA = 30371
-- Items
local ANDELLIA_LETTER = 1036
local MOTHERTREE_FRUIT = 1037

q:register_quest_items(ANDELLIA_LETTER, MOTHERTREE_FRUIT)
q:add_start_npc(ANDELLIA)
q:add_talk_id(ANDELLIA, THALIA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30362-04.htm" then
    st:start_quest()
    q:give_items(player, ANDELLIA_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30362-00.htm"
    elseif player:level() < 3 then
      htmltext = "30362-02.htm"
    else
      htmltext = "30362-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ANDELLIA then
      if cond == 1 then
        htmltext = "30362-05.htm"
      elseif cond == 2 then
        htmltext = "30362-06.htm"
        q:take_items(player, MOTHERTREE_FRUIT, 1)
        q:give_adena(player, 1000, true)
        q:add_exp_and_sp(player, 1000, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == THALIA then
      if cond == 1 then
        htmltext = "30371-01.htm"
        st:set_cond(2, true)
        q:take_items(player, ANDELLIA_LETTER, 1)
        q:give_items(player, MOTHERTREE_FRUIT, 1)
      elseif cond == 2 then
        htmltext = "30371-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
