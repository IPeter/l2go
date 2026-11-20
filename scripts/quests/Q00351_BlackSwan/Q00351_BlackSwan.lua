-- Translated by questport from quests/Q00351_BlackSwan (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(351, "Q00351_BlackSwan", "Black Swan")

-- NPCs
local GOSTA = 30916
local IASON_HEINE = 30969
local ROMAN = 30897
-- Items
local ORDER_OF_GOSTA = 4296
local LIZARD_FANG = 4297
local BARREL_OF_LEAGUE = 4298
local BILL_OF_IASON_HEINE = 4407

q:register_quest_items(ORDER_OF_GOSTA, BARREL_OF_LEAGUE, LIZARD_FANG)
q:add_start_npc(GOSTA)
q:add_talk_id(GOSTA, IASON_HEINE, ROMAN)
q:add_kill_id(20784, 20785, 21639, 21640)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30916-03.htm" then
    st:start_quest()
    q:give_items(player, ORDER_OF_GOSTA, 1)
  elseif event == "30969-02a.htm" then
    local lizardFangs = q:get_quest_items_count(player, LIZARD_FANG)
    if lizardFangs > 0 then
      htmltext = "30969-02.htm"
      q:take_items(player, LIZARD_FANG, -1)
      q:give_adena(player, lizardFangs * 20, true)
    end
  elseif event == "30969-03a.htm" then
    local barrels = q:get_quest_items_count(player, BARREL_OF_LEAGUE)
    if barrels > 0 then
      htmltext = "30969-03.htm"
      q:take_items(player, BARREL_OF_LEAGUE, -1)
      q:reward_items(player, BILL_OF_IASON_HEINE, barrels)
      -- Heine explains than player can speak with Roman in order to exchange bills for rewards.
      if st:is_cond(1) then
        st:set_cond(2, true)
      end
    end
  elseif event == "30969-06.htm" then
    -- If no more quest items finish the quest for real, else send a "Return" type HTM.
    if not q:has_quest_items(player, BARREL_OF_LEAGUE, LIZARD_FANG) then
      htmltext = "30969-07.htm"
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 32 and "30916-00.htm" or "30916-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == GOSTA then
      htmltext = "30916-04.htm"
    elseif sw2 == IASON_HEINE then
      htmltext = "30969-01.htm"
    elseif sw2 == ROMAN then
      htmltext = q:has_quest_items(player, BILL_OF_IASON_HEINE) and "30897-01.htm" or "30897-02.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local random = rnd(4)
  if random < 3 then
    q:give_items(player, LIZARD_FANG, random < 2 and 1 or 2)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    if rnd.bool() then
      q:give_items(player, BARREL_OF_LEAGUE, 1)
    end
  elseif rnd(10) < (npc:id() > 20785 and 3 or 4) then
    q:give_items(player, BARREL_OF_LEAGUE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
