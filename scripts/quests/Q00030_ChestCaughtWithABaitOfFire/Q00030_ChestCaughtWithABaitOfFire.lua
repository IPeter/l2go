-- Translated by questport from quests/Q00030_ChestCaughtWithABaitOfFire (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(30, "Q00030_ChestCaughtWithABaitOfFire", "Chest caught with a bait of fire")
-- The chest is fished up.
q:requires("fishing")

-- NPCs
local LINNAEUS = 31577
local RUKAL = 30629
-- Items
local RED_TREASURE_BOX = 6511
local MUSICAL_SCORE = 7628
local NECKLACE_OF_PROTECTION = 916

q:register_quest_items(MUSICAL_SCORE)
q:add_start_npc(LINNAEUS)
q:add_talk_id(LINNAEUS, RUKAL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31577-04.htm" then
    st:start_quest()
  elseif event == "31577-07.htm" then
    if q:has_quest_items(player, RED_TREASURE_BOX) then
      st:set_cond(2)
      q:take_items(player, RED_TREASURE_BOX, 1)
      q:give_items(player, MUSICAL_SCORE, 1)
    else
      htmltext = "31577-08.htm"
    end
  elseif event == "30629-02.htm" then
    if q:has_quest_items(player, MUSICAL_SCORE) then
      htmltext = "30629-02.htm"
      q:take_items(player, MUSICAL_SCORE, 1)
      q:give_items(player, NECKLACE_OF_PROTECTION, 1)
      st:exit_quest(false, true)
    else
      htmltext = "30629-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 60 then
      htmltext = "31577-02.htm"
    else
      local st2 = player:get_quest_state("Q00053_LinnaeusSpecialBait")
      if st2 ~= nil and st2:is_completed() then
        htmltext = "31577-01.htm"
      else
        htmltext = "31577-03.htm"
      end
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LINNAEUS then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, RED_TREASURE_BOX) and "31577-06.htm" or "31577-05.htm"
      elseif cond == 2 then
        htmltext = "31577-09.htm"
      end
    elseif sw2 == RUKAL then
      if cond == 2 then
        htmltext = "30629-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
