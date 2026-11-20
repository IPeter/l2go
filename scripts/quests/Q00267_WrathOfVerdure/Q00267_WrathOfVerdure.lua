-- Translated by questport from quests/Q00267_WrathOfVerdure (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(267, "Q00267_WrathOfVerdure", "Wrath of Verdure")

-- Items
local GOBLIN_CLUB = 1335
-- Reward
local SILVERY_LEAF = 1340

q:register_quest_items(GOBLIN_CLUB)
q:add_start_npc(31853) -- Bremec
q:add_talk_id(31853)
q:add_kill_id(20325) -- Goblin

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31853-03.htm" then
    st:start_quest()
  elseif event == "31853-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "31853-00.htm"
    elseif player:level() < 4 then
      htmltext = "31853-01.htm"
    else
      htmltext = "31853-02.htm"
    end
  elseif sw == State.STARTED then
    local count = q:get_quest_items_count(player, GOBLIN_CLUB)
    if count > 0 then
      htmltext = "31853-05.htm"
      q:take_items(player, GOBLIN_CLUB, -1)
      q:reward_items(player, SILVERY_LEAF, count)
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and count >= 10 then
        q:give_adena(player, 600, true)
      end
    else
      htmltext = "31853-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd.bool() then
    q:give_items(player, GOBLIN_CLUB, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
