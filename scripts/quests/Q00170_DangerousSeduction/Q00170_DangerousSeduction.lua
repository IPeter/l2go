-- Translated by questport from quests/Q00170_DangerousSeduction (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(170, "Q00170_DangerousSeduction", "Dangerous Seduction")

-- Item
local NIGHTMARE_CRYSTAL = 1046

q:register_quest_items(NIGHTMARE_CRYSTAL)
q:add_start_npc(30305) -- Vellior
q:add_talk_id(30305)
q:add_kill_id(27022) -- Merkenis

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30305-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30305-00.htm"
    elseif player:level() < 21 then
      htmltext = "30305-02.htm"
    else
      htmltext = "30305-03.htm"
    end
  elseif sw == State.STARTED then
    if q:has_quest_items(player, NIGHTMARE_CRYSTAL) then
      htmltext = "30305-06.htm"
      q:take_items(player, NIGHTMARE_CRYSTAL, -1)
      q:give_adena(player, 102680, true)
      st:exit_quest(false, true)
    else
      htmltext = "30305-05.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(1) then
    qs:set_cond(2, true)
    q:give_items(player, NIGHTMARE_CRYSTAL, 1)
    npc:broadcast_say(ChatType.NPC_GENERAL, "Send my soul to Lich King Icarus...")
  end
end)
