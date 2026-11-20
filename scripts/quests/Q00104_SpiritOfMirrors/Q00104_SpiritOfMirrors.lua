-- Translated by questport from quests/Q00104_SpiritOfMirrors (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(104, "Q00104_SpiritOfMirrors", "Spirit of Mirrors")

-- NPCs
local GALLINT = 30017
local ARNOLD = 30041
local JOHNSTONE = 30043
local KENYOS = 30045
-- Items
local GALLINS_OAK_WAND = 748
local WAND_SPIRITBOUND_1 = 1135
local WAND_SPIRITBOUND_2 = 1136
local WAND_SPIRITBOUND_3 = 1137
-- Rewards
local SPIRITSHOT_NO_GRADE = 2509
local SOULSHOT_NO_GRADE = 1835
local WAND_OF_ADEPT = 747
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local LESSER_HEALING_POT = 1060
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416

q:register_quest_items(GALLINS_OAK_WAND, WAND_SPIRITBOUND_1, WAND_SPIRITBOUND_2, WAND_SPIRITBOUND_3)
q:add_start_npc(GALLINT)
q:add_talk_id(GALLINT, ARNOLD, JOHNSTONE, KENYOS)
q:add_kill_id(27003, 27004, 27005)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30017-03.htm" then
    st:start_quest()
    q:give_items(player, GALLINS_OAK_WAND, 1)
    q:give_items(player, GALLINS_OAK_WAND, 1)
    q:give_items(player, GALLINS_OAK_WAND, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.HUMAN then
      htmltext = "30017-00.htm"
    elseif player:level() < 10 then
      htmltext = "30017-01.htm"
    else
      htmltext = "30017-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GALLINT then
      if cond == 1 or cond == 2 then
        htmltext = "30017-04.htm"
      elseif cond == 3 then
        htmltext = "30017-05.htm"
        q:take_items(player, WAND_SPIRITBOUND_1, -1)
        q:take_items(player, WAND_SPIRITBOUND_2, -1)
        q:take_items(player, WAND_SPIRITBOUND_3, -1)
        q:give_items(player, WAND_OF_ADEPT, 1)
        q:reward_items(player, LESSER_HEALING_POT, 100)
        if player:is_mage_class() then
          q:give_items(player, SPIRITSHOT_NO_GRADE, 500)
        else
          q:give_items(player, SOULSHOT_NO_GRADE, 1000)
        end
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
    elseif sw2 == KENYOS or sw2 == JOHNSTONE or sw2 == ARNOLD then
      htmltext = npc:id() .. "-01.htm"
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
  if q:get_item_equipped(player, Paperdoll.RHAND) == GALLINS_OAK_WAND then
    local sw = npc:id()
    if sw == 27003 then
      if not q:has_quest_items(player, WAND_SPIRITBOUND_1) then
        q:take_items(player, GALLINS_OAK_WAND, 1)
        q:give_items(player, WAND_SPIRITBOUND_1, 1)
        if q:has_quest_items(player, WAND_SPIRITBOUND_2, WAND_SPIRITBOUND_3) then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == 27004 then
      if not q:has_quest_items(player, WAND_SPIRITBOUND_2) then
        q:take_items(player, GALLINS_OAK_WAND, 1)
        q:give_items(player, WAND_SPIRITBOUND_2, 1)
        if q:has_quest_items(player, WAND_SPIRITBOUND_1, WAND_SPIRITBOUND_3) then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == 27005 then
      if not q:has_quest_items(player, WAND_SPIRITBOUND_3) then
        q:take_items(player, GALLINS_OAK_WAND, 1)
        q:give_items(player, WAND_SPIRITBOUND_3, 1)
        if q:has_quest_items(player, WAND_SPIRITBOUND_1, WAND_SPIRITBOUND_2) then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
