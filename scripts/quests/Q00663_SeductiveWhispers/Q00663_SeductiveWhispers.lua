-- Translated by questport from quests/Q00663_SeductiveWhispers (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(663, "Q00663_SeductiveWhispers", "Seductive Whispers")

-- NPC
local WILBERT = 30846
-- Quest item
local SPIRIT_BEAD = 8766
-- Rewards
local ADENA = 57
local ENCHANT_WEAPON_A = 729
local ENCHANT_ARMOR_A = 730
local ENCHANT_WEAPON_B = 947
local ENCHANT_ARMOR_B = 948
local ENCHANT_WEAPON_C = 951
local ENCHANT_WEAPON_D = 955
local RECIPES = {2353, 4963, 4967, 5000, 5001, 5002, 5004, 5005, 5006, 5007}
local BLADES = {2115, 4104, 4108, 4114, 4115, 4116, 4118, 4119, 4120, 4121}
-- Text of cards
local CARDS = {}
-- Drop chances
local CHANCES = {}

local getHtm

-- getHtm is Mobius's private getHtm(player, html, index, param3, name): the
-- page with both cards and the player's name filled in.
function getHtm(player, html_, index, param3, name)
  return J.replace(J.replace(J.replace(q:get_htm(player, html_), "%card1pic%", CARDS[index]), "%card2pic%", CARDS[param3]), "%name%", name)
end

CARDS[0] = "No such card"
CARDS[11] = "<font color=\"ff453d\"> Sun Card: 1 </font>"
CARDS[12] = "<font color=\"ff453d\"> Sun Card: 2 </font>"
CARDS[13] = "<font color=\"ff453d\"> Sun Card: 3 </font>"
CARDS[14] = "<font color=\"ff453d\"> Sun Card: 4 </font>"
CARDS[15] = "<font color=\"ff453d\"> Sun Card: 5 </font>"
CARDS[21] = "<font color=\"fff802\"> Moon Card: 1 </font>"
CARDS[22] = "<font color=\"fff802\"> Moon Card: 2 </font>"
CARDS[23] = "<font color=\"fff802\"> Moon Card: 3 </font>"
CARDS[24] = "<font color=\"fff802\"> Moon Card: 4 </font>"
CARDS[25] = "<font color=\"fff802\"> Moon Card: 5 </font>"
CHANCES[20674] = 807000 -- Doom Knight
CHANCES[20678] = 372000 -- Tortured Undead
CHANCES[20954] = 460000 -- Hungered Corpse
CHANCES[20955] = 537000 -- Ghost War
CHANCES[20956] = 540000 -- Past Knight
CHANCES[20957] = 565000 -- Nihil Invader
CHANCES[20958] = 425000 -- Death Agent
CHANCES[20959] = 682000 -- Dark Guard
CHANCES[20960] = 372000 -- Bloody Ghost
CHANCES[20961] = 547000 -- Bloody Knight
CHANCES[20962] = 522000 -- Bloody Priest
CHANCES[20963] = 498000 -- Bloody Lord
CHANCES[20974] = 1000000 -- Spiteful Soul Leader
CHANCES[20975] = 975000 -- Spiteful Soul Wizard
CHANCES[20976] = 825000 -- Spiteful Soul Fighter
CHANCES[20996] = 385000 -- Spiteful Ghost of Ruins
CHANCES[20997] = 342000 -- Soldier of Grief
CHANCES[20998] = 377000 -- Cruel Punisher
CHANCES[20999] = 450000 -- Roving Soul
CHANCES[21000] = 395000 -- Soul of Ruins
CHANCES[21001] = 535000 -- Wretched Archer
CHANCES[21002] = 472000 -- Doom Scout
CHANCES[21006] = 502000 -- Doom Servant
CHANCES[21007] = 540000 -- Doom Guard
CHANCES[21008] = 692000 -- Doom Archer
CHANCES[21009] = 740000 -- Doom Trooper
CHANCES[21010] = 595000 -- Doom Warrior

q:register_quest_items(SPIRIT_BEAD)
q:add_start_npc(WILBERT)
q:add_talk_id(WILBERT)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  local state = st:get_int("state")
  if event == "30846-03.htm" then
    st:start_quest()
    st:set("state", "1")
  elseif event == "30846-09.htm" then
    if math.fmod(state, 10) <= 4 then
      if J.idiv(state, 10) < 1 then
        if q:get_quest_items_count(player, SPIRIT_BEAD) >= 50 then
          q:take_items(player, SPIRIT_BEAD, 50)
          st:set("state", "5")
        else
          htmltext = "30846-10.htm"
        end
      else
        st:set("state", tostring(J.idiv(state, 10) * 10 + 5))
        st:set("stateEx", "0")
        htmltext = "30846-09a.htm"
      end
    end
  elseif event == "30846-14.htm" then
    if math.fmod(state, 10) == 5 and J.idiv(state, 1000) == 0 then
      local i0 = st:get_int("stateEx")
      local i1 = math.fmod(i0, 10)
      local i2 = J.idiv(i0 - i1, 10)
      local param1 = rnd(2) + 1
      local param2 = rnd(5) + 1
      local i5 = J.idiv(state, 10)
      local param3 = param1 * 10 + param2
      if param1 == i2 then
        local i3 = param2 + i1
        if math.fmod(i3, 5) == 0 and i3 ~= 10 then
          if J.idiv(math.fmod(state, 100), 10) >= 7 then
            st:set("state", "4")
            q:reward_items(player, ADENA, 2384000)
            q:reward_items(player, ENCHANT_WEAPON_A, 1)
            q:reward_items(player, ENCHANT_ARMOR_A, 1)
            q:reward_items(player, ENCHANT_ARMOR_A, 1)
            htmltext = getHtm(player, "30846-14.htm", i0, param3, player:name())
          else
            st:set("state", tostring(J.idiv(state, 10) * 10 + 7))
            htmltext = J.replace(getHtm(player, "30846-13.htm", i0, param3, player:name()), "%wincount%", tostring(i5 + 1))
          end
        else
          st:set("state", tostring(J.idiv(state, 10) * 10 + 6))
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-12.htm", i0, param3, player:name())
        end
      else
        if param2 == 5 or i1 == 5 then
          if J.idiv(math.fmod(state, 100), 10) >= 7 then
            st:set("state", "4")
            q:reward_items(player, ADENA, 2384000)
            q:reward_items(player, ENCHANT_WEAPON_A, 1)
            q:reward_items(player, ENCHANT_ARMOR_A, 1)
            q:reward_items(player, ENCHANT_ARMOR_A, 1)
            htmltext = getHtm(player, "30846-14.htm", i0, param3, player:name())
          else
            st:set("state", tostring(J.idiv(state, 10) * 10 + 7))
            htmltext = J.replace(getHtm(player, "30846-13.htm", i0, param3, player:name()), "%wincount%", tostring(i5 + 1))
          end
        else
          st:set("state", tostring(J.idiv(state, 10) * 10 + 6))
          st:set("stateEx", tostring(param1 * 10 + param2))
          htmltext = getHtm(player, "30846-12.htm", i0, param3, player:name())
        end
      end
    end
  elseif event == "30846-19.htm" then
    if math.fmod(state, 10) == 6 and J.idiv(state, 1000) == 0 then
      local i0 = st:get_int("stateEx")
      local i1 = math.fmod(i0, 10)
      local i2 = J.idiv(i0 - i1, 10)
      local param1 = rnd(2) + 1
      local param2 = rnd(5) + 1
      local param3 = param1 * 10 + param2
      if param1 == i2 then
        local i3 = param1 + i1 -- Mobius adds param1 here (param2 everywhere else); kept
        if math.fmod(i3, 5) == 0 and i3 ~= 10 then
          st:set("state", "1")
          st:set("stateEx", "0")
          htmltext = getHtm(player, "30846-19.htm", i0, param3, player:name())
        else
          st:set("state", tostring(J.idiv(state, 10) * 10 + 5))
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-18.htm", i0, param3, player:name())
        end
      else
        if param2 == 5 or i1 == 5 then
          st:set("state", "1")
          htmltext = getHtm(player, "30846-19.htm", i0, param3, player:name())
        else
          st:set("state", tostring(J.idiv(state, 10) * 10 + 5))
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-18.htm", i0, param3, player:name())
        end
      end
    end
  elseif event == "30846-20.htm" then
    if math.fmod(state, 10) == 7 and J.idiv(state, 1000) == 0 then
      st:set("state", tostring((J.idiv(state, 10) + 1) * 10 + 4))
      st:set("stateEx", "0")
    end
  elseif event == "30846-21.htm" then
    if math.fmod(state, 10) == 7 and J.idiv(state, 1000) == 0 then
      local round = J.idiv(state, 10)
      if round == 0 then
        q:reward_items(player, ADENA, 40000)
      elseif round == 1 then
        q:reward_items(player, ADENA, 80000)
      elseif round == 2 then
        q:reward_items(player, ADENA, 110000)
        q:reward_items(player, ENCHANT_WEAPON_D, 1)
      elseif round == 3 then
        q:reward_items(player, ADENA, 199000)
        q:reward_items(player, ENCHANT_WEAPON_C, 1)
      elseif round == 4 then
        q:reward_items(player, ADENA, 388000)
        q:reward_items(player, rnd.entry(RECIPES), 1)
      elseif round == 5 then
        q:reward_items(player, ADENA, 675000)
        q:reward_items(player, rnd.entry(BLADES), 1)
      elseif round == 6 then
        q:reward_items(player, ADENA, 1284000)
        q:reward_items(player, ENCHANT_WEAPON_B, 1)
        q:reward_items(player, ENCHANT_ARMOR_B, 1)
        q:reward_items(player, ENCHANT_WEAPON_B, 1)
        q:reward_items(player, ENCHANT_ARMOR_B, 1)
      end
      st:set("state", "1")
      st:set("stateEx", "0")
    end
  elseif event == "30846-22.htm" then
    if math.fmod(state, 10) == 1 then
      if q:has_quest_items(player, SPIRIT_BEAD) then
        st:set("state", "1005")
        q:take_items(player, SPIRIT_BEAD, 1)
      else
        htmltext = "30846-22a.htm"
      end
    end
  elseif event == "30846-25.htm" then
    if state == 1005 then
      local i0 = st:get_int("stateEx")
      local i1 = math.fmod(i0, 10)
      local i2 = J.idiv(i0 - i1, 10)
      local param1 = rnd(2) + 1
      local param2 = rnd(5) + 1
      local param3 = param1 * 10 + param2
      if param1 == i2 then
        local i3 = param2 + i1
        if math.fmod(i3, 5) == 0 and i3 ~= 10 then
          st:set("state", "1")
          st:set("stateEx", "0")
          q:reward_items(player, ADENA, 800)
          htmltext = J.replace(getHtm(player, "30846-25.htm", i0, param3, player:name()), "%card1%", tostring(i1))
        else
          st:set("state", "1006")
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-24.htm", i0, param3, player:name())
        end
      else
        if param2 == 5 or i2 == 5 then
          st:set("state", "1")
          st:set("stateEx", "0")
          q:reward_items(player, ADENA, 800)
          htmltext = J.replace(getHtm(player, "30846-25.htm", i0, param3, player:name()), "%card1%", tostring(i1))
        else
          st:set("state", "1006")
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-24.htm", i0, param3, player:name())
        end
      end
    end
  elseif event == "30846-29.htm" then
    if state == 1006 then
      local i0 = st:get_int("stateEx")
      local i1 = math.fmod(i0, 10)
      local i2 = J.idiv(i0 - i1, 10)
      local param1 = rnd(2) + 1
      local param2 = rnd(5) + 1
      local param3 = param1 * 10 + param2
      if param1 == i2 then
        local i3 = param2 + i1
        if math.fmod(i3, 5) == 0 and i3 ~= 10 then
          st:set("state", "1")
          st:set("stateEx", "0")
          q:reward_items(player, ADENA, 800)
          htmltext = J.replace(getHtm(player, "30846-29.htm", i0, param3, player:name()), "%card1%", tostring(i1))
        else
          st:set("state", "1005")
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-28.htm", i0, param3, player:name())
        end
      else
        if param2 == 5 or i1 == 5 then
          st:set("state", "1")
          st:set("stateEx", "0")
          htmltext = getHtm(player, "30846-29.htm", i0, param3, player:name())
        else
          st:set("state", "1005")
          st:set("stateEx", tostring(param3))
          htmltext = getHtm(player, "30846-28.htm", i0, param3, player:name())
        end
      end
    end
  elseif event == "30846-30.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 50 and "30846-02.htm" or "30846-01.htm"
  elseif sw == State.STARTED then
    local state = st:get_int("state")
    if state < 4 then
      if q:has_quest_items(player, SPIRIT_BEAD) then
        htmltext = "30846-05.htm"
      else
        htmltext = "30846-04.htm"
      end
    elseif math.fmod(state, 10) == 4 then
      htmltext = "30846-05a.htm"
    elseif math.fmod(state, 10) == 5 then
      htmltext = "30846-11.htm"
    elseif math.fmod(state, 10) == 6 then
      htmltext = "30846-15.htm"
    elseif math.fmod(state, 10) == 7 then
      local round = J.idiv(math.fmod(state, 100), 10)
      if round >= 7 then
        q:reward_items(player, ADENA, 2384000)
        q:reward_items(player, ENCHANT_WEAPON_A, 1)
        q:reward_items(player, ENCHANT_ARMOR_A, 1)
        q:reward_items(player, ENCHANT_ARMOR_A, 1)
        htmltext = "30846-17.htm"
      else
        htmltext = J.replace(q:get_htm(player, "30846-16.htm"), "%wincount%", tostring(J.idiv(state, 10) + 1))
      end
    elseif state == 1005 then
      htmltext = "30846-23.htm"
    elseif state == 1006 then
      htmltext = "30846-26.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    local partyMember = st:get_player()
    q:give_items(partyMember, SPIRIT_BEAD, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
