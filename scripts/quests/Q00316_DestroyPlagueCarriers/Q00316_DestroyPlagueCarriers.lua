-- Translated by questport from quests/Q00316_DestroyPlagueCarriers (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(316, "Q00316_DestroyPlagueCarriers", "Destroy Plague Carriers")

-- Monsters
local SUKAR_WERERAT = 20040
local SUKAR_WERERAT_LEADER = 20047
local VAROOL_FOULCLAW = 27020
-- Items
local WERERAT_FANG = 1042
local VAROOL_FOULCLAW_FANG = 1043

q:register_quest_items(WERERAT_FANG, VAROOL_FOULCLAW_FANG)
q:add_start_npc(30155) -- Ellenia
q:add_talk_id(30155)
q:add_kill_id(SUKAR_WERERAT, SUKAR_WERERAT_LEADER, VAROOL_FOULCLAW)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30155-04.htm" then
    st:start_quest()
  elseif event == "30155-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30155-00.htm"
    elseif player:level() < 18 then
      htmltext = "30155-02.htm"
    else
      htmltext = "30155-03.htm"
    end
  elseif sw == State.STARTED then
    local ratFangs = q:get_quest_items_count(player, WERERAT_FANG)
    local varoolFangs = q:get_quest_items_count(player, VAROOL_FOULCLAW_FANG)
    if ratFangs + varoolFangs == 0 then
      htmltext = "30155-05.htm"
    else
      htmltext = "30155-07.htm"
      q:take_items(player, WERERAT_FANG, -1)
      q:take_items(player, VAROOL_FOULCLAW_FANG, -1)
      local reward = ratFangs * 30 + varoolFangs * 10000
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and ratFangs >= 10 then
        reward = reward + 5000
      end
      q:give_adena(player, reward, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == SUKAR_WERERAT or sw == SUKAR_WERERAT_LEADER then
    if rnd(10) < 4 then
      q:give_items(player, WERERAT_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == VAROOL_FOULCLAW then
    if not q:has_quest_items(player, VAROOL_FOULCLAW_FANG) and rnd(10) < 2 then
      q:give_items(player, VAROOL_FOULCLAW_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
