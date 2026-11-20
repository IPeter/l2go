-- Translated by questport from quests/Q00649_ALooterAndARailroadMan (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(649, "Q00649_ALooterAndARailroadMan", "A Looter and a Railroad Man")

-- NPC
local OBI = 32052
-- Item
local THIEF_GUILD_MARK = 8099

q:register_quest_items(THIEF_GUILD_MARK)
q:add_start_npc(OBI)
q:add_talk_id(OBI)
q:add_kill_id(22017, 22018, 22019, 22021, 22022, 22023, 22024, 22026)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32052-1.htm" then
    st:start_quest()
  elseif event == "32052-3.htm" then
    if q:get_quest_items_count(player, THIEF_GUILD_MARK) < 200 then
      htmltext = "32052-3a.htm"
    else
      q:take_items(player, THIEF_GUILD_MARK, -1)
      q:give_adena(player, 21698, true)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 30 and "32052-0a.htm" or "32052-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32052-2a.htm"
    elseif cond == 2 then
      htmltext = "32052-2.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(10) < 8 then
    q:give_items(player, THIEF_GUILD_MARK, 1)
    if q:get_quest_items_count(player, THIEF_GUILD_MARK) < 200 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
