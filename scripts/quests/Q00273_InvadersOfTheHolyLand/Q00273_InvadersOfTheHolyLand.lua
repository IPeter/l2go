-- Translated by questport from quests/Q00273_InvadersOfTheHolyLand (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(273, "Q00273_InvadersOfTheHolyLand", "Invaders of the Holy Land")

-- Items
local BLACK_SOULSTONE = 1475
local RED_SOULSTONE = 1476
-- Reward
local SOULSHOT_FOR_BEGINNERS = 5789

q:register_quest_items(BLACK_SOULSTONE, RED_SOULSTONE)
q:add_start_npc(30566) -- Varkees
q:add_talk_id(30566)
q:add_kill_id(20311, 20312, 20313)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30566-03.htm" then
    st:start_quest()
  elseif event == "30566-07.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30566-00.htm"
    elseif player:level() < 6 then
      htmltext = "30566-01.htm"
    else
      htmltext = "30566-02.htm"
    end
  elseif sw == State.STARTED then
    local red = q:get_quest_items_count(player, RED_SOULSTONE)
    local black = q:get_quest_items_count(player, BLACK_SOULSTONE)
    if red + black == 0 then
      htmltext = "30566-04.htm"
    else
      if red == 0 then
        htmltext = "30566-05.htm"
      else
        htmltext = "30566-06.htm"
      end
      local reward = black * 3 + red * 10
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD then
        reward = reward + (black >= 10 and (red >= 1 and 1800 or 1500) or 0)
      end
      q:take_items(player, BLACK_SOULSTONE, -1)
      q:take_items(player, RED_SOULSTONE, -1)
      q:give_adena(player, reward, true)
      -- Give newbie reward if player is eligible.
      if player:is_newbie() and st:get_int("Reward") == 0 then
        local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
        if newPlayerRewardsReceived < 1 then
          q:give_items(player, SOULSHOT_FOR_BEGINNERS, 6000)
          st:play_tutorial_voice("tutorial_voice_026")
          st:set("Reward", "1")
          newPlayerRewardsReceived = newPlayerRewardsReceived + 1
          player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
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
  local npcId = npc:id()
  local probability = 77
  if npcId == 20311 then
    probability = 90
  elseif npcId == 20312 then
    probability = 87
  end
  if rnd(100) <= probability then
    q:give_items(player, BLACK_SOULSTONE, 1)
  else
    q:give_items(player, RED_SOULSTONE, 1)
  end
  q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
end)
