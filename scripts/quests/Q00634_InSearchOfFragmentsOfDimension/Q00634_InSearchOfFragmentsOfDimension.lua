-- Translated by questport from quests/Q00634_InSearchOfFragmentsOfDimension (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(634, "Q00634_InSearchOfFragmentsOfDimension", "In Search of Fragments of Dimension")
-- L2Go: the gatekeepers and the monsters live in the Seven Signs
-- necropolises and catacombs, but the quest itself (talk, kills, fragment
-- drops) uses nothing Seven Signs brings, so it is not held back with
-- q:requires("seven_signs") (unlike Q00635, which enters the Rift).

-- Items
local DIMENSION_FRAGMENT = 7079

-- Dimensional Gate Keepers.
for i = 31494, 31507 do
  q:add_start_npc(i)
  q:add_talk_id(i)
end
-- All mobs.
for i = 21208, 21255 do
  q:add_kill_id(i)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "02.htm" then
    st:start_quest()
  elseif event == "05.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "01a.htm" or "01.htm"
  elseif sw == State.STARTED then
    htmltext = "03.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(100) < 8 then
    local partyMember = st:get_player()
    q:give_items(partyMember, DIMENSION_FRAGMENT, J.int(npc:level() * 0.15 + 2.6))
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
