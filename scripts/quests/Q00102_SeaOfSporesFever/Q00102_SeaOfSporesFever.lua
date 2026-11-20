-- Translated by questport from quests/Q00102_SeaOfSporesFever (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(102, "Q00102_SeaOfSporesFever", "Sea of Spores Fever")

-- NPCs
local ALBERIUS = 30284
local COBENDELL = 30156
local BERROS = 30217
local VELTRESS = 30219
local RAYEN = 30221
local GARTRANDELL = 30285
-- Items
local ALBERIUS_LETTER = 964
local EVERGREEN_AMULET = 965
local DRYAD_TEARS = 966
local ALBERIUS_LIST = 746
local COBENDELL_MEDICINE_1 = 1130
local COBENDELL_MEDICINE_2 = 1131
local COBENDELL_MEDICINE_3 = 1132
local COBENDELL_MEDICINE_4 = 1133
local COBENDELL_MEDICINE_5 = 1134
-- Rewards
local SPIRITSHOT_NO_GRADE = 2509
local SOULSHOT_NO_GRADE = 1835
local SWORD_OF_SENTINEL = 743
local STAFF_OF_SENTINEL = 744
local LESSER_HEALING_POT = 1060
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416
-- Newbie Rewards
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789

local function checkItem(st, itemId)
  if q:has_quest_items(st:get_player(), itemId) then
    q:take_items(st:get_player(), itemId, 1)
    local medicinesLeft = st:get_int("medicines") - 1
    if medicinesLeft == 0 then
      st:set_cond(6, true)
    else
      st:set("medicines", tostring(medicinesLeft))
    end
  end
end

q:register_quest_items(ALBERIUS_LETTER, EVERGREEN_AMULET, DRYAD_TEARS, COBENDELL_MEDICINE_1, COBENDELL_MEDICINE_2, COBENDELL_MEDICINE_3, COBENDELL_MEDICINE_4, COBENDELL_MEDICINE_5, ALBERIUS_LIST)
q:add_start_npc(ALBERIUS)
q:add_talk_id(ALBERIUS, COBENDELL, BERROS, RAYEN, GARTRANDELL, VELTRESS)
q:add_kill_id(20013, 20019)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30284-02.htm" then
    st:start_quest()
    q:give_items(player, ALBERIUS_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30284-00.htm"
    elseif player:level() < 12 then
      htmltext = "30284-08.htm"
    else
      htmltext = "30284-07.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ALBERIUS then
      if cond == 1 then
        htmltext = "30284-03.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30284-09.htm"
      elseif cond == 4 then
        htmltext = "30284-04.htm"
        st:set_cond(5, true)
        st:set("medicines", "4")
        q:take_items(player, COBENDELL_MEDICINE_1, 1)
        q:give_items(player, ALBERIUS_LIST, 1)
      elseif cond == 5 then
        htmltext = "30284-05.htm"
      elseif cond == 6 then
        htmltext = "30284-06.htm"
        q:take_items(player, ALBERIUS_LIST, 1)
        if player:is_mage_class() then
          q:give_items(player, STAFF_OF_SENTINEL, 1)
          q:reward_items(player, SPIRITSHOT_NO_GRADE, 500)
        else
          q:give_items(player, SWORD_OF_SENTINEL, 1)
          q:reward_items(player, SOULSHOT_NO_GRADE, 1000)
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
        q:give_items(player, LESSER_HEALING_POT, 100)
        q:give_items(player, ECHO_BATTLE, 10)
        q:give_items(player, ECHO_LOVE, 10)
        q:give_items(player, ECHO_SOLITUDE, 10)
        q:give_items(player, ECHO_FEAST, 10)
        q:give_items(player, ECHO_CELEBRATION, 10)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == COBENDELL then
      if cond == 1 then
        htmltext = "30156-03.htm"
        st:set_cond(2, true)
        q:take_items(player, ALBERIUS_LETTER, 1)
        q:give_items(player, EVERGREEN_AMULET, 1)
      elseif cond == 2 then
        htmltext = "30156-04.htm"
      elseif cond == 5 then
        htmltext = "30156-07.htm"
      elseif cond == 3 then
        htmltext = "30156-05.htm"
        st:set_cond(4, true)
        q:take_items(player, DRYAD_TEARS, -1)
        q:take_items(player, EVERGREEN_AMULET, 1)
        q:give_items(player, COBENDELL_MEDICINE_1, 1)
        q:give_items(player, COBENDELL_MEDICINE_2, 1)
        q:give_items(player, COBENDELL_MEDICINE_3, 1)
        q:give_items(player, COBENDELL_MEDICINE_4, 1)
        q:give_items(player, COBENDELL_MEDICINE_5, 1)
      elseif cond == 4 then
        htmltext = "30156-06.htm"
      end
    elseif sw2 == BERROS then
      if cond == 5 then
        htmltext = "30217-01.htm"
        checkItem(st, COBENDELL_MEDICINE_2)
      end
    elseif sw2 == VELTRESS then
      if cond == 5 then
        htmltext = "30219-01.htm"
        checkItem(st, COBENDELL_MEDICINE_3)
      end
    elseif sw2 == RAYEN then
      if cond == 5 then
        htmltext = "30221-01.htm"
        checkItem(st, COBENDELL_MEDICINE_4)
      end
    elseif sw2 == GARTRANDELL then
      if cond == 5 then
        htmltext = "30285-01.htm"
        checkItem(st, COBENDELL_MEDICINE_5)
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(2) and rnd(10) < 3 then
    q:give_items(killer, DRYAD_TEARS, 1)
    if q:get_quest_items_count(killer, DRYAD_TEARS) < 10 then
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      qs:set_cond(3, true)
    end
  end
end)
