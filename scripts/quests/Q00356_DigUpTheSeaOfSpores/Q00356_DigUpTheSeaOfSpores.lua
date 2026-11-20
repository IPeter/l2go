-- Translated by questport from quests/Q00356_DigUpTheSeaOfSpores (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(356, "Q00356_DigUpTheSeaOfSpores", "Dig Up the Sea of Spores!")

-- Items
local HERB_SPORE = 5866
local CARN_SPORE = 5865
-- Monsters
local ROTTING_TREE = 20558
local SPORE_ZOMBIE = 20562

q:register_quest_items(HERB_SPORE, CARN_SPORE)
q:add_start_npc(30717) -- Gauen
q:add_talk_id(30717)
q:add_kill_id(ROTTING_TREE, SPORE_ZOMBIE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30717-06.htm" then
    st:start_quest()
  elseif event == "30717-16.htm" then
    q:take_items(player, HERB_SPORE, -1)
    q:take_items(player, CARN_SPORE, -1)
    q:give_adena(player, 44000, true)
    st:exit_quest(true, true)
  elseif event == "30717-14.htm" then
    q:take_items(player, HERB_SPORE, -1)
    q:take_items(player, CARN_SPORE, -1)
    q:add_exp_and_sp(player, 35000, 2600)
    st:exit_quest(true, true)
  elseif event == "30717-12.htm" then
    q:take_items(player, HERB_SPORE, -1)
    q:add_exp_and_sp(player, 24500, 0)
  elseif event == "30717-13.htm" then
    q:take_items(player, CARN_SPORE, -1)
    q:add_exp_and_sp(player, 0, 1820)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 43 and "30717-01.htm" or "30717-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30717-07.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, HERB_SPORE) >= 50 then
        htmltext = "30717-08.htm"
      elseif q:get_quest_items_count(player, CARN_SPORE) >= 50 then
        htmltext = "30717-09.htm"
      else
        htmltext = "30717-07.htm"
      end
    elseif cond == 3 then
      htmltext = "30717-10.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local cond = st:get_cond()
  if cond < 3 then
    local sw = npc:id()
    if sw == ROTTING_TREE then
      if rnd(100) < 63 then
        q:give_items(player, HERB_SPORE, 1)
        if q:get_quest_items_count(player, HERB_SPORE) < 50 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:set_cond(cond == 2 and 3 or 2, true)
        end
      end
    elseif sw == SPORE_ZOMBIE then
      if rnd(100) < 76 then
        q:give_items(player, CARN_SPORE, 1)
        if q:get_quest_items_count(player, CARN_SPORE) < 50 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:set_cond(cond == 2 and 3 or 2, true)
        end
      end
    end
  end
end)
