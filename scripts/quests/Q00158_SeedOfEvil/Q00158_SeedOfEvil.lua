-- Translated by questport from quests/Q00158_SeedOfEvil (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(158, "Q00158_SeedOfEvil", "Seed of Evil")

-- Item
local CLAY_TABLET = 1025
-- Reward
local ENCHANT_ARMOR_D = 956

q:register_quest_items(CLAY_TABLET)
q:add_start_npc(30031) -- Biotin
q:add_talk_id(30031)
q:add_kill_id(27016) -- Nerkas

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30031-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 21 and "30031-02.htm" or "30031-03.htm"
  elseif sw == State.STARTED then
    if not q:has_quest_items(player, CLAY_TABLET) then
      htmltext = "30031-05.htm"
    else
      htmltext = "30031-06.htm"
      q:take_items(player, CLAY_TABLET, 1)
      q:give_items(player, ENCHANT_ARMOR_D, 1)
      st:exit_quest(false, true)
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and not q:has_quest_items(killer, CLAY_TABLET) then
    q:give_items(killer, CLAY_TABLET, 1)
    qs:set_cond(2, true)
  end
  npc:broadcast_say(ChatType.NPC_GENERAL, "The power of Lord Beleth rules the whole world...!")
end)
