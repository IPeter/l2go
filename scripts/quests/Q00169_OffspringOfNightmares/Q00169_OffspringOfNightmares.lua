-- Translated by questport from quests/Q00169_OffspringOfNightmares (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(169, "Q00169_OffspringOfNightmares", "Offspring of Nightmares")

-- Items
local CRACKED_SKULL = 1030
local PERFECT_SKULL = 1031
local BONE_GAITERS = 31

q:register_quest_items(CRACKED_SKULL, PERFECT_SKULL)
q:add_start_npc(30145) -- Vlasty
q:add_talk_id(30145)
q:add_kill_id(20105, 20025)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30145-04.htm" then
    st:start_quest()
  elseif event == "30145-08.htm" then
    local reward = 17000 + q:get_quest_items_count(player, CRACKED_SKULL) * 20
    q:take_items(player, PERFECT_SKULL, -1)
    q:take_items(player, CRACKED_SKULL, -1)
    q:give_items(player, BONE_GAITERS, 1)
    q:give_adena(player, reward, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30145-00.htm"
    elseif player:level() < 15 then
      htmltext = "30145-02.htm"
    else
      htmltext = "30145-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      if q:has_quest_items(player, CRACKED_SKULL) then
        htmltext = "30145-06.htm"
      else
        htmltext = "30145-05.htm"
      end
    elseif cond == 2 then
      htmltext = "30145-07.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_started() then
    if rnd(10) > 7 and not q:has_quest_items(killer, PERFECT_SKULL) then
      q:give_items(killer, PERFECT_SKULL, 1)
      qs:set_cond(2, true)
    elseif rnd(10) > 4 then
      q:give_items(killer, CRACKED_SKULL, 1)
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
