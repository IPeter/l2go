-- Translated by questport from quests/Q00340_SubjugationOfLizardmen (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(340, "Q00340_SubjugationOfLizardmen", "Subjugation of Lizardmen")

-- NPCs
local WEISZ = 30385
local ADONIUS = 30375
local LEVIAN = 30037
local CHEST = 30989
-- Items
local CARGO = 4255
local HOLY = 4256
local ROSARY = 4257
local TOTEM = 4258

q:register_quest_items(CARGO, HOLY, ROSARY, TOTEM)
q:add_start_npc(WEISZ)
q:add_talk_id(WEISZ, ADONIUS, LEVIAN, CHEST)
q:add_kill_id(20008, 20010, 20014, 20024, 20027, 20030, 25146)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30385-03.htm" then
    st:start_quest()
  elseif event == "30385-07.htm" then
    st:set_cond(2, true)
    q:take_items(player, CARGO, -1)
  elseif event == "30385-09.htm" then
    q:take_items(player, CARGO, -1)
    q:give_adena(player, 4090, true)
  elseif event == "30385-10.htm" then
    q:take_items(player, CARGO, -1)
    q:give_adena(player, 4090, true)
    st:exit_quest(true)
  elseif event == "30375-02.htm" then
    st:set_cond(3, true)
  elseif event == "30037-02.htm" then
    st:set_cond(5, true)
  elseif event == "30989-02.htm" then
    st:set_cond(6, true)
    q:give_items(player, TOTEM, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 17 and "30385-01.htm" or "30385-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == WEISZ then
      if cond == 1 then
        htmltext = q:get_quest_items_count(player, CARGO) < 30 and "30385-05.htm" or "30385-06.htm"
      elseif cond == 2 then
        htmltext = "30385-11.htm"
      elseif cond == 7 then
        htmltext = "30385-13.htm"
        q:give_adena(player, 14700, true)
        st:exit_quest(false, true)
      end
    elseif sw2 == ADONIUS then
      if cond == 2 then
        htmltext = "30375-01.htm"
      elseif cond == 3 then
        if q:has_quest_items(player, ROSARY, HOLY) then
          htmltext = "30375-04.htm"
          st:set_cond(4, true)
          q:take_items(player, HOLY, -1)
          q:take_items(player, ROSARY, -1)
        else
          htmltext = "30375-03.htm"
        end
      elseif cond == 4 then
        htmltext = "30375-05.htm"
      end
    elseif sw2 == LEVIAN then
      if cond == 4 then
        htmltext = "30037-01.htm"
      elseif cond == 5 then
        htmltext = "30037-03.htm"
      elseif cond == 6 then
        htmltext = "30037-04.htm"
        st:set_cond(7, true)
        q:take_items(player, TOTEM, -1)
      elseif cond == 7 then
        htmltext = "30037-05.htm"
      end
    elseif sw2 == CHEST then
      if cond == 5 then
        htmltext = "30989-01.htm"
      else
        htmltext = "30989-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == 20008 then
    if st:is_cond(1) and q:get_quest_items_count(player, CARGO) < 30 and rnd(100) < 50 then
      q:give_items(player, CARGO, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20010 then
    if st:is_cond(1) and q:get_quest_items_count(player, CARGO) < 30 and rnd(100) < 52 then
      q:give_items(player, CARGO, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20014 then
    if st:is_cond(1) and q:get_quest_items_count(player, CARGO) < 30 and rnd(100) < 55 then
      q:give_items(player, CARGO, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20024 or sw == 20027 or sw == 20030 then
    if st:is_cond(3) then
      -- The rosary only starts dropping once the holy symbol is held, and never replaces it.
      if not q:has_quest_items(player, HOLY) and rnd(100) < 10 then
        q:give_items(player, HOLY, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif q:has_quest_items(player, HOLY) and not q:has_quest_items(player, ROSARY) and rnd(100) < 10 then
        q:give_items(player, ROSARY, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 25146 then
    q:add_spawn(CHEST, npc, false, 30000)
  end
end)
