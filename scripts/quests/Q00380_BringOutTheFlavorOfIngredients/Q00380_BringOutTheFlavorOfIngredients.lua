-- Translated by questport from quests/Q00380_BringOutTheFlavorOfIngredients (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(380, "Q00380_BringOutTheFlavorOfIngredients", "Bring Out the Flavor of Ingredients!")

-- Monsters
local DIRE_WOLF = 20205
local KADIF_WEREWOLF = 20206
local GIANT_MIST_LEECH = 20225
-- Items
local RITRON_FRUIT = 5895
local MOON_FACE_FLOWER = 5896
local LEECH_FLUIDS = 5897
local ANTIDOTE = 1831
-- Rewards
local RITRON_JELLY = 5960
local JELLY_RECIPE = 5959

q:register_quest_items(RITRON_FRUIT, MOON_FACE_FLOWER, LEECH_FLUIDS)
q:add_start_npc(30069) -- Rollant
q:add_talk_id(30069)
q:add_kill_id(DIRE_WOLF, KADIF_WEREWOLF, GIANT_MIST_LEECH)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30069-04.htm" then
    st:start_quest()
  elseif event == "30069-12.htm" then
    q:give_items(player, JELLY_RECIPE, 1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 24 and "30069-00.htm" or "30069-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30069-06.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, ANTIDOTE) >= 2 then
        htmltext = "30069-07.htm"
        st:set_cond(3, true)
        q:take_items(player, RITRON_FRUIT, -1)
        q:take_items(player, MOON_FACE_FLOWER, -1)
        q:take_items(player, LEECH_FLUIDS, -1)
        q:take_items(player, ANTIDOTE, 2)
      else
        htmltext = "30069-06.htm"
      end
    elseif cond == 3 then
      htmltext = "30069-08.htm"
      st:set_cond(4, true)
    elseif cond == 4 then
      htmltext = "30069-09.htm"
      st:set_cond(5, true)
    elseif cond == 5 then
      htmltext = "30069-10.htm"
      st:set_cond(6, true)
    elseif cond == 6 then
      q:give_items(player, RITRON_JELLY, 1)
      if rnd(100) < 55 then
        htmltext = "30069-11.htm"
      else
        htmltext = "30069-13.htm"
        st:exit_quest(true, true)
      end
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
  if sw == DIRE_WOLF then
    if rnd(10) < 1 and q:get_quest_items_count(player, RITRON_FRUIT) < 4 then
      q:give_items(player, RITRON_FRUIT, 1)
      if q:get_quest_items_count(player, RITRON_FRUIT) < 4 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        if q:get_quest_items_count(player, MOON_FACE_FLOWER) == 20 and q:get_quest_items_count(player, LEECH_FLUIDS) == 10 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == KADIF_WEREWOLF then
    if rnd(10) < 5 and q:get_quest_items_count(player, MOON_FACE_FLOWER) < 20 then
      q:give_items(player, MOON_FACE_FLOWER, 1)
      if q:get_quest_items_count(player, MOON_FACE_FLOWER) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        if q:get_quest_items_count(player, RITRON_FRUIT) == 4 and q:get_quest_items_count(player, LEECH_FLUIDS) == 10 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == GIANT_MIST_LEECH then
    if rnd(10) < 5 and q:get_quest_items_count(player, LEECH_FLUIDS) < 10 then
      q:give_items(player, LEECH_FLUIDS, 1)
      if q:get_quest_items_count(player, LEECH_FLUIDS) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        if q:get_quest_items_count(player, RITRON_FRUIT) == 4 and q:get_quest_items_count(player, MOON_FACE_FLOWER) == 20 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
