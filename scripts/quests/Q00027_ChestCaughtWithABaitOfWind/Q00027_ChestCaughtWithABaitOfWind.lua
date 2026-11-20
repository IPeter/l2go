-- Translated by questport from quests/Q00027_ChestCaughtWithABaitOfWind (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(27, "Q00027_ChestCaughtWithABaitOfWind", "Chest caught with a bait of wind")
-- The chest is fished up.
q:requires("fishing")

-- NPCs
local LANOSCO = 31570
local SHALING = 31434
-- Items
local LARGE_BLUE_TREASURE_CHEST = 6500
local STRANGE_BLUEPRINT = 7625
local BLACK_PEARL_RING = 880

q:register_quest_items(STRANGE_BLUEPRINT)
q:add_start_npc(LANOSCO)
q:add_talk_id(LANOSCO, SHALING)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31570-04.htm" then
    st:start_quest()
  elseif event == "31570-07.htm" then
    if q:has_quest_items(player, LARGE_BLUE_TREASURE_CHEST) then
      st:set_cond(2)
      q:take_items(player, LARGE_BLUE_TREASURE_CHEST, 1)
      q:give_items(player, STRANGE_BLUEPRINT, 1)
    else
      htmltext = "31570-08.htm"
    end
  elseif event == "31434-02.htm" then
    if q:has_quest_items(player, STRANGE_BLUEPRINT) then
      htmltext = "31434-02.htm"
      q:take_items(player, STRANGE_BLUEPRINT, 1)
      q:give_items(player, BLACK_PEARL_RING, 1)
      st:exit_quest(false, true)
    else
      htmltext = "31434-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 27 then
      htmltext = "31570-02.htm"
    else
      local st2 = player:get_quest_state("Q00050_LanoscosSpecialBait")
      if st2 ~= nil and st2:is_completed() then
        htmltext = "31570-01.htm"
      else
        htmltext = "31570-03.htm"
      end
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LANOSCO then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, LARGE_BLUE_TREASURE_CHEST) and "31570-06.htm" or "31570-05.htm"
      elseif cond == 2 then
        htmltext = "31570-09.htm"
      end
    elseif sw2 == SHALING then
      if cond == 2 then
        htmltext = "31434-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
