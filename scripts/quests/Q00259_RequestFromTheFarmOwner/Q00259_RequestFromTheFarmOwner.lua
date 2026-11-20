-- Translated by questport from quests/Q00259_RequestFromTheFarmOwner (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(259, "Q00259_RequestFromTheFarmOwner", "Rancher's Plea")

-- NPCs
local EDMOND = 30497
local MARIUS = 30405
-- Monsters
local GIANT_SPIDER = 20103
local TALON_SPIDER = 20106
local BLADE_SPIDER = 20108
-- Items
local GIANT_SPIDER_SKIN = 1495
-- Rewards
local ADENA = 57
local HEALING_POTION = 1061
local WOODEN_ARROW = 17

q:register_quest_items(GIANT_SPIDER_SKIN)
q:add_start_npc(EDMOND)
q:add_talk_id(EDMOND, MARIUS)
q:add_kill_id(GIANT_SPIDER, TALON_SPIDER, BLADE_SPIDER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30497-03.htm" then
    st:start_quest()
  elseif event == "30497-06.htm" then
    st:exit_quest(true, true)
  elseif event == "30405-04.htm" then
    if q:get_quest_items_count(player, GIANT_SPIDER_SKIN) >= 10 then
      q:take_items(player, GIANT_SPIDER_SKIN, 10)
      q:reward_items(player, HEALING_POTION, 1)
    else
      htmltext = "<html><body>Incorrect item count</body></html>"
    end
  elseif event == "30405-05.htm" then
    if q:get_quest_items_count(player, GIANT_SPIDER_SKIN) >= 10 then
      q:take_items(player, GIANT_SPIDER_SKIN, 10)
      q:reward_items(player, WOODEN_ARROW, 50)
    else
      htmltext = "<html><body>Incorrect item count</body></html>"
    end
  elseif event == "30405-07.htm" then
    if q:get_quest_items_count(player, GIANT_SPIDER_SKIN) >= 10 then
      htmltext = "30405-06.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30497-01.htm" or "30497-02.htm"
  elseif sw == State.STARTED then
    local count = q:get_quest_items_count(player, GIANT_SPIDER_SKIN)
    local sw2 = npc:id()
    if sw2 == EDMOND then
      if count == 0 then
        htmltext = "30497-04.htm"
      else
        htmltext = "30497-05.htm"
        q:take_items(player, GIANT_SPIDER_SKIN, -1)
        local reward = count * 25
        if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and count >= 10 then
          reward = reward + 250
        end
        q:reward_items(player, ADENA, reward)
      end
    elseif sw2 == MARIUS then
      htmltext = count < 10 and "30405-01.htm" or "30405-02.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  q:give_items(player, GIANT_SPIDER_SKIN, 1)
  q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
end)
