-- Translated by questport from quests/Q00107_MercilessPunishment (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(107, "Q00107_MercilessPunishment", "Merciless Punishment")

-- NPCs
local HATOS = 30568
local PARUGON = 30580
-- Items
local HATOS_ORDER_1 = 1553
local HATOS_ORDER_2 = 1554
local HATOS_ORDER_3 = 1555
local LETTER_TO_HUMAN = 1557
local LETTER_TO_DARKELF = 1556
local LETTER_TO_ELF = 1558
-- Rewards
local BUTCHER_SWORD = 1510
local SOULSHOT_FOR_BEGINNERS = 5789
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416
local LESSER_HEALING_POTION = 1060

q:register_quest_items(HATOS_ORDER_1, HATOS_ORDER_2, HATOS_ORDER_3, LETTER_TO_HUMAN, LETTER_TO_DARKELF, LETTER_TO_ELF)
q:add_start_npc(HATOS)
q:add_talk_id(HATOS, PARUGON)
q:add_kill_id(27041) -- Baranka's Messenger

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  local htmltext = event
  if st == nil then
    return htmltext
  end
  if event == "30568-03.htm" then
    st:start_quest()
    q:give_items(player, HATOS_ORDER_1, 1)
  elseif event == "30568-06.htm" then
    st:exit_quest(true, true)
  elseif event == "30568-07.htm" then
    st:set_cond(4, true)
    q:take_items(player, HATOS_ORDER_1, 1)
    q:give_items(player, HATOS_ORDER_2, 1)
  elseif event == "30568-09.htm" then
    st:set_cond(6, true)
    q:take_items(player, HATOS_ORDER_2, 1)
    q:give_items(player, HATOS_ORDER_3, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30568-00.htm"
    elseif player:level() < 12 then
      htmltext = "30568-01.htm"
    else
      htmltext = "30568-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == HATOS then
      if cond == 1 or cond == 2 then
        htmltext = "30568-04.htm"
      elseif cond == 3 then
        htmltext = "30568-05.htm"
      elseif cond == 4 or cond == 6 then
        htmltext = "30568-09.htm"
      elseif cond == 5 then
        htmltext = "30568-08.htm"
      elseif cond == 7 then
        htmltext = "30568-10.htm"
        q:take_items(player, HATOS_ORDER_3, -1)
        q:take_items(player, LETTER_TO_DARKELF, -1)
        q:take_items(player, LETTER_TO_HUMAN, -1)
        q:take_items(player, LETTER_TO_ELF, -1)
        q:give_items(player, BUTCHER_SWORD, 1)
        q:give_items(player, LESSER_HEALING_POTION, 100)
        -- Give newbie reward if player is eligible
        if player:is_newbie() then
          local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
          if newPlayerRewardsReceived < 2 then
            st:show_question_mark(26)
            st:play_tutorial_voice("tutorial_voice_026")
            q:give_items(player, SOULSHOT_FOR_BEGINNERS, 7000)
            newPlayerRewardsReceived = newPlayerRewardsReceived + 1
            player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
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
    elseif sw2 == PARUGON then
      htmltext = "30580-01.htm"
      if cond == 1 then
        st:set_cond(2, true)
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
  local cond = st:get_cond()
  if cond == 2 then
    st:set_cond(3, true)
    q:give_items(player, LETTER_TO_HUMAN, 1)
  elseif cond == 4 then
    st:set_cond(5, true)
    q:give_items(player, LETTER_TO_DARKELF, 1)
  elseif cond == 6 then
    st:set_cond(7, true)
    q:give_items(player, LETTER_TO_ELF, 1)
  end
end)
