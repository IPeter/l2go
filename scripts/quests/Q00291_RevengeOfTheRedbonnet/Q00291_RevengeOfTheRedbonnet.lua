-- Translated by questport from quests/Q00291_RevengeOfTheRedbonnet (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(291, "Q00291_RevengeOfTheRedbonnet", "Revenge of the Redbonnet")

-- Quest items
local BLACK_WOLF_PELT = 1482
-- Rewards
local SCROLL_OF_ESCAPE = 736
local GRANDMA_PEARL = 1502
local GRANDMA_MIRROR = 1503
local GRANDMA_NECKLACE = 1504
local GRANDMA_HAIRPIN = 1505

q:register_quest_items(BLACK_WOLF_PELT)
q:add_start_npc(30553) -- Maryse Redbonnet
q:add_talk_id(30553)
q:add_kill_id(20317)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30553-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 4 and "30553-01.htm" or "30553-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30553-04.htm"
    elseif cond == 2 then
      htmltext = "30553-05.htm"
      q:take_items(player, BLACK_WOLF_PELT, -1)
      local random = rnd(100)
      if random < 3 then
        q:reward_items(player, GRANDMA_PEARL, 1)
      elseif random < 21 then
        q:reward_items(player, GRANDMA_MIRROR, 1)
      elseif random < 46 then
        q:reward_items(player, GRANDMA_NECKLACE, 1)
      else
        q:reward_items(player, SCROLL_OF_ESCAPE, 1)
        q:reward_items(player, GRANDMA_HAIRPIN, 1)
      end
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  q:give_items(player, BLACK_WOLF_PELT, 1)
  if q:get_quest_items_count(player, BLACK_WOLF_PELT) < 40 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
end)
