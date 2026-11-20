-- Translated by questport from quests/Q00265_BondsOfSlavery (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(265, "Q00265_BondsOfSlavery", "Chains of Slavery")

-- Item
local SHACKLE = 1368
-- Newbie Items
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789

q:register_quest_items(SHACKLE)
q:add_start_npc(30357) -- Kristin
q:add_talk_id(30357)
q:add_kill_id(20004, 20005)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30357-03.htm" then
    st:start_quest()
  elseif event == "30357-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30357-00.htm"
    elseif player:level() < 6 then
      htmltext = "30357-01.htm"
    else
      htmltext = "30357-02.htm"
    end
  elseif sw == State.STARTED then
    local shackles = q:get_quest_items_count(player, SHACKLE)
    if shackles == 0 then
      htmltext = "30357-04.htm"
    else
      htmltext = "30357-05.htm"
      local reward = 12 * shackles
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and shackles >= 10 then
        reward = reward + 500
      end
      q:take_items(player, SHACKLE, -1)
      q:give_adena(player, reward, true)
      -- Give newbie reward if player is eligible.
      if player:is_newbie() and st:get_int("Reward") == 0 then
        local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
        if newPlayerRewardsReceived < 1 then
          st:show_question_mark(26)
          st:set("Reward", "1")
          if player:is_mage_class() then
            st:play_tutorial_voice("tutorial_voice_027")
            q:give_items(player, SPIRITSHOT_FOR_BEGINNERS, 3000)
            newPlayerRewardsReceived = newPlayerRewardsReceived + 1
            player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
          else
            st:play_tutorial_voice("tutorial_voice_026")
            q:give_items(player, SOULSHOT_FOR_BEGINNERS, 6000)
            newPlayerRewardsReceived = newPlayerRewardsReceived + 1
            player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
          end
        end
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
  if rnd(10) < (npc:id() == 20004 and 5 or 6) then
    q:give_items(player, SHACKLE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
