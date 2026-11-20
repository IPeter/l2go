-- Translated by questport from quests/Q00999_T0Tutorial (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(-1, "Q00999_T0Tutorial", "")

-- Items
local RECOMMENDATION_01 = 1067
local RECOMMENDATION_02 = 1068
local LEAF_OF_MOTHERTREE = 1069
local BLOOD_OF_JUNDIN = 1070
local LICENSE_OF_MINER = 1498
local VOUCHER_OF_FLAME = 1496
local SOULSHOT_NOVICE = 5789
local SPIRITSHOT_NOVICE = 5790
local BLUE_GEM = 6353
-- NPCs
local NPCS = {
  30008,
  30009,
  30017,
  30019,
  30129,
  30131,
  30573,
  30575,
  30370,
  30528,
  30530,
  30400,
  30401,
  30402,
  30403,
  30404,
}
local Event = {}
local Talks = {}

local Talk = {}
Talk.__index = Talk

function Talk.new(raceId, htmlfiles, npcType, item)
  local self = setmetatable({}, Talk)
  self._raceId = 0
  self._npcType = 0
  self._item = 0
  self._raceId = raceId
  self._htmlfiles = htmlfiles
  self._npcType = npcType
  self._item = item
  return self
end

Event["30008_02"] = {"30008-03.htm", -84058, 243239, -3730, RECOMMENDATION_01, 0, SOULSHOT_NOVICE, 200, 0, 0, 0}
Event["30008_04"] = {"30008-04.htm", -84058, 243239, -3730, 0, 0, 0, 0, 0, 0, 0}
Event["30017_02"] = {"30017-03.htm", -84058, 243239, -3730, RECOMMENDATION_02, 10, SPIRITSHOT_NOVICE, 100, 0, 0, 0}
Event["30017_04"] = {"30017-04.htm", -84058, 243239, -3730, 0, 10, 0, 0, 0, 0, 0}
Event["30370_02"] = {
  "30370-03.htm",
  45491,
  48359,
  -3086,
  LEAF_OF_MOTHERTREE,
  25,
  SPIRITSHOT_NOVICE,
  100,
  18,
  SOULSHOT_NOVICE,
  200,
}
Event["30370_04"] = {"30370-04.htm", 45491, 48359, -3086, 0, 25, 0, 0, 18, 0, 0}
Event["30129_02"] = {
  "30129-03.htm",
  12116,
  16666,
  -4610,
  BLOOD_OF_JUNDIN,
  38,
  SPIRITSHOT_NOVICE,
  100,
  31,
  SOULSHOT_NOVICE,
  200,
}
Event["30129_04"] = {"30129-04.htm", 12116, 16666, -4610, 0, 38, 0, 0, 31, 0, 0}
Event["30528_02"] = {"30528-03.htm", 115642, -178046, -941, LICENSE_OF_MINER, 53, SOULSHOT_NOVICE, 200, 0, 0, 0}
Event["30528_04"] = {"30528-04.htm", 115642, -178046, -941, 0, 53, 0, 0, 0, 0, 0}
Event["30573_02"] = {
  "30573-03.htm",
  -45067,
  -113549,
  -235,
  VOUCHER_OF_FLAME,
  49,
  SPIRITSHOT_NOVICE,
  100,
  44,
  SOULSHOT_NOVICE,
  200,
}
Event["30573_04"] = {"30573-04.htm", -45067, -113549, -235, 0, 49, 0, 0, 44, 0, 0}
Talks[30017] = Talk.new(0, {"30017-01.htm", "30017-02.htm", "30017-04.htm"}, 0, 0)
Talks[30008] = Talk.new(0, {"30008-01.htm", "30008-02.htm", "30008-04.htm"}, 0, 0)
Talks[30370] = Talk.new(1, {"30370-01.htm", "30370-02.htm", "30370-04.htm"}, 0, 0)
Talks[30129] = Talk.new(2, {"30129-01.htm", "30129-02.htm", "30129-04.htm"}, 0, 0)
Talks[30573] = Talk.new(3, {"30573-01.htm", "30573-02.htm", "30573-04.htm"}, 0, 0)
Talks[30528] = Talk.new(4, {"30528-01.htm", "30528-02.htm", "30528-04.htm"}, 0, 0)
Talks[30018] = Talk.new(0, {"30131-01.htm", "", "30019-03a.htm", "30019-04.htm"}, 1, RECOMMENDATION_02)
Talks[30019] = Talk.new(0, {"30131-01.htm", "", "30019-03a.htm", "30019-04.htm"}, 1, RECOMMENDATION_02)
Talks[30020] = Talk.new(0, {"30131-01.htm", "", "30019-03a.htm", "30019-04.htm"}, 1, RECOMMENDATION_02)
Talks[30021] = Talk.new(0, {"30131-01.htm", "", "30019-03a.htm", "30019-04.htm"}, 1, RECOMMENDATION_02)
Talks[30009] = Talk.new(0, {"30530-01.htm", "30009-03.htm", "", "30009-04.htm"}, 1, RECOMMENDATION_01)
Talks[30011] = Talk.new(0, {"30530-01.htm", "30009-03.htm", "", "30009-04.htm"}, 1, RECOMMENDATION_01)
Talks[30012] = Talk.new(0, {"30530-01.htm", "30009-03.htm", "", "30009-04.htm"}, 1, RECOMMENDATION_01)
Talks[30056] = Talk.new(0, {"30530-01.htm", "30009-03.htm", "", "30009-04.htm"}, 1, RECOMMENDATION_01)
Talks[30400] = Talk.new(1, {"30131-01.htm", "30400-03.htm", "30400-03a.htm", "30400-04.htm"}, 1, LEAF_OF_MOTHERTREE)
Talks[30401] = Talk.new(1, {"30131-01.htm", "30400-03.htm", "30400-03a.htm", "30400-04.htm"}, 1, LEAF_OF_MOTHERTREE)
Talks[30402] = Talk.new(1, {"30131-01.htm", "30400-03.htm", "30400-03a.htm", "30400-04.htm"}, 1, LEAF_OF_MOTHERTREE)
Talks[30403] = Talk.new(1, {"30131-01.htm", "30400-03.htm", "30400-03a.htm", "30400-04.htm"}, 1, LEAF_OF_MOTHERTREE)
Talks[30131] = Talk.new(2, {"30131-01.htm", "30131-03.htm", "30131-03a.htm", "30131-04.htm"}, 1, BLOOD_OF_JUNDIN)
Talks[30404] = Talk.new(2, {"30131-01.htm", "30131-03.htm", "30131-03a.htm", "30131-04.htm"}, 1, BLOOD_OF_JUNDIN)
Talks[30574] = Talk.new(3, {"30575-01.htm", "30575-03.htm", "30575-03a.htm", "30575-04.htm"}, 1, VOUCHER_OF_FLAME)
Talks[30575] = Talk.new(3, {"30575-01.htm", "30575-03.htm", "30575-03a.htm", "30575-04.htm"}, 1, VOUCHER_OF_FLAME)
Talks[30530] = Talk.new(4, {"30530-01.htm", "30530-03.htm", "", "30530-04.htm"}, 1, LICENSE_OF_MINER)
for _, startNpc in ipairs(NPCS) do
  q:add_start_npc(startNpc)
end
for _, FirstTalkId in ipairs(NPCS) do
  q:add_first_talk_id(FirstTalkId)
end
for _, TalkId in ipairs(NPCS) do
  q:add_talk_id(TalkId)
end
q:add_kill_id(18342)
q:add_kill_id(20001)

q:on_event(function(event, npc, player)
  if Config.DISABLE_TUTORIAL then
    return nil
  end
  local qs = player:get_quest_state("Q00255_Tutorial")
  if qs == nil then
    return event
  end
  local htmltext = event
  local st = player:get_quest_state(q:name())
  local Ex = qs:get_int("Ex")
  if J.equals_ignore_case(event, "TimerEx_NewbieHelper") then
    if Ex == 0 then
      if ClassId.is_mage(player:class_id()) then
        st:play_tutorial_voice("tutorial_voice_009b")
      else
        st:play_tutorial_voice("tutorial_voice_009a")
      end
      qs:set("Ex", "1")
    elseif Ex == 3 then
      st:play_tutorial_voice("tutorial_voice_010a")
      qs:set("Ex", "4")
    end
    return nil
  elseif J.equals_ignore_case(event, "TimerEx_GrandMaster") then
    if Ex >= 4 then
      st:show_question_mark(7)
      q:play_sound(player, "ItemSound.quest_tutorial")
      st:play_tutorial_voice("tutorial_voice_025")
    end
    return nil
  else
    local map = Event[event]
    htmltext = map[1]
    local radarX = map[2]
    local radarY = map[3]
    local radarZ = map[4]
    local item = map[5]
    local classId1 = map[6]
    local gift1 = map[7]
    local count1 = map[8]
    local classId2 = map[9]
    local gift2 = map[10]
    local count2 = map[11]
    if radarX ~= 0 then
      q:add_radar(player, radarX, radarY, radarZ)
    end
    if q:get_quest_items_count(player, item) > 0 and st:get_int("onlyone") == 0 then
      q:add_exp_and_sp(player, 0, 50)
      q:start_quest_timer("TimerEx_GrandMaster", 60000, npc, player)
      q:take_items(player, item, 1)
      st:set("step", "3")
      if Ex <= 3 then
        qs:set("Ex", "4")
      end
      if player:class_id() == classId1 then
        q:give_items(player, gift1, count1)
        if gift1 == SPIRITSHOT_NOVICE then
          st:play_tutorial_voice("tutorial_voice_027")
        else
          st:play_tutorial_voice("tutorial_voice_026")
        end
      elseif player:class_id() == classId2 and gift2 ~= 0 then
        q:give_items(player, gift2, count2)
        st:play_tutorial_voice("tutorial_voice_026")
      end
      st:set("step", "4")
      st:set("onlyone", "1")
    end
  end
  return htmltext
end)

q:on_first_talk(function(npc, player)
  if Config.DISABLE_TUTORIAL then
    return nil
  end
  local htmltext = ""
  local st = player:get_quest_state(q:name())
  if st == nil then
    st = q:new_quest_state(player)
  end
  local qs = player:get_quest_state("Q00255_Tutorial")
  if qs == nil or qs:is_completed() then
    npc:show_chat_window(player)
    return nil
  end
  local onlyone = st:get_int("onlyone")
  local Ex = qs:get_int("Ex")
  local step = st:get_int("step")
  local talk = Talks[npc:id()]
  if (player:level() >= 10 or onlyone ~= 0) and talk._npcType == 1 then
    htmltext = "30575-05.htm"
  elseif onlyone == 0 and player:level() < 10 then
    if talk ~= nil and player:race() == talk._raceId then
      htmltext = talk._htmlfiles[1]
      if talk._npcType == 1 then
        if step == 0 and Ex < 0 then
          qs:set("Ex", "0")
          q:start_quest_timer("TimerEx_NewbieHelper", 30000, npc, player)
          if ClassId.is_mage(player:class_id()) then
            st:set("step", "1")
            st:set_state(State.STARTED)
          else
            htmltext = "30530-01.htm"
            st:set("step", "1")
            st:set_state(State.STARTED)
          end
        elseif step == 1 and not q:has_quest_items(player, talk._item) and Ex <= 2 then
          if q:has_quest_items(player, BLUE_GEM) then
            q:take_items(player, BLUE_GEM, -1)
            q:give_items(player, talk._item, 1)
            st:set("step", "2")
            qs:set("Ex", "3")
            q:start_quest_timer("TimerEx_NewbieHelper", 30000, npc, player)
            qs:set("ucMemo", "3")
            if ClassId.is_mage(player:class_id()) then
              st:play_tutorial_voice("tutorial_voice_027")
              q:give_items(player, SPIRITSHOT_NOVICE, 100)
              htmltext = talk._htmlfiles[3]
              if htmltext == "" then
                htmltext = "<html><body>I`m sorry. I only help warriors. Please go to another Newbie Helper who may assist you.</body></html>"
              end
            else
              st:play_tutorial_voice("tutorial_voice_026")
              q:give_items(player, SOULSHOT_NOVICE, 200)
              htmltext = talk._htmlfiles[2]
              if htmltext == "" then
                htmltext = "<html><body>I`m sorry. I only help mystics. Please go to another Newbie Helper who may assist you.</body></html>"
              end
            end
          else
            if ClassId.is_mage(player:class_id()) then
              if player:race() == Race.ORC then
                htmltext = "30575-02.htm"
              else
                htmltext = "30131-02.htm"
              end
            else
              htmltext = "30530-02.htm"
            end
          end
        elseif step == 2 then
          htmltext = talk._htmlfiles[4]
        end
      elseif talk._npcType == 0 then
        if step == 1 then
          htmltext = talk._htmlfiles[1]
        elseif step == 2 then
          htmltext = talk._htmlfiles[2]
        elseif step == 3 then
          htmltext = talk._htmlfiles[3]
        end
      end
    end
  elseif step == 4 and player:level() < 10 then
    htmltext = npc:id() .. "-04.htm"
  end
  if htmltext == nil or htmltext == "" then
    npc:show_chat_window(player)
  end
  npc:show_chat_window(player)
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return
  end
  local qs = player:get_quest_state("Q00255_Tutorial")
  if qs == nil then
    return
  end
  local Ex = qs:get_int("Ex")
  if Ex <= 1 then
    st:play_tutorial_voice("tutorial_voice_011")
    st:show_question_mark(3)
    qs:set("Ex", "2")
  end
  if Ex == 2 and not q:has_quest_items(player, BLUE_GEM) then
    npc:drop_item(player, BLUE_GEM, 1)
    q:play_sound(player, "ItemSound.quest_tutorial")
    st:set("step", "1")
  end
end)
