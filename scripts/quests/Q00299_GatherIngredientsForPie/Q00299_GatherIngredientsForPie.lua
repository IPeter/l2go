-- Translated by questport from quests/Q00299_GatherIngredientsForPie (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(299, "Q00299_GatherIngredientsForPie", "Gather Ingredients for Pie")

-- NPCs
local LARA = 30063
local BRIGHT = 30466
local EMILY = 30620
-- Items
local FRUIT_BASKET = 7136
local AVELLAN_SPICE = 7137
local HONEY_POUCH = 7138
-- Reward resources
local VARNISH = 1865

q:register_quest_items(FRUIT_BASKET, AVELLAN_SPICE, HONEY_POUCH)
q:add_start_npc(EMILY)
q:add_talk_id(EMILY, LARA, BRIGHT)
q:add_kill_id(20934, 20935) -- Wasp Worker, Wasp Leader

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30620-1.htm" then
    st:start_quest()
  elseif event == "30620-3.htm" then
    st:set_cond(3, true)
    q:take_items(player, HONEY_POUCH, -1)
  elseif event == "30063-1.htm" then
    st:set_cond(4, true)
    q:give_items(player, AVELLAN_SPICE, 1)
  elseif event == "30620-5.htm" then
    st:set_cond(5, true)
    q:take_items(player, AVELLAN_SPICE, 1)
  elseif event == "30466-1.htm" then
    st:set_cond(6, true)
    q:give_items(player, FRUIT_BASKET, 1)
  elseif event == "30620-7a.htm" then
    if q:has_quest_items(player, FRUIT_BASKET) then
      htmltext = "30620-7.htm"
      q:take_items(player, FRUIT_BASKET, 1)
      if rnd(100) < 70 then
        q:give_adena(player, 25000, true)
      else
        q:give_items(player, VARNISH, 50)
      end
      st:exit_quest(true, true)
    else
      st:set_cond(5)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 34 and "30620-0a.htm" or "30620-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == EMILY then
      if cond == 1 then
        htmltext = "30620-1a.htm"
      elseif cond == 2 then
        if q:get_quest_items_count(player, HONEY_POUCH) >= 100 then
          htmltext = "30620-2.htm"
        else
          htmltext = "30620-2a.htm"
          st:exit_quest(true)
        end
      elseif cond == 3 then
        htmltext = "30620-3a.htm"
      elseif cond == 4 then
        if q:has_quest_items(player, AVELLAN_SPICE) then
          htmltext = "30620-4.htm"
        else
          htmltext = "30620-4a.htm"
          st:exit_quest(true)
        end
      elseif cond == 5 then
        htmltext = "30620-5a.htm"
      elseif cond == 6 then
        htmltext = "30620-6.htm"
      end
    elseif sw2 == LARA then
      if cond == 3 then
        htmltext = "30063-0.htm"
      elseif cond > 3 then
        htmltext = "30063-1a.htm"
      end
    elseif sw2 == BRIGHT then
      if cond == 5 then
        htmltext = "30466-0.htm"
      elseif cond > 5 then
        htmltext = "30466-1a.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, 1, 3, npc)
  if st == nil then
    return
  end
  if rnd(1000) < (npc:id() == 20934 and 571 or 625) then
    local partyMember = st:get_player()
    q:give_items(partyMember, HONEY_POUCH, 1)
    if q:get_quest_items_count(partyMember, HONEY_POUCH) < 100 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
