-- Translated by questport from quests/Q00157_RecoverSmuggledGoods (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(157, "Q00157_RecoverSmuggledGoods", "Recover Smuggled Goods")

-- Item
local ADAMANTITE_ORE = 1024
-- Reward
local BUCKLER = 20

q:register_quest_items(ADAMANTITE_ORE)
q:add_start_npc(30005) -- Wilford
q:add_talk_id(30005)
q:add_kill_id(20121) -- Toad

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30005-05.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 5 and "30005-02.htm" or "30005-03.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30005-06.htm"
    elseif cond == 2 then
      htmltext = "30005-07.htm"
      q:take_items(player, ADAMANTITE_ORE, -1)
      q:give_items(player, BUCKLER, 1)
      st:exit_quest(false, true)
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) and rnd(10) < 4 and q:get_quest_items_count(killer, ADAMANTITE_ORE) < 20 then
    q:give_items(killer, ADAMANTITE_ORE, 1)
    if q:get_quest_items_count(killer, ADAMANTITE_ORE) >= 20 then
      qs:set_cond(2, true)
    else
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
