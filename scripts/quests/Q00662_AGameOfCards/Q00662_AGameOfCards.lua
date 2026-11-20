-- Translated by questport from quests/Q00662_AGameOfCards (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Finished by hand: giveReward's item template lookup (ItemData) is replaced
-- by the stackable flags of the few reward items.
local J = require("lib.java")

local q = quest.new(662, "Q00662_AGameOfCards", "A Game of Cards")

-- NPC
local KLUMP = 30845
-- Quest Item
local RED_GEM = 8765
-- Reward Items
local EW_S = 959
local EW_A = 729
local EW_B = 947
local EW_C = 951
local EW_D = 955
local EA_D = 956
local ZIGGO_GEMSTONE = 8868
-- All cards
local CARDS = {}
-- Drop chances
local CHANCES = {}

-- The rewards that stack (Mobius item data, is_stackable); the enchant
-- scrolls of Interlude do not.
local STACKABLE = {[ZIGGO_GEMSTONE] = true}

-- giveReward is Mobius's giveReward: a stackable item in one go, any other
-- one by one.
local function giveReward(player, item, count)
  if STACKABLE[item] then
    q:give_items(player, item, count)
  else
    for i = 0, count - 1 do
      q:give_items(player, item, 1)
    end
  end
end

CARDS[0] = "?"
CARDS[1] = "!"
CARDS[2] = "="
CARDS[3] = "T"
CARDS[4] = "V"
CARDS[5] = "O"
CARDS[6] = "P"
CARDS[7] = "S"
CARDS[8] = "E"
CARDS[9] = "H"
CARDS[10] = "A"
CARDS[11] = "R"
CARDS[12] = "D"
CARDS[13] = "I"
CARDS[14] = "N"
CHANCES[18001] = 232000 -- Blood Queen
CHANCES[20672] = 357000 -- Trives
CHANCES[20673] = 373000 -- Falibati
CHANCES[20674] = 583000 -- Doom Knight
CHANCES[20677] = 435000 -- Tulben
CHANCES[20955] = 358000 -- Ghostly Warrior
CHANCES[20958] = 283000 -- Death Agent
CHANCES[20959] = 455000 -- Dark Guard
CHANCES[20961] = 365000 -- Bloody Knight
CHANCES[20962] = 348000 -- Bloody Priest
CHANCES[20965] = 457000 -- Chimera Piece
CHANCES[20966] = 493000 -- Changed Creation
CHANCES[20968] = 418000 -- Nonexistant Man
CHANCES[20972] = 350000 -- Shaman of Ancient Times
CHANCES[20973] = 453000 -- Forgotten Ancient People
CHANCES[21002] = 315000 -- Doom Scout
CHANCES[21004] = 320000 -- Dismal Pole
CHANCES[21006] = 335000 -- Doom Servant
CHANCES[21008] = 462000 -- Doom Archer
CHANCES[21010] = 397000 -- Doom Warrior
CHANCES[21109] = 507000 -- Hames Orc Scout
CHANCES[21112] = 552000 -- Hames Orc Footman
CHANCES[21114] = 587000 -- Cursed Guardian
CHANCES[21116] = 812000 -- Hames Orc Overlord
CHANCES[21278] = 483000 -- Antelope
CHANCES[21279] = 483000 -- Antelope
CHANCES[21280] = 483000 -- Antelope
CHANCES[21281] = 483000 -- Antelope
CHANCES[21286] = 515000 -- Buffalo
CHANCES[21287] = 515000 -- Buffalo
CHANCES[21288] = 515000 -- Buffalo
CHANCES[21289] = 515000 -- Buffalo
CHANCES[21508] = 493000 -- Splinter Stakato
CHANCES[21510] = 527000 -- Splinter Stakato Soldier
CHANCES[21513] = 562000 -- Needle Stakato
CHANCES[21515] = 598000 -- Needle Stakato Soldier
CHANCES[21520] = 458000 -- Eye of Splendor
CHANCES[21526] = 552000 -- Wisdom of Splendor
CHANCES[21530] = 488000 -- Victory of Splendor
CHANCES[21535] = 573000 -- Signet of Splendor

-- fold maps a draw of 1..70 to a card 1..14.
local function fold(i)
  if i >= 57 then
    return i - 56
  elseif i >= 43 then
    return i - 42
  elseif i >= 29 then
    return i - 28
  elseif i >= 15 then
    return i - 14
  end
  return i
end

-- showCards fills the five cells of a card page: a face-down card ("?" in
-- yellow) until it is turned (in red).
local function showCards(htmltext, i9, i1, i2, i3, i4, i5)
  htmltext = J.replace(J.replace(htmltext, "%FontColor1%", math.fmod(i9, 2) < 1 and "ffff00" or "ff6f6f"), "%Cell1%", math.fmod(i9, 2) < 1 and CARDS[0] or CARDS[i1])
  htmltext = J.replace(J.replace(htmltext, "%FontColor2%", math.fmod(i9, 4) < 2 and "ffff00" or "ff6f6f"), "%Cell2%", math.fmod(i9, 4) < 2 and CARDS[0] or CARDS[i2])
  htmltext = J.replace(J.replace(htmltext, "%FontColor3%", math.fmod(i9, 8) < 4 and "ffff00" or "ff6f6f"), "%Cell3%", math.fmod(i9, 8) < 4 and CARDS[0] or CARDS[i3])
  htmltext = J.replace(J.replace(htmltext, "%FontColor4%", math.fmod(i9, 16) < 8 and "ffff00" or "ff6f6f"), "%Cell4%", math.fmod(i9, 16) < 8 and CARDS[0] or CARDS[i4])
  htmltext = J.replace(J.replace(htmltext, "%FontColor5%", math.fmod(i9, 32) < 16 and "ffff00" or "ff6f6f"), "%Cell5%", math.fmod(i9, 32) < 16 and CARDS[0] or CARDS[i5])
  return htmltext
end

q:register_quest_items(RED_GEM)
q:add_start_npc(KLUMP)
q:add_talk_id(KLUMP)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30845-03.htm" then
    st:start_quest()
    st:set("state", "0")
    st:set("stateEx", "0")
  elseif event == "30845-04.htm" then
    local state = st:get_int("state")
    local stateEx = st:get_int("stateEx")
    if state == 0 and stateEx == 0 and q:get_quest_items_count(player, RED_GEM) >= 50 then
      htmltext = "30845-05.htm"
    end
  elseif event == "30845-07.htm" then
    st:exit_quest(true, true)
  elseif event == "30845-11.htm" then
    local state = st:get_int("state")
    local stateEx = st:get_int("stateEx")
    if state == 0 and stateEx == 0 and q:get_quest_items_count(player, RED_GEM) >= 50 then
      -- Five draws in Mobius's order, then folded to cards.
      local i1 = rnd(70) + 1
      local i2 = rnd(70) + 1
      local i3 = rnd(70) + 1
      local i4 = rnd(70) + 1
      local i5 = rnd(70) + 1
      i1, i2, i3, i4, i5 = fold(i1), fold(i2), fold(i3), fold(i4), fold(i5)
      st:set("state", tostring(i4 * 1000000 + i3 * 10000 + i2 * 100 + i1))
      st:set("stateEx", tostring(i5))
      q:take_items(player, RED_GEM, 50)
    end
  elseif event == "First" or event == "Second" or event == "Third" or event == "Fourth" or event == "Fifth" then
    local state = st:get_int("state")
    local stateEx = st:get_int("stateEx")
    local i0 = state
    local i1 = stateEx
    local i5 = math.fmod(i1, 100)
    local i9 = J.idiv(i1, 100)
    i1 = math.fmod(i0, 100)
    local i2 = J.idiv(math.fmod(i0, 10000), 100)
    local i3 = J.idiv(math.fmod(i0, 1000000), 10000)
    local i4 = J.idiv(math.fmod(i0, 100000000), 1000000)
    if event == "First" then
      if math.fmod(i9, 2) < 1 then
        i9 = i9 + 1
      end
    elseif event == "Second" then
      if math.fmod(i9, 4) < 2 then
        i9 = i9 + 2
      end
    elseif event == "Third" then
      if math.fmod(i9, 8) < 4 then
        i9 = i9 + 4
      end
    elseif event == "Fourth" then
      if math.fmod(i9, 16) < 8 then
        i9 = i9 + 8
      end
    elseif event == "Fifth" then
      if math.fmod(i9, 32) < 16 then
        i9 = i9 + 16
      end
    end
    if math.fmod(i9, 32) < 31 then
      st:set("stateEx", tostring(i9 * 100 + i5))
      htmltext = q:get_htm(player, "30845-12.htm")
    elseif math.fmod(i9, 32) == 31 then
      local i6 = 0
      local i8 = 0
      if i1 >= 1 and i1 <= 14 and i2 >= 1 and i2 <= 14 and i3 >= 1 and i3 <= 14 and i4 >= 1 and i4 <= 14 and i5 >= 1 and i5 <= 14 then
        if i1 == i2 then
          i6 = i6 + 10
          i8 = i8 + 8
        end
        if i1 == i3 then
          i6 = i6 + 10
          i8 = i8 + 4
        end
        if i1 == i4 then
          i6 = i6 + 10
          i8 = i8 + 2
        end
        if i1 == i5 then
          i6 = i6 + 10
          i8 = i8 + 1
        end
        if math.fmod(i6, 100) < 10 then
          if math.fmod(i8, 16) < 8 then
            if math.fmod(i8, 8) < 4 and i2 == i3 then
              i6 = i6 + 10
              i8 = i8 + 4
            end
            if math.fmod(i8, 4) < 2 and i2 == i4 then
              i6 = i6 + 10
              i8 = i8 + 2
            end
            if math.fmod(i8, 2) < 1 and i2 == i5 then
              i6 = i6 + 10
              i8 = i8 + 1
            end
          end
        elseif math.fmod(i6, 10) == 0 and math.fmod(i8, 16) < 8 then
          if math.fmod(i8, 8) < 4 and i2 == i3 then
            i6 = i6 + 1
            i8 = i8 + 4
          end
          if math.fmod(i8, 4) < 2 and i2 == i4 then
            i6 = i6 + 1
            i8 = i8 + 2
          end
          if math.fmod(i8, 2) < 1 and i2 == i5 then
            i6 = i6 + 1
            i8 = i8 + 1
          end
        end
        if math.fmod(i6, 100) < 10 then
          if math.fmod(i8, 8) < 4 then
            if math.fmod(i8, 4) < 2 and i3 == i4 then
              i6 = i6 + 10
              i8 = i8 + 2
            end
            if math.fmod(i8, 2) < 1 and i3 == i5 then
              i6 = i6 + 10
              i8 = i8 + 1
            end
          end
        elseif math.fmod(i6, 10) == 0 and math.fmod(i8, 8) < 4 then
          if math.fmod(i8, 4) < 2 and i3 == i4 then
            i6 = i6 + 1
            i8 = i8 + 2
          end
          if math.fmod(i8, 2) < 1 and i3 == i5 then
            i6 = i6 + 1
            i8 = i8 + 1
          end
        end
        if math.fmod(i6, 100) < 10 then
          if math.fmod(i8, 4) < 2 and math.fmod(i8, 2) < 1 and i4 == i5 then
            i6 = i6 + 10
            i8 = i8 + 1
          end
        elseif math.fmod(i6, 10) == 0 and math.fmod(i8, 4) < 2 and math.fmod(i8, 2) < 1 and i4 == i5 then
          i6 = i6 + 1
          i8 = i8 + 1
        end
      end
      if i6 == 40 then
        giveReward(player, ZIGGO_GEMSTONE, 43)
        giveReward(player, EW_S, 3)
        giveReward(player, EW_A, 1)
        htmltext = q:get_htm(player, "30845-13.htm")
      elseif i6 == 30 then
        giveReward(player, EW_S, 2)
        giveReward(player, EW_C, 2)
        htmltext = q:get_htm(player, "30845-14.htm")
      elseif i6 == 21 or i6 == 12 then
        giveReward(player, EW_A, 1)
        giveReward(player, EW_B, 2)
        giveReward(player, EW_D, 1)
        htmltext = q:get_htm(player, "30845-15.htm")
      elseif i6 == 20 then
        giveReward(player, EW_C, 2)
        htmltext = q:get_htm(player, "30845-16.htm")
      elseif i6 == 11 then
        giveReward(player, EW_C, 1)
        htmltext = q:get_htm(player, "30845-17.htm")
      elseif i6 == 10 then
        giveReward(player, EA_D, 2)
        htmltext = q:get_htm(player, "30845-18.htm")
      elseif i6 == 0 then
        htmltext = q:get_htm(player, "30845-19.htm")
      end
      st:set("state", "0")
      st:set("stateEx", "0")
    end
    htmltext = showCards(htmltext, i9, i1, i2, i3, i4, i5)
  elseif event == "30845-20.htm" then
    if q:get_quest_items_count(player, RED_GEM) < 50 then
      htmltext = "30845-21.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 61 and "30845-02.htm" or "30845-01.htm"
  elseif sw == State.STARTED then
    local state = st:get_int("state")
    local stateEx = st:get_int("stateEx")
    if state == 0 and stateEx == 0 then
      htmltext = q:get_quest_items_count(player, RED_GEM) < 50 and "30845-04.htm" or "30845-05.htm"
    elseif state ~= 0 and stateEx ~= 0 then
      local i0 = state
      local i1 = stateEx
      local i5 = math.fmod(i1, 100)
      local i9 = J.idiv(i1, 100)
      i1 = math.fmod(i0, 100)
      local i2 = J.idiv(math.fmod(i0, 10000), 100)
      local i3 = J.idiv(math.fmod(i0, 1000000), 10000)
      local i4 = J.idiv(math.fmod(i0, 100000000), 1000000)
      htmltext = showCards(q:get_htm(player, "30845-11a.htm"), i9, i1, i2, i3, i4, i5)
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
    q:give_items(partyMember, RED_GEM, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
