-- Translated by questport from quests/Q00103_SpiritOfCraftsman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(103, "Q00103_SpiritOfCraftsman", "Spirit of Craftsman")

-- NPCs
local KARROD = 30307
local CECKTINON = 30132
local HARNE = 30144
-- Items
local KARROD_LETTER = 968
local CECKTINON_VOUCHER_1 = 969
local CECKTINON_VOUCHER_2 = 970
local SOUL_CATCHER = 971
local PRESERVING_OIL = 972
local ZOMBIE_HEAD = 973
local STEELBENDER_HEAD = 974
local BONE_FRAGMENT = 1107
-- Rewards
local SPIRITSHOT_NO_GRADE = 2509
local SOULSHOT_NO_GRADE = 1835
local BLOODSABER = 975
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local LESSER_HEALING_POT = 1060
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416

q:register_quest_items(KARROD_LETTER, CECKTINON_VOUCHER_1, CECKTINON_VOUCHER_2, BONE_FRAGMENT, SOUL_CATCHER, PRESERVING_OIL, ZOMBIE_HEAD, STEELBENDER_HEAD)
q:add_start_npc(KARROD)
q:add_talk_id(KARROD, CECKTINON, HARNE)
q:add_kill_id(20015, 20020, 20455, 20517, 20518)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30307-05.htm" then
    st:start_quest()
    q:give_items(player, KARROD_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30307-00.htm"
    elseif player:level() < 11 then
      htmltext = "30307-02.htm"
    else
      htmltext = "30307-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == KARROD then
      if cond < 8 then
        htmltext = "30307-06.htm"
      elseif cond == 8 then
        htmltext = "30307-07.htm"
        q:take_items(player, STEELBENDER_HEAD, 1)
        q:give_items(player, BLOODSABER, 1)
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
    elseif sw2 == CECKTINON then
      if cond == 1 then
        htmltext = "30132-01.htm"
        st:set_cond(2, true)
        q:take_items(player, KARROD_LETTER, 1)
        q:give_items(player, CECKTINON_VOUCHER_1, 1)
      elseif cond > 1 and cond < 5 then
        htmltext = "30132-02.htm"
      elseif cond == 5 then
        htmltext = "30132-03.htm"
        st:set_cond(6, true)
        q:take_items(player, SOUL_CATCHER, 1)
        q:give_items(player, PRESERVING_OIL, 1)
      elseif cond == 6 then
        htmltext = "30132-04.htm"
      elseif cond == 7 then
        htmltext = "30132-05.htm"
        st:set_cond(8, true)
        q:take_items(player, ZOMBIE_HEAD, 1)
        q:give_items(player, STEELBENDER_HEAD, 1)
      elseif cond == 8 then
        htmltext = "30132-06.htm"
      end
    elseif sw2 == HARNE then
      if cond == 2 then
        htmltext = "30144-01.htm"
        st:set_cond(3, true)
        q:take_items(player, CECKTINON_VOUCHER_1, 1)
        q:give_items(player, CECKTINON_VOUCHER_2, 1)
      elseif cond == 3 then
        htmltext = "30144-02.htm"
      elseif cond == 4 then
        htmltext = "30144-03.htm"
        st:set_cond(5, true)
        q:take_items(player, CECKTINON_VOUCHER_2, 1)
        q:take_items(player, BONE_FRAGMENT, 10)
        q:give_items(player, SOUL_CATCHER, 1)
      elseif cond == 5 then
        htmltext = "30144-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, -1, 3, npc)
  if qs == nil then
    return
  end
  -- The helper picks the party member that gets the drop, so every step below has to address that same player.
  local player = qs:get_player()
  local sw = npc:id()
  if sw == 20015 or sw == 20020 then
    if q:has_quest_items(player, PRESERVING_OIL) and rnd(10) < 5 then
      q:give_items(player, ZOMBIE_HEAD, 1)
      q:take_items(player, PRESERVING_OIL, -1)
      qs:set_cond(7, true)
    end
  elseif sw == 20517 or sw == 20518 or sw == 20455 then
    if q:has_quest_items(player, CECKTINON_VOUCHER_2) and q:give_item_randomly(player, npc, BONE_FRAGMENT, 1, 1, 10, 1, true) then
      qs:set_cond(4, true)
    end
  end
end)
