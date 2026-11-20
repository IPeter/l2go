-- Translated by questport from quests/Q00418_PathOfTheArtisan (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(418, "Q00418_PathOfTheArtisan", "Path to an Artisan")

-- NPCs
local SILVERA = 30527
local KLUTO = 30317
local PINTER = 30298
local OBI = 32052
local HITCHI = 31963
local LOCKIRIN = 30531
local RYDEL = 31956
-- Items
local SILVERA_RING = 1632
local FIRST_PASS_CERTIFICATE = 1633
local SECOND_PASS_CERTIFICATE = 1634
local FINAL_PASS_CERTIFICATE = 1635
local BOOGLE_RATMAN_TOOTH = 1636
local BOOGLE_RATMAN_LEADER_TOOTH = 1637
local KLUTO_LETTER = 1638
local FOOTPRINT_OF_THIEF = 1639
local STOLEN_SECRET_BOX = 1640
local SECRET_BOX = 1641

q:register_quest_items(SILVERA_RING, FIRST_PASS_CERTIFICATE, SECOND_PASS_CERTIFICATE, BOOGLE_RATMAN_TOOTH, BOOGLE_RATMAN_LEADER_TOOTH, KLUTO_LETTER, FOOTPRINT_OF_THIEF, STOLEN_SECRET_BOX, SECRET_BOX)
q:add_start_npc(SILVERA)
q:add_talk_id(SILVERA, KLUTO, PINTER, OBI, HITCHI, LOCKIRIN, RYDEL)
q:add_kill_id(20389, 20390, 20017)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30527-05.htm" then
    if player:class_id() ~= ClassId.DWARVEN_FIGHTER then
      htmltext = player:class_id() == ClassId.ARTISAN and "30527-02a.htm" or "30527-02.htm"
    elseif player:level() < 19 then
      htmltext = "30527-03.htm"
    elseif q:has_quest_items(player, FINAL_PASS_CERTIFICATE) then
      htmltext = "30527-04.htm"
    end
  elseif event == "30527-06.htm" then
    st:start_quest()
    q:give_items(player, SILVERA_RING, 1)
  elseif event == "30527-08a.htm" then
    st:set_cond(3, true)
    q:take_items(player, BOOGLE_RATMAN_LEADER_TOOTH, -1)
    q:take_items(player, BOOGLE_RATMAN_TOOTH, -1)
    q:take_items(player, SILVERA_RING, 1)
    q:give_items(player, FIRST_PASS_CERTIFICATE, 1)
  elseif event == "30527-08b.htm" then
    st:set_cond(8, true)
    q:take_items(player, BOOGLE_RATMAN_LEADER_TOOTH, -1)
    q:take_items(player, BOOGLE_RATMAN_TOOTH, -1)
    q:take_items(player, SILVERA_RING, 1)
  elseif event == "30317-04.htm" or event == "30317-07.htm" then
    st:set_cond(4, true)
    q:give_items(player, KLUTO_LETTER, 1)
  elseif event == "30317-10.htm" then
    q:take_items(player, FIRST_PASS_CERTIFICATE, 1)
    q:take_items(player, SECOND_PASS_CERTIFICATE, 1)
    q:take_items(player, SECRET_BOX, 1)
    q:give_items(player, FINAL_PASS_CERTIFICATE, 1)
    q:add_exp_and_sp(player, 3200, 6980)
    player:social_action(3)
    st:exit_quest(true, true)
  elseif event == "30317-12.htm" or event == "30531-05.htm" or event == "32052-11.htm" or event == "31963-10.htm" or event == "31956-04.htm" then
    q:take_items(player, FIRST_PASS_CERTIFICATE, 1)
    q:take_items(player, SECOND_PASS_CERTIFICATE, 1)
    q:take_items(player, SECRET_BOX, 1)
    q:give_items(player, FINAL_PASS_CERTIFICATE, 1)
    q:add_exp_and_sp(player, 3200, 3490)
    player:social_action(3)
    st:exit_quest(true, true)
  elseif event == "30298-03.htm" then
    st:set_cond(5, true)
    q:take_items(player, KLUTO_LETTER, -1)
    q:give_items(player, FOOTPRINT_OF_THIEF, 1)
  elseif event == "30298-06.htm" then
    st:set_cond(7, true)
    q:take_items(player, FOOTPRINT_OF_THIEF, -1)
    q:take_items(player, STOLEN_SECRET_BOX, -1)
    q:give_items(player, SECOND_PASS_CERTIFICATE, 1)
    q:give_items(player, SECRET_BOX, 1)
  elseif event == "32052-06.htm" then
    st:set_cond(9, true)
  elseif event == "31963-04.htm" then
    st:set_cond(10, true)
  elseif event == "31963-05.htm" then
    st:set_cond(11, true)
  elseif event == "31963-07.htm" then
    st:set_cond(12, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30527-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SILVERA then
      if cond == 1 then
        htmltext = "30527-07.htm"
      elseif cond == 2 then
        htmltext = "30527-08.htm"
      elseif cond == 3 then
        htmltext = "30527-09.htm"
      elseif cond == 8 then
        htmltext = "30527-09a.htm"
      end
    elseif sw2 == KLUTO then
      if cond == 3 then
        htmltext = "30317-01.htm"
      elseif cond == 4 then
        htmltext = "30317-08.htm"
      elseif cond == 7 then
        htmltext = "30317-09.htm"
      end
    elseif sw2 == PINTER then
      if cond == 4 then
        htmltext = "30298-01.htm"
      elseif cond == 5 then
        htmltext = "30298-04.htm"
      elseif cond == 6 then
        htmltext = "30298-05.htm"
      elseif cond == 7 then
        htmltext = "30298-07.htm"
      end
    elseif sw2 == OBI then
      if cond == 8 then
        htmltext = "32052-01.htm"
      elseif cond == 9 then
        htmltext = "32052-06a.htm"
      elseif cond == 11 then
        htmltext = "32052-07.htm"
      end
    elseif sw2 == HITCHI then
      if cond == 9 then
        htmltext = "31963-01.htm"
      elseif cond == 10 then
        htmltext = "31963-04.htm"
      elseif cond == 11 then
        htmltext = "31963-06a.htm"
      elseif cond == 12 then
        htmltext = "31963-08.htm"
      end
    elseif sw2 == LOCKIRIN then
      if cond == 10 then
        htmltext = "30531-01.htm"
      end
    elseif sw2 == RYDEL then
      if cond == 12 then
        htmltext = "31956-01.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == 20389 then
    if st:is_cond(1) and rnd(10) < 7 then
      if q:get_quest_items_count(player, BOOGLE_RATMAN_TOOTH) < 10 then
        q:give_items(player, BOOGLE_RATMAN_TOOTH, 1)
      end
      if q:get_quest_items_count(player, BOOGLE_RATMAN_TOOTH) >= 10 and q:get_quest_items_count(player, BOOGLE_RATMAN_LEADER_TOOTH) >= 2 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20390 then
    if st:is_cond(1) and rnd(10) < 5 then
      if q:get_quest_items_count(player, BOOGLE_RATMAN_LEADER_TOOTH) < 2 then
        q:give_items(player, BOOGLE_RATMAN_LEADER_TOOTH, 1)
      end
      if q:get_quest_items_count(player, BOOGLE_RATMAN_LEADER_TOOTH) >= 2 and q:get_quest_items_count(player, BOOGLE_RATMAN_TOOTH) >= 10 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20017 then
    if st:is_cond(5) and rnd(10) < 2 then
      q:give_items(player, STOLEN_SECRET_BOX, 1)
      st:set_cond(6, true)
    end
  end
end)
