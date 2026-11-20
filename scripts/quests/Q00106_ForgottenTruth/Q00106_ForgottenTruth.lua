-- Translated by questport from quests/Q00106_ForgottenTruth (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(106, "Q00106_ForgottenTruth", "Forgotten Truth")

-- NPCs
local THIFIELL = 30358
local KARTIA = 30133
-- Items
local ONYX_TALISMAN_1 = 984
local ONYX_TALISMAN_2 = 985
local ANCIENT_SCROLL = 986
local ANCIENT_CLAY_TABLET = 987
local KARTIA_TRANSLATION = 988
-- Rewards
local SPIRITSHOT_NO_GRADE = 2509
local SOULSHOT_NO_GRADE = 1835
local ELDRITCH_DAGGER = 989
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416
local LESSER_HEALING_POTION = 1060

q:register_quest_items(ONYX_TALISMAN_1, ONYX_TALISMAN_2, ANCIENT_SCROLL, ANCIENT_CLAY_TABLET, KARTIA_TRANSLATION)
q:add_start_npc(THIFIELL)
q:add_talk_id(THIFIELL, KARTIA)
q:add_kill_id(27070) -- Tumran Orc Brigand

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  local htmltext = event
  if st == nil then
    return htmltext
  end
  if event == "30358-05.htm" then
    st:start_quest()
    q:give_items(player, ONYX_TALISMAN_1, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30358-00.htm"
    elseif player:level() < 10 then
      htmltext = "30358-02.htm"
    else
      htmltext = "30358-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == THIFIELL then
      if cond == 1 then
        htmltext = "30358-06.htm"
      elseif cond == 2 then
        htmltext = "30358-06.htm"
      elseif cond == 3 then
        htmltext = "30358-06.htm"
      elseif cond == 4 then
        htmltext = "30358-07.htm"
        q:take_items(player, KARTIA_TRANSLATION, 1)
        q:give_items(player, ELDRITCH_DAGGER, 1)
        q:give_items(player, LESSER_HEALING_POTION, 100)
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
    elseif sw2 == KARTIA then
      if cond == 1 then
        htmltext = "30133-01.htm"
        st:set_cond(2, true)
        q:take_items(player, ONYX_TALISMAN_1, 1)
        q:give_items(player, ONYX_TALISMAN_2, 1)
      elseif cond == 2 then
        htmltext = "30133-02.htm"
      elseif cond == 3 then
        htmltext = "30133-03.htm"
        st:set_cond(4, true)
        q:take_items(player, ONYX_TALISMAN_2, 1)
        q:take_items(player, ANCIENT_SCROLL, 1)
        q:take_items(player, ANCIENT_CLAY_TABLET, 1)
        q:give_items(player, KARTIA_TRANSLATION, 1)
      elseif cond == 4 then
        htmltext = "30133-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(2) and util.in_range(Config.ALT_PARTY_RANGE, npc, killer, true) and rnd(100) < 20 and q:has_quest_items(killer, ONYX_TALISMAN_2) then
    if not q:has_quest_items(killer, ANCIENT_SCROLL) then
      q:give_items(killer, ANCIENT_SCROLL, 1)
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    elseif not q:has_quest_items(killer, ANCIENT_CLAY_TABLET) then
      qs:set_cond(3, true)
      q:give_items(killer, ANCIENT_CLAY_TABLET, 1)
    end
  end
end)
