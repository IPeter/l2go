-- Translated by questport from quests/Q00637_ThroughOnceMore (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(637, "Q00637_ThroughOnceMore", "Through the Gate Once More")

-- NPC
local FLAURON = 32010
-- Items
local FADED_VISITOR_MARK = 8065
local NECROMANCER_HEART = 8066
-- Reward
local PAGAN_MARK = 8067

q:register_quest_items(NECROMANCER_HEART)
q:add_start_npc(FLAURON)
q:add_talk_id(FLAURON)
q:add_kill_id(21565, 21566, 21567)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32010-04.htm" then
    st:start_quest()
  elseif event == "32010-10.htm" then
    st:exit_quest(true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 73 or not q:has_quest_items(player, FADED_VISITOR_MARK) then
      htmltext = "32010-01a.htm"
    elseif q:has_quest_items(player, PAGAN_MARK) then
      htmltext = "32010-00.htm"
    else
      htmltext = "32010-01.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(2) then
      if q:get_quest_items_count(player, NECROMANCER_HEART) == 10 then
        htmltext = "32010-06.htm"
        q:take_items(player, FADED_VISITOR_MARK, 1)
        q:take_items(player, NECROMANCER_HEART, -1)
        q:give_items(player, PAGAN_MARK, 1)
        q:give_items(player, 8273, 10)
        st:exit_quest(true, true)
      else
        st:set_cond(1)
      end
    else
      htmltext = "32010-05.htm"
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
  if rnd(10) < 4 then
    q:give_items(partyMember, NECROMANCER_HEART, 1)
    if q:get_quest_items_count(partyMember, NECROMANCER_HEART) < 10 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
