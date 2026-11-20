-- Translated by questport from quests/Q00105_SkirmishWithOrcs (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(105, "Q00105_SkirmishWithOrcs", "Skirmish with the Orcs")

-- Monster
local KABOO_CHIEF_UOPH = 27059
local KABOO_CHIEF_KRACHA = 27060
local KABOO_CHIEF_BATOH = 27061
local KABOO_CHIEF_TANUKIA = 27062
local KABOO_CHIEF_TUREL = 27064
local KABOO_CHIEF_ROKO = 27065
local KABOO_CHIEF_KAMUT = 27067
local KABOO_CHIEF_MURTIKA = 27068
-- Item
local KENDELL_ORDER_1 = 1836
local KENDELL_ORDER_2 = 1837
local KENDELL_ORDER_3 = 1838
local KENDELL_ORDER_4 = 1839
local KENDELL_ORDER_5 = 1840
local KENDELL_ORDER_6 = 1841
local KENDELL_ORDER_7 = 1842
local KENDELL_ORDER_8 = 1843
local KABOO_CHIEF_TORC_1 = 1844
local KABOO_CHIEF_TORC_2 = 1845
-- Rewards
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local RED_SUNSET_STAFF = 754
local RED_SUNSET_SWORD = 981
local LESSER_HEALING_POT = 1060
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416

q:register_quest_items(KENDELL_ORDER_1, KENDELL_ORDER_2, KENDELL_ORDER_3, KENDELL_ORDER_4, KENDELL_ORDER_5, KENDELL_ORDER_6, KENDELL_ORDER_7, KENDELL_ORDER_8, KABOO_CHIEF_TORC_1, KABOO_CHIEF_TORC_2)
q:add_start_npc(30218) -- Kendell
q:add_talk_id(30218)
q:add_kill_id(KABOO_CHIEF_UOPH, KABOO_CHIEF_KRACHA, KABOO_CHIEF_BATOH, KABOO_CHIEF_TANUKIA, KABOO_CHIEF_TUREL, KABOO_CHIEF_ROKO, KABOO_CHIEF_KAMUT, KABOO_CHIEF_MURTIKA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30218-03.htm" then
    st:start_quest()
    q:give_items(player, rnd(1836, 1839), 1) -- Kendell's orders 1 to 4.
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30218-00.htm"
    elseif player:level() < 10 then
      htmltext = "30218-01.htm"
    else
      htmltext = "30218-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30218-05.htm"
    elseif cond == 2 then
      htmltext = "30218-06.htm"
      st:set_cond(3, true)
      q:take_items(player, KABOO_CHIEF_TORC_1, 1)
      q:take_items(player, KENDELL_ORDER_1, 1)
      q:take_items(player, KENDELL_ORDER_2, 1)
      q:take_items(player, KENDELL_ORDER_3, 1)
      q:take_items(player, KENDELL_ORDER_4, 1)
      q:give_items(player, rnd(1840, 1843), 1) -- Kendell's orders 5 to 8.
    elseif cond == 3 then
      htmltext = "30218-07.htm"
    elseif cond == 4 then
      htmltext = "30218-08.htm"
      q:take_items(player, KABOO_CHIEF_TORC_2, 1)
      q:take_items(player, KENDELL_ORDER_5, 1)
      q:take_items(player, KENDELL_ORDER_6, 1)
      q:take_items(player, KENDELL_ORDER_7, 1)
      q:take_items(player, KENDELL_ORDER_8, 1)
      if player:is_mage_class() then
        q:give_items(player, RED_SUNSET_STAFF, 1)
      else
        q:give_items(player, RED_SUNSET_SWORD, 1)
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
      q:reward_items(player, LESSER_HEALING_POT, 100)
      q:give_items(player, ECHO_BATTLE, 10)
      q:give_items(player, ECHO_LOVE, 10)
      q:give_items(player, ECHO_SOLITUDE, 10)
      q:give_items(player, ECHO_FEAST, 10)
      q:give_items(player, ECHO_CELEBRATION, 10)
      player:social_action(3)
      st:exit_quest(false, true)
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
  if sw == KABOO_CHIEF_UOPH or sw == KABOO_CHIEF_KRACHA or sw == KABOO_CHIEF_BATOH or sw == KABOO_CHIEF_TANUKIA then
    if st:is_cond(1) and q:has_quest_items(player, npc:id() - 25223) then
      st:set_cond(2, true)
      q:give_items(player, KABOO_CHIEF_TORC_1, 1)
    end
  elseif sw == KABOO_CHIEF_TUREL or sw == KABOO_CHIEF_ROKO then
    if st:is_cond(3) and q:has_quest_items(player, npc:id() - 25224) then
      st:set_cond(4, true)
      q:give_items(player, KABOO_CHIEF_TORC_2, 1)
    end
  elseif sw == KABOO_CHIEF_KAMUT or sw == KABOO_CHIEF_MURTIKA then
    if st:is_cond(3) and q:has_quest_items(player, npc:id() - 25225) then
      st:set_cond(4, true)
      q:give_items(player, KABOO_CHIEF_TORC_2, 1)
    end
  end
end)
