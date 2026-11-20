-- Translated by questport from quests/Q00127_KamaelAWindowToTheFuture (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(127, "Q00127_KamaelAWindowToTheFuture", "Kamael: A Window to the Future")

-- NPCs
local DOMINIC = 31350
local KLAUS = 30187
local ALDER = 32092
local AKLAN = 31288
local OLTLIN = 30862
local JURIS = 30113
local RODEMAI = 30756
-- Items
local MARK_DOMINIC = 8939
local MARK_HUMAN = 8940
local MARK_DWARF = 8941
local MARK_ORC = 8944
local MARK_DELF = 8943
local MARK_ELF = 8942

q:register_quest_items(MARK_DOMINIC, MARK_HUMAN, MARK_DWARF, MARK_ORC, MARK_DELF, MARK_ELF)
q:add_start_npc(DOMINIC)
q:add_talk_id(DOMINIC, KLAUS, ALDER, AKLAN, OLTLIN, JURIS, RODEMAI)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  if event == "31350-04.htm" then
    st:start_quest()
    q:give_items(player, MARK_DOMINIC, 1)
  elseif event == "31350-06.htm" then
    q:take_items(player, MARK_HUMAN, -1)
    q:take_items(player, MARK_DWARF, -1)
    q:take_items(player, MARK_ELF, -1)
    q:take_items(player, MARK_DELF, -1)
    q:take_items(player, MARK_ORC, -1)
    q:take_items(player, MARK_DOMINIC, -1)
    q:give_adena(player, 159100, true)
    st:exit_quest(false, true)
  elseif event == "30187-06.htm" then
    st:set_cond(2)
  elseif event == "30187-08.htm" then
    st:set_cond(3, true)
    q:give_items(player, MARK_HUMAN, 1)
  elseif event == "32092-05.htm" then
    st:set_cond(4, true)
    q:give_items(player, MARK_DWARF, 1)
  elseif event == "31288-04.htm" then
    st:set_cond(5, true)
    q:give_items(player, MARK_ORC, 1)
  elseif event == "30862-04.htm" then
    st:set_cond(6, true)
    q:give_items(player, MARK_DELF, 1)
  elseif event == "30113-04.htm" then
    st:set_cond(7, true)
    q:give_items(player, MARK_ELF, 1)
  elseif event == "kamaelstory" then
    st:set_cond(8, true)
    -- Mobius: player.sendPacket(ExShowSlideshowKamael.STATIC_PACKET). The
    -- slideshow is left out until the API has player:show_slideshow_kamael.
    return nil
  elseif event == "30756-05.htm" then
    st:set_cond(9, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "31350-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == KLAUS then
      if st:is_cond(1) then
        htmltext = "30187-01.htm"
      elseif st:is_cond(2) then
        htmltext = "30187-06.htm"
      end
    elseif sw2 == ALDER then
      if st:is_cond(3) then
        htmltext = "32092-01.htm"
      end
    elseif sw2 == AKLAN then
      if st:is_cond(4) then
        htmltext = "31288-01.htm"
      end
    elseif sw2 == OLTLIN then
      if st:is_cond(5) then
        htmltext = "30862-01.htm"
      end
    elseif sw2 == JURIS then
      if st:is_cond(6) then
        htmltext = "30113-01.htm"
      end
    elseif sw2 == RODEMAI then
      if st:is_cond(7) then
        htmltext = "30756-01.htm"
      elseif st:is_cond(8) then
        htmltext = "30756-04.htm"
      end
    elseif sw2 == DOMINIC then
      if st:is_cond(9) then
        htmltext = "31350-05.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
    return htmltext
  end
  return htmltext
end)
