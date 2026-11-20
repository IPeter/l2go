-- Translated by questport from quests/Q00028_ChestCaughtWithABaitOfIcyAir (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(28, "Q00028_ChestCaughtWithABaitOfIcyAir", "Chest caught with a bait of icy air")
-- The chest is fished up.
q:requires("fishing")

-- NPCs
local OFULLE = 31572
local KIKI = 31442
-- Items
local BIG_YELLOW_TREASURE_CHEST = 6503
local KIKI_LETTER = 7626
local ELVEN_RING = 881

q:register_quest_items(KIKI_LETTER)
q:add_start_npc(OFULLE)
q:add_talk_id(OFULLE, KIKI)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31572-04.htm" then
    st:start_quest()
  elseif event == "31572-07.htm" then
    if q:has_quest_items(player, BIG_YELLOW_TREASURE_CHEST) then
      st:set_cond(2)
      q:take_items(player, BIG_YELLOW_TREASURE_CHEST, 1)
      q:give_items(player, KIKI_LETTER, 1)
    else
      htmltext = "31572-08.htm"
    end
  elseif event == "31442-02.htm" then
    if q:has_quest_items(player, KIKI_LETTER) then
      htmltext = "31442-02.htm"
      q:take_items(player, KIKI_LETTER, 1)
      q:give_items(player, ELVEN_RING, 1)
      st:exit_quest(false, true)
    else
      htmltext = "31442-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 36 then
      htmltext = "31572-02.htm"
    else
      local st2 = player:get_quest_state("Q00051_OFullesSpecialBait")
      if st2 ~= nil and st2:is_completed() then
        htmltext = "31572-01.htm"
      else
        htmltext = "31572-03.htm"
      end
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == OFULLE then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, BIG_YELLOW_TREASURE_CHEST) and "31572-06.htm" or "31572-05.htm"
      elseif cond == 2 then
        htmltext = "31572-09.htm"
      end
    elseif sw2 == KIKI then
      if cond == 2 then
        htmltext = "31442-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
