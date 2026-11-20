-- Translated by questport from quests/Q00101_SwordOfSolidarity (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(101, "Q00101_SwordOfSolidarity", "Sword of Solidarity")

-- NPCs
local ROIEN = 30008
local ALTRAN = 30283
-- Items
local BROKEN_SWORD_HANDLE = 739
local BROKEN_BLADE_BOTTOM = 740
local BROKEN_BLADE_TOP = 741
local ROIENS_LETTER = 796
local DIR_TO_RUINS = 937
local ALTRANS_NOTE = 742
local SWORD_OF_SOLIDARITY = 738
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local LESSER_HEALING_POT = 1060
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416

q:register_quest_items(BROKEN_SWORD_HANDLE, BROKEN_BLADE_BOTTOM, BROKEN_BLADE_TOP)
q:add_start_npc(ROIEN)
q:add_talk_id(ROIEN, ALTRAN)
q:add_kill_id(20361, 20362)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30008-03.htm" then
    st:start_quest()
    q:give_items(player, ROIENS_LETTER, 1)
  elseif event == "30283-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, ROIENS_LETTER, 1)
    q:give_items(player, DIR_TO_RUINS, 1)
  elseif event == "30283-06.htm" then
    q:take_items(player, BROKEN_SWORD_HANDLE, 1)
    q:give_items(player, SWORD_OF_SOLIDARITY, 1)
    q:give_items(player, LESSER_HEALING_POT, 100)
    -- Give newbie reward if player is eligible
    if player:is_newbie() then
      local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
      if newPlayerRewardsReceived < 2 then
        st:show_question_mark(26)
        if player:is_mage_class() then
          st:play_tutorial_voice("tutorial_voice_027")
          q:give_items(player, SPIRITSHOT_FOR_BEGINNERS, 3000)
          newPlayerRewardsReceived = newPlayerRewardsReceived + 1
          player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
        else
          st:play_tutorial_voice("tutorial_voice_026")
          q:give_items(player, SOULSHOT_FOR_BEGINNERS, 7000)
          newPlayerRewardsReceived = newPlayerRewardsReceived + 1
          player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
        end
      end
    end
    q:give_items(player, ECHO_BATTLE, 10)
    q:give_items(player, ECHO_LOVE, 10)
    q:give_items(player, ECHO_SOLITUDE, 10)
    q:give_items(player, ECHO_FEAST, 10)
    q:give_items(player, ECHO_CELEBRATION, 10)
    player:social_action(3)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.HUMAN then
      htmltext = "30008-01a.htm"
    elseif player:level() < 9 then
      htmltext = "30008-01.htm"
    else
      htmltext = "30008-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ROIEN then
      if cond == 1 then
        htmltext = "30008-04.htm"
      elseif cond == 2 then
        htmltext = "30008-03a.htm"
      elseif cond == 3 then
        htmltext = "30008-06.htm"
      elseif cond == 4 then
        htmltext = "30008-05.htm"
        st:set_cond(5, true)
        q:take_items(player, ALTRANS_NOTE, 1)
        q:give_items(player, BROKEN_SWORD_HANDLE, 1)
      elseif cond == 5 then
        htmltext = "30008-05a.htm"
      end
    elseif sw2 == ALTRAN then
      if cond == 1 then
        htmltext = "30283-01.htm"
      elseif cond == 2 then
        htmltext = "30283-03.htm"
      elseif cond == 3 then
        htmltext = "30283-04.htm"
        st:set_cond(4, true)
        q:take_items(player, DIR_TO_RUINS, 1)
        q:take_items(player, BROKEN_BLADE_TOP, 1)
        q:take_items(player, BROKEN_BLADE_BOTTOM, 1)
        q:give_items(player, ALTRANS_NOTE, 1)
      elseif cond == 4 then
        htmltext = "30283-04a.htm"
      elseif cond == 5 then
        htmltext = "30283-05.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(2) and rnd(5) == 0 then
    if not q:has_quest_items(killer, BROKEN_BLADE_TOP) then
      q:give_items(killer, BROKEN_BLADE_TOP, 1)
      if q:has_quest_items(killer, BROKEN_BLADE_BOTTOM) then
        qs:set_cond(3, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif not q:has_quest_items(killer, BROKEN_BLADE_BOTTOM) then
      q:give_items(killer, BROKEN_BLADE_BOTTOM, 1)
      if q:has_quest_items(killer, BROKEN_BLADE_TOP) then
        qs:set_cond(3, true)
      else
        q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
