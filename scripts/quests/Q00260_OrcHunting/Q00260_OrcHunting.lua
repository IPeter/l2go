-- Translated by questport from quests/Q00260_OrcHunting (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(260, "Q00260_OrcHunting", "Hunt the Orcs")

-- NPC
local RAYEN = 30221
-- Monsters
local KABOO_ORC = 20468
local KABOO_ORC_ARCHER = 20469
local KABOO_ORC_GRUNT = 20470
local KABOO_ORC_FIGHTER = 20471
local KABOO_ORC_FIGHTER_LEADER = 20472
local KABOO_ORC_FIGHTER_LIEUTENANT = 20473
-- Items
local ORC_AMULET = 1114
local ORC_NECKLACE = 1115
-- Newbie Items
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789

q:register_quest_items(ORC_AMULET, ORC_NECKLACE)
q:add_start_npc(RAYEN)
q:add_talk_id(RAYEN)
q:add_kill_id(KABOO_ORC, KABOO_ORC_ARCHER, KABOO_ORC_GRUNT, KABOO_ORC_FIGHTER, KABOO_ORC_FIGHTER_LEADER, KABOO_ORC_FIGHTER_LIEUTENANT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30221-03.htm" then
    st:start_quest()
  elseif event == "30221-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30221-00.htm"
    elseif player:level() < 6 then
      htmltext = "30221-01.htm"
    else
      htmltext = "30221-02.htm"
    end
  elseif sw == State.STARTED then
    local amulet = q:get_quest_items_count(player, ORC_AMULET)
    local necklace = q:get_quest_items_count(player, ORC_NECKLACE)
    if amulet == 0 and necklace == 0 then
      htmltext = "30221-04.htm"
    else
      htmltext = "30221-05.htm"
      local reward = amulet * 12 + necklace * 30
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and amulet + necklace >= 10 then
        reward = reward + 1000
      end
      q:take_items(player, ORC_AMULET, -1)
      q:take_items(player, ORC_NECKLACE, -1)
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
  local sw = npc:id()
  if sw == KABOO_ORC or sw == KABOO_ORC_GRUNT or sw == KABOO_ORC_ARCHER then
    if rnd(10) < 5 then
      q:give_items(player, ORC_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == KABOO_ORC_FIGHTER or sw == KABOO_ORC_FIGHTER_LEADER or sw == KABOO_ORC_FIGHTER_LIEUTENANT then
    if rnd(10) < 5 then
      q:give_items(player, ORC_NECKLACE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
