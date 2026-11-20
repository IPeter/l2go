-- Translated by questport from quests/Q00296_TarantulasSpiderSilk (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(296, "Q00296_TarantulasSpiderSilk", "Tarantula's Spider Silk")

-- NPCs
local MION = 30519
local DEFENDER_NATHAN = 30548
-- Quest Items
local TARANTULA_SPIDER_SILK = 1493
local TARANTULA_SPINNERETTE = 1494
-- Items
local RING_OF_RACCOON = 1508
local RING_OF_FIREFLY = 1509

q:register_quest_items(TARANTULA_SPIDER_SILK, TARANTULA_SPINNERETTE)
q:add_start_npc(MION)
q:add_talk_id(MION, DEFENDER_NATHAN)
q:add_kill_id(20394, 20403, 20508) -- Crimson Tarantula, Hunter Tarantula, Plunder arantula

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30519-03.htm" then
    if q:has_at_least_one_quest_item(player, RING_OF_RACCOON, RING_OF_FIREFLY) then
      st:start_quest()
    else
      htmltext = "30519-03a.htm"
    end
  elseif event == "30519-06.htm" then
    q:take_items(player, TARANTULA_SPIDER_SILK, -1)
    q:take_items(player, TARANTULA_SPINNERETTE, -1)
    st:exit_quest(true, true)
  elseif event == "30548-02.htm" then
    local count = q:get_quest_items_count(player, TARANTULA_SPINNERETTE)
    if count > 0 then
      htmltext = "30548-03.htm"
      q:take_items(player, TARANTULA_SPINNERETTE, -1)
      q:give_items(player, TARANTULA_SPIDER_SILK, count * (15 + rnd(10)))
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30519-01.htm" or "30519-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == MION then
      local count = q:get_quest_items_count(player, TARANTULA_SPIDER_SILK)
      if count == 0 then
        htmltext = "30519-04.htm"
      else
        htmltext = "30519-05.htm"
        q:take_items(player, TARANTULA_SPIDER_SILK, -1)
        local reward = count * 20
        if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and count >= 10 then
          reward = reward + 2000
        end
        q:give_adena(player, reward, true)
      end
    elseif sw2 == DEFENDER_NATHAN then
      htmltext = "30548-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local rnd_ = rnd(100)
  if rnd_ > 95 then
    q:give_items(player, TARANTULA_SPINNERETTE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif rnd_ > 45 then
    q:give_items(player, TARANTULA_SPIDER_SILK, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
