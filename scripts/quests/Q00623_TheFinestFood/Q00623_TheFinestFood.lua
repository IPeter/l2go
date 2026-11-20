-- Translated by questport from quests/Q00623_TheFinestFood (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(623, "Q00623_TheFinestFood", "The Finest Food")

-- NPC
local JEREMY = 31521
-- Monsters
local FLAVA = 21316
local BUFFALO = 21315
local ANTELOPE = 21318
-- Items
local LEAF_OF_FLAVA = 7199
local BUFFALO_MEAT = 7200
local ANTELOPE_HORN = 7201

q:register_quest_items(LEAF_OF_FLAVA, BUFFALO_MEAT, ANTELOPE_HORN)
q:add_start_npc(JEREMY)
q:add_talk_id(JEREMY)
q:add_kill_id(FLAVA, BUFFALO, ANTELOPE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31521-02.htm" then
    if player:level() >= 71 then
      st:start_quest()
    else
      htmltext = "31521-03.htm"
    end
  elseif event == "31521-05.htm" then
    q:take_items(player, LEAF_OF_FLAVA, -1)
    q:take_items(player, BUFFALO_MEAT, -1)
    q:take_items(player, ANTELOPE_HORN, -1)
    local luck = rnd(100)
    if luck < 11 then
      q:give_adena(player, 25000, true)
      q:give_items(player, 6849, 1)
    elseif luck < 23 then
      q:give_adena(player, 65000, true)
      q:give_items(player, 6847, 1)
    elseif luck < 33 then
      q:give_adena(player, 25000, true)
      q:give_items(player, 6851, 1)
    else
      q:give_adena(player, 73000, true)
      q:add_exp_and_sp(player, 230000, 18250)
    end
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "31521-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31521-06.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, LEAF_OF_FLAVA) >= 100 and q:get_quest_items_count(player, BUFFALO_MEAT) >= 100 and q:get_quest_items_count(player, ANTELOPE_HORN) >= 100 then
        htmltext = "31521-04.htm"
      else
        htmltext = "31521-07.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  local sw = npc:id()
  if sw == FLAVA then
    if q:get_quest_items_count(partyMember, LEAF_OF_FLAVA) < 100 then
      q:give_items(partyMember, LEAF_OF_FLAVA, 1)
      if q:get_quest_items_count(partyMember, LEAF_OF_FLAVA) >= 100 and q:get_quest_items_count(partyMember, BUFFALO_MEAT) >= 100 and q:get_quest_items_count(partyMember, ANTELOPE_HORN) >= 100 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == BUFFALO then
    if q:get_quest_items_count(partyMember, BUFFALO_MEAT) < 100 then
      q:give_items(partyMember, BUFFALO_MEAT, 1)
      if q:get_quest_items_count(partyMember, BUFFALO_MEAT) >= 100 and q:get_quest_items_count(partyMember, LEAF_OF_FLAVA) >= 100 and q:get_quest_items_count(partyMember, ANTELOPE_HORN) >= 100 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ANTELOPE then
    if q:get_quest_items_count(partyMember, ANTELOPE_HORN) < 100 then
      q:give_items(partyMember, ANTELOPE_HORN, 1)
      if q:get_quest_items_count(partyMember, ANTELOPE_HORN) >= 100 and q:get_quest_items_count(partyMember, LEAF_OF_FLAVA) >= 100 and q:get_quest_items_count(partyMember, BUFFALO_MEAT) >= 100 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
