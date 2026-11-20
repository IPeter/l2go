-- Translated by questport from quests/Q00292_BrigandsSweep (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(292, "Q00292_BrigandsSweep", "Brigands Sweep")

-- NPCs
local SPIRON = 30532
local BALANKI = 30533
-- Monsters
local GOBLIN_BRIGAND = 20322
local GOBLIN_BRIGAND_LEADER = 20323
local GOBLIN_BRIGAND_LIEUTENANT = 20324
local GOBLIN_SNOOPER = 20327
local GOBLIN_LORD = 20528
-- Items
local GOBLIN_NECKLACE = 1483
local GOBLIN_PENDANT = 1484
local GOBLIN_LORD_PENDANT = 1485
local SUSPICIOUS_MEMO = 1486
local SUSPICIOUS_CONTRACT = 1487

q:register_quest_items(GOBLIN_NECKLACE, GOBLIN_PENDANT, GOBLIN_LORD_PENDANT, SUSPICIOUS_MEMO, SUSPICIOUS_CONTRACT)
q:add_start_npc(SPIRON)
q:add_talk_id(SPIRON, BALANKI)
q:add_kill_id(GOBLIN_BRIGAND, GOBLIN_BRIGAND_LEADER, GOBLIN_BRIGAND_LIEUTENANT, GOBLIN_SNOOPER, GOBLIN_LORD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30532-03.htm" then
    st:start_quest()
  elseif event == "30532-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DWARF then
      htmltext = "30532-00.htm"
    elseif player:level() < 5 then
      htmltext = "30532-01.htm"
    else
      htmltext = "30532-02.htm"
    end
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SPIRON then
      local goblinNecklaces = q:get_quest_items_count(player, GOBLIN_NECKLACE)
      local goblinPendants = q:get_quest_items_count(player, GOBLIN_PENDANT)
      local goblinLordPendants = q:get_quest_items_count(player, GOBLIN_LORD_PENDANT)
      local suspiciousMemos = q:get_quest_items_count(player, SUSPICIOUS_MEMO)
      local countAll = goblinNecklaces + goblinPendants + goblinLordPendants
      local hasContract = q:has_quest_items(player, SUSPICIOUS_CONTRACT)
      if countAll == 0 then
        htmltext = "30532-04.htm"
      else
        if hasContract then
          htmltext = "30532-10.htm"
        elseif suspiciousMemos > 0 then
          if suspiciousMemos > 1 then
            htmltext = "30532-09.htm"
          else
            htmltext = "30532-08.htm"
          end
        else
          htmltext = "30532-05.htm"
        end
        q:take_items(player, GOBLIN_NECKLACE, -1)
        q:take_items(player, GOBLIN_PENDANT, -1)
        q:take_items(player, GOBLIN_LORD_PENDANT, -1)
        if hasContract then
          st:set_cond(1)
          q:take_items(player, SUSPICIOUS_CONTRACT, -1)
        end
        local reward = 12 * goblinNecklaces + 36 * goblinPendants + 33 * goblinLordPendants + (hasContract and 1120 or 0)
        if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and countAll >= 10 then
          reward = reward + 1000
        end
        q:give_adena(player, reward, true)
      end
    elseif sw2 == BALANKI then
      if not q:has_quest_items(player, SUSPICIOUS_CONTRACT) then
        htmltext = "30533-01.htm"
      else
        htmltext = "30533-02.htm"
        st:set_cond(1)
        q:take_items(player, SUSPICIOUS_CONTRACT, -1)
        q:give_adena(player, 1500, true)
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local chance = rnd(10)
  if chance > 5 then
    local sw = npc:id()
    if sw == GOBLIN_BRIGAND or sw == GOBLIN_SNOOPER or sw == GOBLIN_BRIGAND_LIEUTENANT then
      q:give_items(player, GOBLIN_NECKLACE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    elseif sw == GOBLIN_BRIGAND_LEADER then
      q:give_items(player, GOBLIN_PENDANT, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    elseif sw == GOBLIN_LORD then
      q:give_items(player, GOBLIN_LORD_PENDANT, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif chance > 4 and st:is_cond(1) then
    q:give_items(player, SUSPICIOUS_MEMO, 1)
    if q:get_quest_items_count(player, SUSPICIOUS_MEMO) < 3 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      q:take_items(player, SUSPICIOUS_MEMO, -1)
      q:give_items(player, SUSPICIOUS_CONTRACT, 1)
      st:set_cond(2, true)
    end
  end
end)
