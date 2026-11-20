-- Translated by questport from quests/Q00024_InhabitantsOfTheForestOfTheDead (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(24, "Q00024_InhabitantsOfTheForestOfTheDead", "Inhabitants of the Forest of the Dead")
-- L2Go: the translator asked for q:requires("pets") for the summon branch
-- of on_aggro_range_enter only. Summons do not exist yet, so that branch
-- never runs and the quest is offered (see on_aggro_range_enter).

-- NPCs
local DORIAN = 31389
local WIZARD = 31522
local TOMBSTONE = 31531
local MAID_OF_LIDIA = 31532
local MOBS = {21557, 21558, 21560, 21563, 21564, 21565, 21566, 21567}
-- Items
local LETTER = 7065
local HAIRPIN = 7148
local TOTEM = 7151
local FLOWER = 7152
local SILVER_CROSS = 7153
local BROKEN_SILVER_CROSS = 7154
local SUSPICIOUS_TOTEM = 7156

q:add_start_npc(DORIAN)
q:add_talk_id(DORIAN, TOMBSTONE, MAID_OF_LIDIA, WIZARD)
q:register_quest_items(FLOWER, SILVER_CROSS, BROKEN_SILVER_CROSS, LETTER, HAIRPIN, TOTEM)
q:add_kill_id(MOBS)
q:add_aggro_range_enter_id(25332)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return event
  end
  local htmltext = event
  if event == "31389-03.htm" then
    st:start_quest()
    st:set("state", "1")
    q:give_items(player, FLOWER, 1)
  elseif event == "31389-08.htm" then
    st:set("state", "3")
  elseif event == "31389-13.htm" then
    st:set_cond(3, true)
    st:set("state", "4")
    q:give_items(player, SILVER_CROSS, 1)
  elseif event == "31389-18.htm" then
    q:play_sound(player, QuestSound.INTERFACESOUND_CHARSTAT_OPEN)
  elseif event == "31389-19.htm" then
    st:set_cond(5, true)
    st:set("state", "5")
    q:take_items(player, BROKEN_SILVER_CROSS, -1)
  elseif event == "31522-03.htm" then
    st:set("state", "12")
    q:take_items(player, TOTEM, -1)
  elseif event == "31522-08.htm" then
    st:set_cond(11, true)
    st:set("state", "13")
  elseif event == "31522-17.htm" then
    st:set("state", "14")
  elseif event == "31522-21.htm" then
    q:give_items(player, SUSPICIOUS_TOTEM, 1)
    st:exit_quest(false, true)
  elseif event == "31532-04.htm" then
    st:set_cond(6, true)
    st:set("state", "6")
    q:give_items(player, LETTER, 1)
  elseif event == "31532-06.htm" then
    if q:has_quest_items(player, HAIRPIN) then
      st:set("state", "8")
      q:take_items(player, LETTER, -1)
      q:take_items(player, HAIRPIN, -1)
    else
      st:set_cond(7)
      st:set("state", "7")
      htmltext = "31532-07.htm"
    end
  elseif event == "31532-10.htm" then
    st:set("state", "9")
  elseif event == "31532-14.htm" then
    st:set("state", "10")
  elseif event == "31532-19.htm" then
    st:set_cond(9, true)
    st:set("state", "11")
  elseif event == "31531-02.htm" then
    st:set_cond(2, true)
    st:set("state", "2")
    q:take_items(player, FLOWER, -1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local st2 = player:get_quest_state("Q00023_LidiasHeart")
    if st2 ~= nil and st2:is_completed() and player:level() >= 65 then
      htmltext = "31389-01.htm"
    else
      htmltext = "31389-02.htm"
    end
  elseif sw == State.STARTED then
    local state = st:get_int("state")
    local sw2 = npc:id()
    if sw2 == 31389 then
      if state == 1 then
        htmltext = "31389-04.htm"
      elseif state == 2 then
        htmltext = "31389-05.htm"
      elseif state == 3 then
        htmltext = "31389-09.htm"
      elseif state == 4 then
        if q:has_quest_items(player, SILVER_CROSS) then
          htmltext = "31389-14.htm"
        elseif q:has_quest_items(player, BROKEN_SILVER_CROSS) then
          htmltext = "31389-15.htm"
        end
      elseif state == 5 then
        htmltext = "31389-20.htm"
      elseif state == 7 and not q:has_quest_items(player, HAIRPIN) then
        htmltext = "31389-21.htm"
        st:set_cond(8, true)
        q:give_items(player, HAIRPIN, 1)
      elseif state == 7 and q:has_quest_items(player, HAIRPIN) or state == 6 then
        htmltext = "31389-22.htm"
      end
    elseif sw2 == 31522 then
      if state == 11 and q:has_quest_items(player, TOTEM) then
        htmltext = "31522-01.htm"
      elseif state == 12 then
        htmltext = "31522-04.htm"
      elseif state == 13 then
        htmltext = "31522-09.htm"
      elseif state == 14 then
        htmltext = "31522-18.htm"
      end
    elseif sw2 == 31531 then
      if state == 1 and q:has_quest_items(player, FLOWER) then
        htmltext = "31531-01.htm"
        q:play_sound(player, QuestSound.AMDSOUND_WIND_LOOT)
      elseif state == 2 then
        htmltext = "31531-03.htm"
      end
    elseif sw2 == 31532 then
      if state == 5 then
        htmltext = "31532-01.htm"
      elseif state == 6 and q:has_quest_items(player, LETTER) then
        htmltext = "31532-05.htm"
      elseif state == 7 then
        htmltext = "31532-07a.htm"
      elseif state == 8 then
        htmltext = "31532-08.htm"
      elseif state == 9 then
        htmltext = "31532-11.htm"
      elseif state == 10 then
        htmltext = "31532-15.htm"
      elseif state == 11 then
        htmltext = "31532-20.htm"
      end
    end
    -- L2Go: every Java case of the STARTED branch returns; the fall-through
    -- into the COMPLETED case the translator copied could never run.
  elseif sw == State.COMPLETED then
    if npc:id() == 31522 then
      htmltext = "31522-22.htm"
    else
      htmltext = q:get_already_completed_msg(player)
    end
  end
  return htmltext
end)

q:on_aggro_range_enter(function(npc, player, isPet)
  if isPet then
    -- L2Go: Mobius takes the summon (player.getSummon()) off the attack-by
    -- list. Summons do not exist yet, so this never runs.
    return
  end
  npc:remove_attacker(player)
  local st = player:get_quest_state(q:name())
  if st ~= nil and q:get_quest_items_count(player, SILVER_CROSS) > 0 then
    q:take_items(player, SILVER_CROSS, -1)
    q:give_items(player, BROKEN_SILVER_CROSS, 1)
    st:set_cond(4)
    npc:broadcast_say(ChatType.GENERAL, "That sign!")
  end
end)

q:on_kill(function(npc, player, isSummon)
  local st = q:get_quest_state(player, false)
  if st ~= nil and st:is_cond(9) and rnd(100) < 10 then
    q:give_items(player, TOTEM, 1)
    st:set_cond(10, true)
  end
end)
