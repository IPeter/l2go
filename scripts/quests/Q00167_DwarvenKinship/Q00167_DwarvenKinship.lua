-- Translated by questport from quests/Q00167_DwarvenKinship (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(167, "Q00167_DwarvenKinship", "Dwarven Kinship")

-- NPCs
local CARLON = 30350
local NORMAN = 30210
local HAPROCK = 30255
-- Items
local CARLON_LETTER = 1076
local NORMAN_LETTER = 1106

q:register_quest_items(CARLON_LETTER, NORMAN_LETTER)
q:add_start_npc(CARLON)
q:add_talk_id(CARLON, HAPROCK, NORMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30350-04.htm" then
    st:start_quest()
    q:give_items(player, CARLON_LETTER, 1)
  elseif event == "30255-03.htm" then
    st:set_cond(2)
    q:take_items(player, CARLON_LETTER, 1)
    q:give_items(player, NORMAN_LETTER, 1)
    q:give_adena(player, 2000, true)
  elseif event == "30255-04.htm" then
    q:take_items(player, CARLON_LETTER, 1)
    q:give_adena(player, 3000, true)
    st:exit_quest(false, true)
  elseif event == "30210-02.htm" then
    q:take_items(player, NORMAN_LETTER, 1)
    q:give_adena(player, 20000, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30350-02.htm" or "30350-03.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == CARLON then
      if cond == 1 then
        htmltext = "30350-05.htm"
      end
    elseif sw2 == HAPROCK then
      if cond == 1 then
        htmltext = "30255-01.htm"
      elseif cond == 2 then
        htmltext = "30255-05.htm"
      end
    elseif sw2 == NORMAN then
      if cond == 2 then
        htmltext = "30210-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
