-- Translated by questport from quests/Q00367_ElectrifyingRecharge (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(367, "Q00367_ElectrifyingRecharge", "Electrifying Recharge!")

-- NPCs
local LORAIN = 30673
-- Monsters
local CATHEROK = 21035
-- Item
local LORAIN_LAMP = 5875
local TITAN_LAMP_1 = 5876
local TITAN_LAMP_2 = 5877
local TITAN_LAMP_3 = 5878
local TITAN_LAMP_4 = 5879
local TITAN_LAMP_5 = 5880
-- Reward
local REWARD = {4553, 4554, 4555, 4556, 4557, 4558, 4559, 4560, 4561, 4562, 4563, 4564}

q:register_quest_items(LORAIN_LAMP, TITAN_LAMP_1, TITAN_LAMP_2, TITAN_LAMP_3, TITAN_LAMP_4, TITAN_LAMP_5)
q:add_start_npc(LORAIN)
q:add_talk_id(LORAIN)
q:add_spell_finished_id(CATHEROK)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30673-03.htm" then
    st:start_quest()
    q:give_items(player, LORAIN_LAMP, 1)
  elseif event == "30673-09.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    q:give_items(player, LORAIN_LAMP, 1)
  elseif event == "30673-08.htm" then
    st:exit_quest(true, true)
  elseif event == "30673-07.htm" then
    st:set_cond(1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    q:give_items(player, LORAIN_LAMP, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 37 and "30673-02.htm" or "30673-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      if q:has_quest_items(player, 5880) then
        htmltext = "30673-05.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
        q:take_items(player, 5880, 1)
        q:give_items(player, LORAIN_LAMP, 1)
      elseif q:has_quest_items(player, 5876) then
        htmltext = "30673-04.htm"
        q:take_items(player, 5876, 1)
      elseif q:has_quest_items(player, 5877) then
        htmltext = "30673-04.htm"
        q:take_items(player, 5877, 1)
      elseif q:has_quest_items(player, 5878) then
        htmltext = "30673-04.htm"
        q:take_items(player, 5878, 1)
      else
        htmltext = "30673-03.htm"
      end
    elseif cond == 2 and q:has_quest_items(player, 5879) then
      htmltext = "30673-06.htm"
      q:take_items(player, 5879, 1)
      q:reward_items(player, rnd.entry(REWARD), 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FINISH)
    end
  end
  return htmltext
end)

q:on_spell_finished(function(npc, player, skill)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  -- L2Go: skill:id() is a method (the translation read the field skill.id).
  if skill:id() == 4072 and q:has_quest_items(player, LORAIN_LAMP) then
    local randomItem = rnd(5876, 5880)
    q:take_items(player, LORAIN_LAMP, 1)
    q:give_items(player, randomItem, 1)
    if randomItem == 5879 then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
