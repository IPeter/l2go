-- Translated by questport from quests/Q00298_LizardmensConspiracy (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(298, "Q00298_LizardmensConspiracy", "Lizardmen's Conspiracy")

-- NPCs
local PRAGA = 30333
local ROHMER = 30344
-- Items
local PATROL_REPORT = 7182
local WHITE_GEM = 7183
local RED_GEM = 7184

q:register_quest_items(PATROL_REPORT, WHITE_GEM, RED_GEM)
q:add_start_npc(PRAGA)
q:add_talk_id(PRAGA, ROHMER)
q:add_kill_id(20926, 20927, 20922, 20923, 20924)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30333-1.htm" then
    st:start_quest()
    q:give_items(player, PATROL_REPORT, 1)
  elseif event == "30344-1.htm" then
    st:set_cond(2, true)
    q:take_items(player, PATROL_REPORT, 1)
  elseif event == "30344-4.htm" then
    if st:is_cond(3) then
      htmltext = "30344-3.htm"
      q:take_items(player, WHITE_GEM, -1)
      q:take_items(player, RED_GEM, -1)
      q:add_exp_and_sp(player, 0, 42000)
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
    htmltext = player:level() < 25 and "30333-0b.htm" or "30333-0a.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == PRAGA then
      htmltext = "30333-2.htm"
    elseif sw2 == ROHMER then
      if st:is_cond(1) then
        htmltext = q:has_quest_items(player, PATROL_REPORT) and "30344-0.htm" or "30344-0a.htm"
      else
        htmltext = "30344-2.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 2, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  local sw = npc:id()
  if sw == 20922 then
    if rnd(100) < 40 then
      q:give_items(player, WHITE_GEM, 1)
      if q:get_quest_items_count(player, WHITE_GEM) < 50 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif q:get_quest_items_count(player, RED_GEM) >= 50 then
        st:set_cond(3, true)
      end
    end
  elseif sw == 20923 then
    if rnd(100) < 45 then
      q:give_items(player, WHITE_GEM, 1)
      if q:get_quest_items_count(player, WHITE_GEM) < 50 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif q:get_quest_items_count(player, RED_GEM) >= 50 then
        st:set_cond(3, true)
      end
    end
  elseif sw == 20924 then
    if rnd(100) < 35 then
      q:give_items(player, WHITE_GEM, 1)
      if q:get_quest_items_count(player, WHITE_GEM) < 50 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif q:get_quest_items_count(player, RED_GEM) >= 50 then
        st:set_cond(3, true)
      end
    end
  elseif sw == 20926 or sw == 20927 then
    if rnd(100) < 40 then
      q:give_items(player, RED_GEM, 1)
      if q:get_quest_items_count(player, RED_GEM) < 50 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif q:get_quest_items_count(player, WHITE_GEM) >= 50 then
        st:set_cond(3, true)
      end
    end
  end
end)
