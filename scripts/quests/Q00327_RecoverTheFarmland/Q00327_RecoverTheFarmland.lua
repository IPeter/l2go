-- Translated by questport from quests/Q00327_RecoverTheFarmland (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(327, "Q00327_RecoverTheFarmland", "Recover the Farmland")

-- NPCs
local LEIKAN = 30382
local PIOTUR = 30597
local IRIS = 30034
local ASHA = 30313
local NESTLE = 30314
-- Monsters
local TUREK_ORC_WARLORD = 20495
local TUREK_ORC_ARCHER = 20496
local TUREK_ORC_SKIRMISHER = 20497
local TUREK_ORC_SUPPLIER = 20498
local TUREK_ORC_FOOTMAN = 20499
local TUREK_ORC_SENTINEL = 20500
local TUREK_ORC_SHAMAN = 20501
-- Items
local LEIKAN_LETTER = 5012
local TUREK_DOGTAG = 1846
local TUREK_MEDALLION = 1847
local CLAY_URN_FRAGMENT = 1848
local BRASS_TRINKET_PIECE = 1849
local BRONZE_MIRROR_PIECE = 1850
local JADE_NECKLACE_BEAD = 1851
local ANCIENT_CLAY_URN = 1852
local ANCIENT_BRASS_TIARA = 1853
local ANCIENT_BRONZE_MIRROR = 1854
local ANCIENT_JADE_NECKLACE = 1855
-- Rewards
local ADENA = 57
local SOULSHOT_D = 1463
local SPIRITSHOT_D = 2510
local HEALING_POTION = 1061
local HASTE_POTION = 734
local POTION_OF_ALACRITY = 735
local SCROLL_OF_ESCAPE = 736
local SCROLL_OF_RESURRECTION = 737
-- Chances
local DROPLIST = {
  {TUREK_ORC_ARCHER, 140000, TUREK_DOGTAG},
  {TUREK_ORC_SKIRMISHER, 70000, TUREK_DOGTAG},
  {TUREK_ORC_SUPPLIER, 120000, TUREK_DOGTAG},
  {TUREK_ORC_FOOTMAN, 100000, TUREK_DOGTAG},
  {TUREK_ORC_SENTINEL, 80000, TUREK_DOGTAG},
  {TUREK_ORC_SHAMAN, 90000, TUREK_MEDALLION},
  {TUREK_ORC_WARLORD, 180000, TUREK_MEDALLION},
}
-- Exp
local EXP_REWARD = {}

EXP_REWARD[ANCIENT_CLAY_URN] = 2766
EXP_REWARD[ANCIENT_BRASS_TIARA] = 3227
EXP_REWARD[ANCIENT_BRONZE_MIRROR] = 3227
EXP_REWARD[ANCIENT_JADE_NECKLACE] = 3919

q:register_quest_items(LEIKAN_LETTER)
q:add_start_npc(LEIKAN, PIOTUR)
q:add_talk_id(LEIKAN, PIOTUR, IRIS, ASHA, NESTLE)
q:add_kill_id(TUREK_ORC_WARLORD, TUREK_ORC_ARCHER, TUREK_ORC_SKIRMISHER, TUREK_ORC_SUPPLIER, TUREK_ORC_FOOTMAN, TUREK_ORC_SENTINEL, TUREK_ORC_SHAMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30597-03.htm" then
    if st:get_cond() < 1 then
      st:start_quest()
    end
  elseif event == "30597-06.htm" then
    st:exit_quest(true, true)
  elseif event == "30382-03.htm" then
    st:start_quest()
    st:set_cond(2)
    q:give_items(player, LEIKAN_LETTER, 1)
  elseif event == "30313-02.htm" then
    if q:get_quest_items_count(player, CLAY_URN_FRAGMENT) >= 5 then
      q:take_items(player, CLAY_URN_FRAGMENT, 5)
      if rnd(6) < 5 then
        htmltext = "30313-03.htm"
        q:reward_items(player, ANCIENT_CLAY_URN, 1)
      else
        htmltext = "30313-10.htm"
      end
    end
  elseif event == "30313-04.htm" then
    if q:get_quest_items_count(player, BRASS_TRINKET_PIECE) >= 5 then
      q:take_items(player, BRASS_TRINKET_PIECE, 5)
      if rnd(7) < 6 then
        htmltext = "30313-05.htm"
        q:reward_items(player, ANCIENT_BRASS_TIARA, 1)
      else
        htmltext = "30313-10.htm"
      end
    end
  elseif event == "30313-06.htm" then
    if q:get_quest_items_count(player, BRONZE_MIRROR_PIECE) >= 5 then
      q:take_items(player, BRONZE_MIRROR_PIECE, 5)
      if rnd(7) < 6 then
        htmltext = "30313-07.htm"
        q:reward_items(player, ANCIENT_BRONZE_MIRROR, 1)
      else
        htmltext = "30313-10.htm"
      end
    end
  elseif event == "30313-08.htm" then
    if q:get_quest_items_count(player, JADE_NECKLACE_BEAD) >= 5 then
      q:take_items(player, JADE_NECKLACE_BEAD, 5)
      if rnd(8) < 7 then
        htmltext = "30313-09.htm"
        q:reward_items(player, ANCIENT_JADE_NECKLACE, 1)
      else
        htmltext = "30313-10.htm"
      end
    end
  elseif event == "30034-03.htm" then
    local n = q:get_quest_items_count(player, CLAY_URN_FRAGMENT)
    if n == 0 then
      htmltext = "30034-02.htm"
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, CLAY_URN_FRAGMENT, n)
      q:add_exp_and_sp(player, n * 307, 0)
    end
  elseif event == "30034-04.htm" then
    local n = q:get_quest_items_count(player, BRASS_TRINKET_PIECE)
    if n == 0 then
      htmltext = "30034-02.htm"
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, BRASS_TRINKET_PIECE, n)
      q:add_exp_and_sp(player, n * 368, 0)
    end
  elseif event == "30034-05.htm" then
    local n = q:get_quest_items_count(player, BRONZE_MIRROR_PIECE)
    if n == 0 then
      htmltext = "30034-02.htm"
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, BRONZE_MIRROR_PIECE, n)
      q:add_exp_and_sp(player, n * 368, 0)
    end
  elseif event == "30034-06.htm" then
    local n = q:get_quest_items_count(player, JADE_NECKLACE_BEAD)
    if n == 0 then
      htmltext = "30034-02.htm"
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, JADE_NECKLACE_BEAD, n)
      q:add_exp_and_sp(player, n * 430, 0)
    end
  elseif event == "30034-07.htm" then
    local isRewarded = false
    for i = 1852, 1855 do
      local n = q:get_quest_items_count(player, i)
      if n > 0 then
        q:take_items(player, i, n)
        q:add_exp_and_sp(player, n * EXP_REWARD[i], 0)
        isRewarded = true
      end
    end
    if not isRewarded then
      htmltext = "30034-02.htm"
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30314-03.htm" then
    if not q:has_quest_items(player, ANCIENT_CLAY_URN) then
      htmltext = "30314-07.htm"
    else
      q:take_items(player, ANCIENT_CLAY_URN, 1)
      q:reward_items(player, SOULSHOT_D, 70 + rnd(41))
    end
  elseif event == "30314-04.htm" then
    if not q:has_quest_items(player, ANCIENT_BRASS_TIARA) then
      htmltext = "30314-07.htm"
    else
      q:take_items(player, ANCIENT_BRASS_TIARA, 1)
      local rnd_ = rnd(100)
      if rnd_ < 40 then
        q:reward_items(player, HEALING_POTION, 1)
      elseif rnd_ < 84 then
        q:reward_items(player, HASTE_POTION, 1)
      else
        q:reward_items(player, POTION_OF_ALACRITY, 1)
      end
    end
  elseif event == "30314-05.htm" then
    if not q:has_quest_items(player, ANCIENT_BRONZE_MIRROR) then
      htmltext = "30314-07.htm"
    else
      q:take_items(player, ANCIENT_BRONZE_MIRROR, 1)
      q:reward_items(player, rnd(100) < 59 and SCROLL_OF_ESCAPE or SCROLL_OF_RESURRECTION, 1)
    end
  elseif event == "30314-06.htm" then
    if not q:has_quest_items(player, ANCIENT_JADE_NECKLACE) then
      htmltext = "30314-07.htm"
    else
      q:take_items(player, ANCIENT_JADE_NECKLACE, 1)
      q:reward_items(player, SPIRITSHOT_D, 50 + rnd(41))
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = npc:id() .. (player:level() < 25 and "-01.htm" or "-02.htm")
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == PIOTUR then
      if not q:has_quest_items(player, LEIKAN_LETTER) then
        if q:has_at_least_one_quest_item(player, TUREK_DOGTAG, TUREK_MEDALLION) then
          htmltext = "30597-05.htm"
          if cond < 4 then
            st:set_cond(4, true)
          end
          local dogtag = q:get_quest_items_count(player, TUREK_DOGTAG)
          local medallion = q:get_quest_items_count(player, TUREK_MEDALLION)
          q:take_items(player, TUREK_DOGTAG, -1)
          q:take_items(player, TUREK_MEDALLION, -1)
          q:reward_items(player, ADENA, dogtag * 40 + medallion * 50 + (dogtag + medallion >= 10 and 619 or 0))
        else
          htmltext = "30597-04.htm"
        end
      else
        htmltext = "30597-03a.htm"
        st:set_cond(3, true)
        q:take_items(player, LEIKAN_LETTER, 1)
      end
    elseif sw2 == LEIKAN then
      if cond == 2 then
        htmltext = "30382-04.htm"
      elseif cond == 3 or cond == 4 then
        htmltext = "30382-05.htm"
        st:set_cond(5, true)
      elseif cond == 5 then
        htmltext = "30382-05.htm"
      end
    else
      htmltext = npc:id() .. "-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  for _, npcData in ipairs(DROPLIST) do
    if npcData[1] == npc:id() then
      q:give_items(player, npcData[3], 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      if rnd(1000000) < npcData[2] then
        q:give_items(player, rnd(1848, 1851), 1)
      end
      break
    end
  end
end)
