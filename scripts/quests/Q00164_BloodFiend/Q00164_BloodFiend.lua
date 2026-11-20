-- Translated by questport from quests/Q00164_BloodFiend (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(164, "Q00164_BloodFiend", "Blood Fiend")

-- Item
local KIRUNAK_SKULL = 1044

q:register_quest_items(KIRUNAK_SKULL)
q:add_start_npc(30149)
q:add_talk_id(30149)
q:add_kill_id(27021)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30149-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() == Race.DARK_ELF then
      htmltext = "30149-00.htm"
    elseif player:level() < 21 then
      htmltext = "30149-02.htm"
    else
      htmltext = "30149-03.htm"
    end
  elseif sw == State.STARTED then
    if q:has_quest_items(player, KIRUNAK_SKULL) then
      htmltext = "30149-06.htm"
      q:take_items(player, KIRUNAK_SKULL, 1)
      q:give_adena(player, 42130, true)
      st:exit_quest(false, true)
    else
      htmltext = "30149-05.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) then
    npc:broadcast_say(ChatType.GENERAL, "I have fulfilled my contract with Trader Creamees.")
    q:give_items(killer, KIRUNAK_SKULL, 1)
    qs:set_cond(2, true)
  end
end)
