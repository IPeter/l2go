-- Translated by questport from quests/Q00119_LastImperialPrince (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(119, "Q00119_LastImperialPrince", "Last Imperial Prince")

-- NPCs
local NAMELESS_SPIRIT = 31453
local DEVORIN = 32009
-- Item
local ANTIQUE_BROOCH = 7262

q:register_quest_items(ANTIQUE_BROOCH)
q:add_start_npc(NAMELESS_SPIRIT)
q:add_talk_id(NAMELESS_SPIRIT, DEVORIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31453-04.htm" then
    if q:has_quest_items(player, ANTIQUE_BROOCH) then
      st:start_quest()
    else
      htmltext = "31453-04b.htm"
      st:exit_quest(true)
    end
  elseif event == "32009-02.htm" then
    if not q:has_quest_items(player, ANTIQUE_BROOCH) then
      htmltext = "31453-02a.htm"
      st:exit_quest(true)
    end
  elseif event == "32009-03.htm" then
    st:set_cond(2, true)
  elseif event == "31453-07.htm" then
    q:give_adena(player, 68787, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (not q:has_quest_items(player, ANTIQUE_BROOCH) or player:level() < 74) and "31453-00a.htm" or "31453-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == NAMELESS_SPIRIT then
      if cond == 1 then
        htmltext = "31453-04a.htm"
      elseif cond == 2 then
        htmltext = "31453-05.htm"
      end
    elseif sw2 == DEVORIN then
      if cond == 1 then
        htmltext = "32009-01.htm"
      elseif cond == 2 then
        htmltext = "32009-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = "31453-00b.htm"
  end
  return htmltext
end)
