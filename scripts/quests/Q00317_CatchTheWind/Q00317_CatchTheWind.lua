-- Translated by questport from quests/Q00317_CatchTheWind (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(317, "Q00317_CatchTheWind", "Catch the Wind")

-- Item
local WIND_SHARD = 1078

q:register_quest_items(WIND_SHARD)
q:add_start_npc(30361) -- Rizraell
q:add_talk_id(30361)
q:add_kill_id(20036, 20044)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30361-04.htm" then
    st:start_quest()
  elseif event == "30361-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 18 and "30361-02.htm" or "30361-03.htm"
  elseif sw == State.STARTED then
    local shards = q:get_quest_items_count(player, WIND_SHARD)
    if shards == 0 then
      htmltext = "30361-05.htm"
    else
      htmltext = "30361-07.htm"
      q:take_items(player, WIND_SHARD, -1)
      local reward = 40 * shards
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and shards >= 10 then
        reward = reward + 2988
      end
      q:give_adena(player, reward, true)
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
    q:give_items(player, WIND_SHARD, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
