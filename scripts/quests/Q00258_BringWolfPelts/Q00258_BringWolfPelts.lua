-- Translated by questport from quests/Q00258_BringWolfPelts (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(258, "Q00258_BringWolfPelts", "Bring Wolf Pelts")

-- Item
local WOLF_PELT = 702
-- Rewards
local COTTON_SHIRT = 390
local LEATHER_PANTS = 29
local LEATHER_SHIRT = 22
local SHORT_LEATHER_GLOVES = 1119
local TUNIC = 426

q:register_quest_items(WOLF_PELT)
q:add_start_npc(30001) -- Lector
q:add_talk_id(30001)
q:add_kill_id(20120, 20442) -- Wolf, Elder Wolf

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30001-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 3 and "30001-01.htm" or "30001-02.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, WOLF_PELT) < 40 then
      htmltext = "30001-05.htm"
    else
      q:take_items(player, WOLF_PELT, -1)
      local randomNumber = rnd(16)
      -- Reward is based on a random number (1D16).
      if randomNumber == 0 then
        q:give_items(player, COTTON_SHIRT, 1)
      elseif randomNumber < 6 then
        q:give_items(player, LEATHER_PANTS, 1)
      elseif randomNumber < 9 then
        q:give_items(player, LEATHER_SHIRT, 1)
      elseif randomNumber < 13 then
        q:give_items(player, SHORT_LEATHER_GLOVES, 1)
      else
        q:give_items(player, TUNIC, 1)
      end
      htmltext = "30001-06.htm"
      if randomNumber == 0 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_JACKPOT)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FINISH)
      end
      st:exit_quest(true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  q:give_items(player, WOLF_PELT, 1)
  if q:get_quest_items_count(player, WOLF_PELT) >= 40 then
    st:set_cond(2, true)
  else
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
