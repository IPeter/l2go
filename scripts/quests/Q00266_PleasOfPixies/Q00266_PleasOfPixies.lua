-- Translated by questport from quests/Q00266_PleasOfPixies (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(266, "Q00266_PleasOfPixies", "Pleas of Pixies")

-- Items
local PREDATOR_FANG = 1334
-- Rewards
local GLASS_SHARD = 1336
local EMERALD = 1337
local BLUE_ONYX = 1338
local ONYX = 1339

q:register_quest_items(PREDATOR_FANG)
q:add_start_npc(31852) -- Murika
q:add_talk_id(31852)
q:add_kill_id(20525, 20530, 20534, 20537)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31852-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "31852-00.htm"
    elseif player:level() < 3 then
      htmltext = "31852-01.htm"
    else
      htmltext = "31852-02.htm"
    end
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, PREDATOR_FANG) < 100 then
      htmltext = "31852-04.htm"
    else
      htmltext = "31852-05.htm"
      q:take_items(player, PREDATOR_FANG, -1)
      local n = rnd(100)
      if n < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_JACKPOT)
        q:reward_items(player, EMERALD, 1)
      elseif n < 30 then
        q:reward_items(player, BLUE_ONYX, 1)
      elseif n < 60 then
        q:reward_items(player, ONYX, 1)
      else
        q:reward_items(player, GLASS_SHARD, 1)
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
  local sw = npc:id()
  if sw == 20525 then
    q:give_items(player, PREDATOR_FANG, rnd(2, 3))
    if q:get_quest_items_count(player, PREDATOR_FANG) < 100 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  elseif sw == 20530 then
    if rnd(10) < 8 then
      q:give_items(player, PREDATOR_FANG, 1)
      if q:get_quest_items_count(player, PREDATOR_FANG) < 100 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(2, true)
      end
    end
  elseif sw == 20534 then
    if rnd(10) < 6 then
      q:give_items(player, PREDATOR_FANG, rnd(3) == 0 and 1 or 2)
      if q:get_quest_items_count(player, PREDATOR_FANG) < 100 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(2, true)
      end
    end
  elseif sw == 20537 then
    q:give_items(player, PREDATOR_FANG, 2)
    if q:get_quest_items_count(player, PREDATOR_FANG) < 100 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
