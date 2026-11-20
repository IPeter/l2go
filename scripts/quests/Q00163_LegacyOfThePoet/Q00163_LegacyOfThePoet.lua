-- Translated by questport from quests/Q00163_LegacyOfThePoet (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Legacy of the Poet (163)
-- @author xban1x
local J = require("lib.java")

local q = quest.new(163, "Q00163_LegacyOfThePoet", "Legacy of the Poet")

-- NPC
local STARDEN = 30220
-- Monsters
local MONSTERS = {20372, 20373}
-- Items
local RUMIELS_1ST_POEM = 1038
local RUMIELS_2ND_POEM = 1039
local RUMIELS_3RD_POEM = 1040
local RUMIELS_4TH_POEM = 1041
-- Misc
local MIN_LEVEL = 11

q:add_start_npc(STARDEN)
q:add_talk_id(STARDEN)
q:add_kill_id(MONSTERS)
q:register_quest_items(RUMIELS_1ST_POEM, RUMIELS_2ND_POEM, RUMIELS_3RD_POEM, RUMIELS_4TH_POEM)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  local htmltext = nil
  if qs ~= nil then
    if event == "30220-03.html" or event == "30220-04.html" then
      htmltext = event
    elseif event == "30220-05.htm" then
      qs:start_quest()
      htmltext = event
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:race() ~= Race.DARK_ELF and (player:level() >= MIN_LEVEL and "30220-02.htm" or "30220-01.htm") or "30220-00.htm"
  elseif sw == State.STARTED then
    if q:has_quest_items(player, RUMIELS_1ST_POEM, RUMIELS_2ND_POEM, RUMIELS_3RD_POEM, RUMIELS_4TH_POEM) then
      q:give_adena(player, 13890, true)
      qs:exit_quest(false, true)
      htmltext = "30220-07.html"
    else
      htmltext = "30220-06.html"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) then
    if rnd(10) == 0 and not q:has_quest_items(killer, RUMIELS_1ST_POEM) then
      q:give_items(killer, RUMIELS_1ST_POEM, 1)
      if q:has_quest_items(killer, RUMIELS_2ND_POEM, RUMIELS_3RD_POEM, RUMIELS_4TH_POEM) then
        qs:set_cond(2, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
    if rnd(10) > 7 and not q:has_quest_items(killer, RUMIELS_2ND_POEM) then
      q:give_items(killer, RUMIELS_2ND_POEM, 1)
      if q:has_quest_items(killer, RUMIELS_1ST_POEM, RUMIELS_3RD_POEM, RUMIELS_4TH_POEM) then
        qs:set_cond(2, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
    if rnd(10) > 7 and not q:has_quest_items(killer, RUMIELS_3RD_POEM) then
      q:give_items(killer, RUMIELS_3RD_POEM, 1)
      if q:has_quest_items(killer, RUMIELS_1ST_POEM, RUMIELS_2ND_POEM, RUMIELS_4TH_POEM) then
        qs:set_cond(2, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
    if rnd(10) > 5 and not q:has_quest_items(killer, RUMIELS_4TH_POEM) then
      q:give_items(killer, RUMIELS_4TH_POEM, 1)
      if q:has_quest_items(killer, RUMIELS_1ST_POEM, RUMIELS_2ND_POEM, RUMIELS_3RD_POEM) then
        qs:set_cond(2, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
