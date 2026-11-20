-- Translated by questport from quests/Q00660_AidingTheFloranVillage (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(660, "Q00660_AidingTheFloranVillage", "Aiding the Floran Village")

-- NPCs
local MARIA = 30608
local ALEX = 30291
-- Monsters
local PLAIN_WATCHMAN = 21102
local ROCK_GOLEM = 21103
local LIZARDMEN_SUPPLIER = 21104
local LIZARDMEN_AGENT = 21105
local CURSED_SEER = 21106
local LIZARDMEN_COMMANDER = 21107
local LIZARDMEN_SHAMAN = 20781
-- Items
local WATCHING_EYES = 8074
local GOLEM_SHARD = 8075
local LIZARDMEN_SCALE = 8076
-- Rewards
local ADENA = 57
local ENCHANT_WEAPON_D = 955
local ENCHANT_ARMOR_D = 956

-- This method drops items following current counts.
-- @param player The Player to affect.
-- @param numberToVerify The count of qItems to drop from the different categories.
-- @return false when counter isn't reached, true otherwise.
local function verifyAndRemoveItems(player, numberToVerify)
  local eyes = q:get_quest_items_count(player, WATCHING_EYES)
  local scale = q:get_quest_items_count(player, LIZARDMEN_SCALE)
  local shard = q:get_quest_items_count(player, GOLEM_SHARD)
  if eyes + scale + shard < numberToVerify then
    return false
  end
  if eyes >= numberToVerify then
    q:take_items(player, WATCHING_EYES, numberToVerify)
  else
    local currentNumber = numberToVerify - eyes
    q:take_items(player, WATCHING_EYES, -1)
    if scale >= currentNumber then
      q:take_items(player, LIZARDMEN_SCALE, currentNumber)
    else
      currentNumber = currentNumber - scale
      q:take_items(player, LIZARDMEN_SCALE, -1)
      q:take_items(player, GOLEM_SHARD, currentNumber)
    end
  end
  return true
end

q:register_quest_items(WATCHING_EYES, LIZARDMEN_SCALE, GOLEM_SHARD)
q:add_start_npc(MARIA, ALEX)
q:add_talk_id(MARIA, ALEX)
q:add_kill_id(CURSED_SEER, PLAIN_WATCHMAN, ROCK_GOLEM, LIZARDMEN_SHAMAN, LIZARDMEN_SUPPLIER, LIZARDMEN_COMMANDER, LIZARDMEN_AGENT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30608-04.htm" then
    st:start_quest()
  elseif event == "30291-02.htm" then
    if player:level() < 30 then
      htmltext = "30291-02a.htm"
    else
      st:start_quest()
      st:set_cond(2)
    end
  elseif event == "30291-05.htm" then
    local count = q:get_quest_items_count(player, WATCHING_EYES) + q:get_quest_items_count(player, LIZARDMEN_SCALE) + q:get_quest_items_count(player, GOLEM_SHARD)
    if count == 0 then
      htmltext = "30291-05a.htm"
    else
      q:take_items(player, GOLEM_SHARD, -1)
      q:take_items(player, LIZARDMEN_SCALE, -1)
      q:take_items(player, WATCHING_EYES, -1)
      q:reward_items(player, ADENA, count * 100 + (count >= 45 and 9000 or 0))
    end
  elseif event == "30291-06.htm" then
    st:exit_quest(true, true)
  elseif event == "30291-11.htm" then
    if not verifyAndRemoveItems(player, 100) then
      htmltext = "30291-11a.htm"
    else
      if rnd(10) < 8 then
        q:reward_items(player, ADENA, 1000)
      else
        q:reward_items(player, ADENA, 13000)
        q:reward_items(player, ENCHANT_ARMOR_D, 1)
      end
    end
  elseif event == "30291-12.htm" then
    if not verifyAndRemoveItems(player, 200) then
      htmltext = "30291-12a.htm"
    else
      local luck = rnd(15)
      if luck < 8 then
        q:reward_items(player, ADENA, 2000)
      elseif luck < 12 then
        q:reward_items(player, ADENA, 20000)
        q:reward_items(player, ENCHANT_ARMOR_D, 1)
      else
        q:reward_items(player, ENCHANT_WEAPON_D, 1)
      end
    end
  elseif event == "30291-13.htm" then
    if not verifyAndRemoveItems(player, 500) then
      htmltext = "30291-13a.htm"
    else
      if rnd(10) < 8 then
        q:reward_items(player, ADENA, 5000)
      else
        q:reward_items(player, ADENA, 45000)
        q:reward_items(player, ENCHANT_WEAPON_D, 1)
      end
    end
  elseif event == "30291-17.htm" then
    local count = q:get_quest_items_count(player, WATCHING_EYES) + q:get_quest_items_count(player, LIZARDMEN_SCALE) + q:get_quest_items_count(player, GOLEM_SHARD)
    if count ~= 0 then
      htmltext = "30291-17a.htm"
      q:take_items(player, WATCHING_EYES, -1)
      q:take_items(player, LIZARDMEN_SCALE, -1)
      q:take_items(player, GOLEM_SHARD, -1)
      q:reward_items(player, ADENA, count * 100 + (count >= 45 and 9000 or 0))
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
    local sw2 = npc:id()
    if sw2 == MARIA then
      htmltext = player:level() < 30 and "30608-01.htm" or "30608-02.htm"
    elseif sw2 == ALEX then
      htmltext = "30291-01.htm"
    end
  elseif sw == State.STARTED then
    local sw3 = npc:id()
    if sw3 == MARIA then
      htmltext = "30608-06.htm"
    elseif sw3 == ALEX then
      local cond = st:get_cond()
      if cond == 1 then
        htmltext = "30291-03.htm"
        st:set_cond(2, true)
      elseif cond == 2 then
        htmltext = q:has_at_least_one_quest_item(player, WATCHING_EYES, LIZARDMEN_SCALE, GOLEM_SHARD) and "30291-04.htm" or "30291-05a.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 2, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  local sw = npc:id()
  if sw == PLAIN_WATCHMAN or sw == CURSED_SEER then
    if rnd(100) < 79 then
      q:give_items(partyMember, WATCHING_EYES, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == ROCK_GOLEM then
    if rnd(100) < 75 then
      q:give_items(partyMember, GOLEM_SHARD, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == LIZARDMEN_SHAMAN or sw == LIZARDMEN_SUPPLIER or sw == LIZARDMEN_AGENT or sw == LIZARDMEN_COMMANDER then
    if rnd(100) < 67 then
      q:give_items(partyMember, LIZARDMEN_SCALE, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
