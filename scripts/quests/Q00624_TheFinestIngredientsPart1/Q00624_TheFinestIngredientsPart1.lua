-- Translated by questport from quests/Q00624_TheFinestIngredientsPart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(624, "Q00624_TheFinestIngredientsPart1", "The Finest Ingredients - Part 1")

-- Monsters
local NEPENTHES = 21319
local ATROX = 21321
local ATROXSPAWN = 21317
local BANDERSNATCH = 21314
-- Items
local TRUNK_OF_NEPENTHES = 7202
local FOOT_OF_BANDERSNATCHLING = 7203
local SECRET_SPICE = 7204
-- Rewards
local ICE_CRYSTAL = 7080
local SOY_SAUCE_JAR = 7205

q:register_quest_items(TRUNK_OF_NEPENTHES, FOOT_OF_BANDERSNATCHLING, SECRET_SPICE)
q:add_start_npc(31521) -- Jeremy
q:add_talk_id(31521)
q:add_kill_id(NEPENTHES, ATROX, ATROXSPAWN, BANDERSNATCH)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31521-02.htm" then
    st:start_quest()
  elseif event == "31521-05.htm" then
    if q:get_quest_items_count(player, TRUNK_OF_NEPENTHES) >= 50 and q:get_quest_items_count(player, FOOT_OF_BANDERSNATCHLING) >= 50 and q:get_quest_items_count(player, SECRET_SPICE) >= 50 then
      q:take_items(player, TRUNK_OF_NEPENTHES, -1)
      q:take_items(player, FOOT_OF_BANDERSNATCHLING, -1)
      q:take_items(player, SECRET_SPICE, -1)
      q:give_items(player, ICE_CRYSTAL, 1)
      q:give_items(player, SOY_SAUCE_JAR, 1)
      st:exit_quest(true, true)
    else
      st:set_cond(1)
      htmltext = "31521-07.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 73 and "31521-03.htm" or "31521-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31521-06.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, TRUNK_OF_NEPENTHES) >= 50 and q:get_quest_items_count(player, FOOT_OF_BANDERSNATCHLING) >= 50 and q:get_quest_items_count(player, SECRET_SPICE) >= 50 then
        htmltext = "31521-04.htm"
      else
        htmltext = "31521-07.htm"
      end
    end
  end
  return htmltext
end)

-- Mobius checks the other two items on the killer, not the party member
-- who got the drop; kept as is.
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
  if sw == NEPENTHES then
    if q:get_quest_items_count(partyMember, TRUNK_OF_NEPENTHES) < 50 then
      q:give_items(partyMember, TRUNK_OF_NEPENTHES, 1)
      if q:get_quest_items_count(partyMember, TRUNK_OF_NEPENTHES) >= 50 and q:get_quest_items_count(player, FOOT_OF_BANDERSNATCHLING) >= 50 and q:get_quest_items_count(player, SECRET_SPICE) >= 50 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ATROX or sw == ATROXSPAWN then
    if q:get_quest_items_count(partyMember, SECRET_SPICE) < 50 then
      q:give_items(partyMember, SECRET_SPICE, 1)
      if q:get_quest_items_count(partyMember, SECRET_SPICE) >= 50 and q:get_quest_items_count(player, TRUNK_OF_NEPENTHES) >= 50 and q:get_quest_items_count(player, FOOT_OF_BANDERSNATCHLING) >= 50 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == BANDERSNATCH then
    if q:get_quest_items_count(partyMember, FOOT_OF_BANDERSNATCHLING) < 50 then
      q:give_items(partyMember, FOOT_OF_BANDERSNATCHLING, 1)
      if q:get_quest_items_count(partyMember, FOOT_OF_BANDERSNATCHLING) >= 50 and q:get_quest_items_count(player, TRUNK_OF_NEPENTHES) >= 50 and q:get_quest_items_count(player, SECRET_SPICE) >= 50 then
        st:set_cond(2, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
