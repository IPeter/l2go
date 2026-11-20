-- Translated by questport from quests/Q00264_KeenClaws (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(264, "Q00264_KeenClaws", "Keen Claws")

-- Item
local WOLF_CLAW = 1367
-- Rewards
local LEATHER_SANDALS = 36
local WOODEN_HELMET = 43
local STOCKINGS = 462
local HEALING_POTION = 1061
local SHORT_GLOVES = 48
local CLOTH_SHOES = 35

q:register_quest_items(WOLF_CLAW)
q:add_start_npc(30136) -- Payne
q:add_talk_id(30136)
q:add_kill_id(20003, 20456) -- Goblin, Wolf

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30136-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 3 and "30136-01.htm" or "30136-02.htm"
  elseif sw == State.STARTED then
    local count = q:get_quest_items_count(player, WOLF_CLAW)
    if count < 50 then
      htmltext = "30136-04.htm"
    else
      htmltext = "30136-05.htm"
      q:take_items(player, WOLF_CLAW, -1)
      local n = rnd(17)
      if n == 0 then
        q:give_items(player, WOODEN_HELMET, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_JACKPOT)
      elseif n < 2 then
        q:give_adena(player, 1000, true)
      elseif n < 5 then
        q:give_items(player, LEATHER_SANDALS, 1)
      elseif n < 8 then
        q:give_items(player, STOCKINGS, 1)
        q:give_adena(player, 50, true)
      elseif n < 11 then
        q:give_items(player, HEALING_POTION, 1)
      elseif n < 14 then
        q:give_items(player, SHORT_GLOVES, 1)
      else
        q:give_items(player, CLOTH_SHOES, 1)
      end
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  q:give_items(player, WOLF_CLAW, (rnd.bool() and 1 or 2) * (npc:id() == 20003 and 2 or 1))
  if q:get_quest_items_count(player, WOLF_CLAW) >= 50 then
    st:set_cond(2, true)
  else
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
