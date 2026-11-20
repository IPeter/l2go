-- Translated by questport from quests/Q00257_TheGuardIsBusy (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(257, "Q00257_TheGuardIsBusy", "The Guard is Busy")

-- Items
local GLUDIO_LORD_MARK = 1084
local ORC_AMULET = 752
local ORC_NECKLACE = 1085
local WEREWOLF_FANG = 1086
-- Newbie Items
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789

q:register_quest_items(ORC_AMULET, ORC_NECKLACE, WEREWOLF_FANG, GLUDIO_LORD_MARK)
q:add_start_npc(30039) -- Gilbert
q:add_talk_id(30039)
q:add_kill_id(20006, 20093, 20096, 20098, 20130, 20131, 20132, 20342, 20343)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30039-03.htm" then
    st:start_quest()
    q:give_items(player, GLUDIO_LORD_MARK, 1)
  elseif event == "30039-05.htm" then
    q:take_items(player, GLUDIO_LORD_MARK, 1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 6 and "30039-01.htm" or "30039-02.htm"
  elseif sw == State.STARTED then
    local amulets = q:get_quest_items_count(player, ORC_AMULET)
    local necklaces = q:get_quest_items_count(player, ORC_NECKLACE)
    local fangs = q:get_quest_items_count(player, WEREWOLF_FANG)
    if amulets + necklaces + fangs == 0 then
      htmltext = "30039-04.htm"
    else
      htmltext = "30039-07.htm"
      local reward = 10 * amulets + 20 * fangs + 20 * necklaces
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and amulets + necklaces + fangs >= 10 then
        reward = reward + 1000
      end
      q:take_items(player, ORC_AMULET, -1)
      q:take_items(player, ORC_NECKLACE, -1)
      q:take_items(player, WEREWOLF_FANG, -1)
      q:give_adena(player, reward, true)
      -- Give newbie reward if player is eligible.
      if player:is_newbie() and st:get_int("Reward") == 0 then
        local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
        if newPlayerRewardsReceived < 1 then
          st:show_question_mark(26)
          st:set("Reward", "1")
          if player:is_mage_class() and player:race() ~= Race.ORC then
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
  local sw = npc:id()
  if sw == 20006 or sw == 20130 or sw == 20131 then
    if rnd(10) < 5 then
      q:give_items(player, ORC_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20093 or sw == 20096 or sw == 20098 then
    if rnd(10) < 5 then
      q:give_items(player, ORC_NECKLACE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20342 then
    if rnd(10) < 2 then
      q:give_items(player, WEREWOLF_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20343 then
    if rnd(10) < 4 then
      q:give_items(player, WEREWOLF_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20132 then
    if rnd(10) < 5 then
      q:give_items(player, WEREWOLF_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
