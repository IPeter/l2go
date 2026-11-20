-- Translated by questport from quests/Q00641_AttackSailren (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(641, "Q00641_AttackSailren", "Attack Sailren!")
-- The Gazkh opens Sailren's lair (grand bosses). The reward also needs
-- npc:magic_skill_use (Mobius MagicSkillUse broadcast), which does not
-- exist yet.
q:requires("grand_bosses")

-- NPC
local STATUE = 32109
-- Quest Item
local GAZKH_FRAGMENT = 8782
local GAZKH = 8784

q:register_quest_items(GAZKH_FRAGMENT)
q:add_start_npc(STATUE)
q:add_talk_id(STATUE)
q:add_kill_id(22196, 22197, 22198, 22199, 22218, 22223)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return nil
  end
  local htmltext = event
  if event == "32109-5.htm" then
    st:start_quest()
  elseif event == "32109-8.htm" then
    if q:get_quest_items_count(player, GAZKH_FRAGMENT) >= 30 then
      npc:magic_skill_use(player, 5089, 1, 3000, 0)
      q:take_items(player, GAZKH_FRAGMENT, -1)
      q:give_items(player, GAZKH, 1)
      st:exit_quest(true, true)
    else
      htmltext = "32109-6.htm"
      st:set_cond(1)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 77 then
      htmltext = "32109-3.htm"
    else
      local st2 = player:get_quest_state("Q00126_TheNameOfEvil2")
      htmltext = st2 ~= nil and st2:is_completed() and "32109-1.htm" or "32109-2.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32109-5.htm"
    elseif cond == 2 then
      htmltext = "32109-7.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd(100) < 5 then
    q:give_items(partyMember, GAZKH_FRAGMENT, 1)
    if q:get_quest_items_count(partyMember, GAZKH_FRAGMENT) < 30 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
