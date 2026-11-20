-- Translated by questport from quests/Q00029_ChestCaughtWithABaitOfEarth (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(29, "Q00029_ChestCaughtWithABaitOfEarth", "Chest caught with a bait of earth")
-- The chest is fished up.
q:requires("fishing")

-- NPCs
local WILLIE = 31574
local ANABEL = 30909
-- Items
local SMALL_PURPLE_TREASURE_CHEST = 6507
local SMALL_GLASS_BOX = 7627
local PLATED_LEATHER_GLOVES = 2455

q:register_quest_items(SMALL_GLASS_BOX)
q:add_start_npc(WILLIE)
q:add_talk_id(WILLIE, ANABEL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31574-04.htm" then
    st:start_quest()
  elseif event == "31574-07.htm" then
    if q:has_quest_items(player, SMALL_PURPLE_TREASURE_CHEST) then
      st:set_cond(2)
      q:take_items(player, SMALL_PURPLE_TREASURE_CHEST, 1)
      q:give_items(player, SMALL_GLASS_BOX, 1)
    else
      htmltext = "31574-08.htm"
    end
  elseif event == "30909-02.htm" then
    if q:has_quest_items(player, SMALL_GLASS_BOX) then
      htmltext = "30909-02.htm"
      q:take_items(player, SMALL_GLASS_BOX, 1)
      q:give_items(player, PLATED_LEATHER_GLOVES, 1)
      st:exit_quest(false, true)
    else
      htmltext = "30909-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 48 then
      htmltext = "31574-02.htm"
    else
      local st2 = player:get_quest_state("Q00052_WilliesSpecialBait")
      if st2 ~= nil and st2:is_completed() then
        htmltext = "31574-01.htm"
      else
        htmltext = "31574-03.htm"
      end
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == WILLIE then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, SMALL_PURPLE_TREASURE_CHEST) and "31574-06.htm" or "31574-05.htm"
      elseif cond == 2 then
        htmltext = "31574-09.htm"
      end
    elseif sw2 == ANABEL then
      if cond == 2 then
        htmltext = "30909-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
