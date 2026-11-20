-- Translated by questport from quests/Q00255_Tutorial (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(-1, "Q00255_Tutorial", "Tutorial")

-- table for Quest Timer ( Ex == -2 ) [raceId, voice, html]
local QTEXMTWO = {
  {"0", "tutorial_voice_001a", "tutorial_human_fighter001.htm"},
  {"10", "tutorial_voice_001b", "tutorial_human_mage001.htm"},
  {"18", "tutorial_voice_001c", "tutorial_elven_fighter001.htm"},
  {"25", "tutorial_voice_001d", "tutorial_elven_mage001.htm"},
  {"31", "tutorial_voice_001e", "tutorial_delf_fighter001.htm"},
  {"38", "tutorial_voice_001f", "tutorial_delf_mage001.htm"},
  {"44", "tutorial_voice_001g", "tutorial_orc_fighter001.htm"},
  {"49", "tutorial_voice_001h", "tutorial_orc_mage001.htm"},
  {"53", "tutorial_voice_001i", "tutorial_dwarven_fighter001.htm"},
}
-- table for Client Event Enable (8) [raceId, html, x, y, z]
local CEEa = {
  {"0", "tutorial_human_fighter007.htm", "-71424", "258336", "-3109"},
  {"10", "tutorial_human_mage007.htm", "-91036", "248044", "-3568"},
  {"18", "tutorial_elf007.htm", "46112", "41200", "-3504"},
  {"25", "tutorial_elf007.htm", "46112", "41200", "-3504"},
  {"31", "tutorial_delf007.htm", "28384", "11056", "-4233"},
  {"38", "tutorial_delf007.htm", "28384", "11056", "-4233"},
  {"44", "tutorial_orc007.htm", "-56736", "-113680", "-672"},
  {"49", "tutorial_orc007.htm", "-56736", "-113680", "-672"},
  {"53", "tutorial_dwarven_fighter007.htm", "108567", "-173994", "-406"},
}
-- table for Question Mark Clicked (9 & 11) learning skills [raceId, html, x, y, z]
local QMCa = {
  {"0", "tutorial_fighter017.htm", "-83165", "242711", "-3720"},
  {"10", "tutorial_mage017.htm", "-85247", "244718", "-3720"},
  {"18", "tutorial_fighter017.htm", "45610", "52206", "-2792"},
  {"25", "tutorial_mage017.htm", "45610", "52206", "-2792"},
  {"31", "tutorial_fighter017.htm", "10344", "14445", "-4242"},
  {"38", "tutorial_mage017.htm", "10344", "14445", "-4242"},
  {"44", "tutorial_fighter017.htm", "-46324", "-114384", "-200"},
  {"49", "tutorial_fighter017.htm", "-46305", "-112763", "-200"},
  {"53", "tutorial_fighter017.htm", "115447", "-182672", "-1440"},
}
-- table for Question Mark Clicked (24) newbie lvl [raceId, html]
local QMCb = {}
-- table for Question Mark Clicked (35) 1st class transfer [raceId, html]
local QMCc = {}
-- table for Tutorial Close Link (26) 2nd class transfer [raceId, html]
local TCLa = {}
-- table for Tutorial Close Link (23) 2nd class transfer [raceId, html]
local TCLb = {}
-- table for Tutorial Close Link (24) 2nd class transfer [raceId, html]
local TCLc = {}

QMCb[0] = "tutorial_human009.htm"
QMCb[10] = "tutorial_human009.htm"
QMCb[18] = "tutorial_elf009.htm"
QMCb[25] = "tutorial_elf009.htm"
QMCb[31] = "tutorial_delf009.htm"
QMCb[38] = "tutorial_delf009.htm"
QMCb[44] = "tutorial_orc009.htm"
QMCb[49] = "tutorial_orc009.htm"
QMCb[53] = "tutorial_dwarven009.htm"
QMCc[0] = "tutorial_21.htm"
QMCc[10] = "tutorial_21a.htm"
QMCc[18] = "tutorial_21b.htm"
QMCc[25] = "tutorial_21c.htm"
QMCc[31] = "tutorial_21g.htm"
QMCc[38] = "tutorial_21h.htm"
QMCc[44] = "tutorial_21d.htm"
QMCc[49] = "tutorial_21e.htm"
QMCc[53] = "tutorial_21f.htm"
QMCc[53] = "tutorial_21f.htm"
TCLa[1] = "tutorial_22w.htm"
TCLa[4] = "tutorial_22.htm"
TCLa[7] = "tutorial_22b.htm"
TCLa[11] = "tutorial_22c.htm"
TCLa[15] = "tutorial_22d.htm"
TCLa[19] = "tutorial_22e.htm"
TCLa[22] = "tutorial_22f.htm"
TCLa[26] = "tutorial_22g.htm"
TCLa[29] = "tutorial_22h.htm"
TCLa[32] = "tutorial_22n.htm"
TCLa[35] = "tutorial_22o.htm"
TCLa[39] = "tutorial_22p.htm"
TCLa[42] = "tutorial_22q.htm"
TCLa[45] = "tutorial_22i.htm"
TCLa[47] = "tutorial_22j.htm"
TCLa[50] = "tutorial_22k.htm"
TCLa[54] = "tutorial_22l.htm"
TCLa[56] = "tutorial_22m.htm"
TCLb[4] = "tutorial_22aa.htm"
TCLb[7] = "tutorial_22ba.htm"
TCLb[11] = "tutorial_22ca.htm"
TCLb[15] = "tutorial_22da.htm"
TCLb[19] = "tutorial_22ea.htm"
TCLb[22] = "tutorial_22fa.htm"
TCLb[26] = "tutorial_22ga.htm"
TCLb[32] = "tutorial_22na.htm"
TCLb[35] = "tutorial_22oa.htm"
TCLb[39] = "tutorial_22pa.htm"
TCLb[50] = "tutorial_22ka.htm"
TCLc[4] = "tutorial_22ab.htm"
TCLc[7] = "tutorial_22bb.htm"
TCLc[11] = "tutorial_22cb.htm"
TCLc[15] = "tutorial_22db.htm"
TCLc[19] = "tutorial_22eb.htm"
TCLc[22] = "tutorial_22fb.htm"
TCLc[26] = "tutorial_22gb.htm"
TCLc[32] = "tutorial_22nb.htm"
TCLc[35] = "tutorial_22ob.htm"
TCLc[39] = "tutorial_22pb.htm"
TCLc[50] = "tutorial_22kb.htm"

q:on_event(function(event, npc, player)
  if Config.DISABLE_TUTORIAL then
    return ""
  end
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local html_ = ""
  local classId = player:class_id()
  local Ex = qs:get_int("Ex")
  if J.starts_with(event, "UC") then
    if player:level() < 6 and qs:get_int("onlyone") == 0 then
      local sw = qs:get_int("ucMemo")
      if sw == 0 then
        qs:set("ucMemo", "0")
        q:start_quest_timer("QT", 10000, nil, player, false)
        qs:set("Ex", "-2")
      elseif sw == 1 then
        qs:show_question_mark(1)
        qs:play_tutorial_voice("tutorial_voice_006")
        q:play_sound(player, "ItemSound.quest_tutorial")
      elseif sw == 2 then
        if Ex == 2 then
          qs:show_question_mark(3)
        elseif q:get_quest_items_count(player, 6353) > 0 then
          qs:show_question_mark(5)
        end
        q:play_sound(player, "ItemSound.quest_tutorial")
      elseif sw == 3 then
        qs:show_question_mark(12)
        q:play_sound(player, "ItemSound.quest_tutorial")
        qs:on_tutorial_client_event(0)
      end
    end
  elseif J.starts_with(event, "QT") then
    if Ex == -2 then
      local voice = ""
      for _, element in ipairs(QTEXMTWO) do
        if classId == J.parse_int(element[1]) then
          voice = element[2]
          html_ = element[3]
        end
      end
      qs:play_tutorial_voice(voice)
      qs:set("Ex", "-3")
      q:cancel_quest_timers("QT")
      q:start_quest_timer("QT", 30000, nil, player, false)
    elseif Ex == -3 then
      qs:play_tutorial_voice("tutorial_voice_002")
      qs:set("Ex", "0")
    elseif Ex == -4 then
      qs:play_tutorial_voice("tutorial_voice_008")
      qs:set("Ex", "-5")
    end
  elseif J.starts_with(event, "TE") then
    q:cancel_quest_timers("TE")
    if not J.equals_ignore_case(event, "TE") then
      local sw2 = J.parse_int(J.substring(event, 2))
      if sw2 == 0 then
        qs:close_tutorial_html()
      elseif sw2 == 1 then
        qs:close_tutorial_html()
        qs:play_tutorial_voice("tutorial_voice_006")
        qs:show_question_mark(1)
        q:play_sound(player, "ItemSound.quest_tutorial")
        q:start_quest_timer("QT", 30000, nil, player, false)
        qs:set("Ex", "-4")
      elseif sw2 == 2 then
        qs:play_tutorial_voice("tutorial_voice_003")
        html_ = "tutorial_02.htm"
        qs:on_tutorial_client_event(1)
        qs:set("Ex", "-5")
      elseif sw2 == 3 then
        html_ = "tutorial_03.htm"
        qs:on_tutorial_client_event(2)
      elseif sw2 == 5 then
        html_ = "tutorial_05.htm"
        qs:on_tutorial_client_event(8)
      elseif sw2 == 7 then
        html_ = "tutorial_100.htm"
        qs:on_tutorial_client_event(0)
      elseif sw2 == 8 then
        html_ = "tutorial_101.htm"
        qs:on_tutorial_client_event(0)
      elseif sw2 == 10 then
        html_ = "tutorial_103.htm"
        qs:on_tutorial_client_event(0)
      elseif sw2 == 12 then
        qs:close_tutorial_html()
      elseif sw2 == 23 then
        local entry = TCLb[classId]
        if entry ~= nil then
          html_ = entry
        end
      elseif sw2 == 24 then
        local entry = TCLc[classId]
        if entry ~= nil then
          html_ = entry
        end
      elseif sw2 == 25 then
        html_ = "tutorial_22cc.htm"
      elseif sw2 == 26 then
        local entry = TCLa[classId]
        if entry ~= nil then
          html_ = entry
        end
      elseif sw2 == 27 then
        html_ = "tutorial_29.htm"
      elseif sw2 == 28 then
        html_ = "tutorial_28.htm"
      end
    end
  elseif J.starts_with(event, "CE") then
    local event_id = J.parse_int(J.substring(event, 2))
    if event_id == 1 and player:level() < 6 then
      qs:play_tutorial_voice("tutorial_voice_004")
      html_ = "tutorial_03.htm"
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:on_tutorial_client_event(2)
    elseif event_id == 2 and player:level() < 6 then
      qs:play_tutorial_voice("tutorial_voice_005")
      html_ = "tutorial_05.htm"
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:on_tutorial_client_event(8)
    elseif event_id == 8 and player:level() < 6 then
      local x = 0
      local y = 0
      local z = 0
      for _, element in ipairs(CEEa) do
        if classId == J.parse_int(element[1]) then
          html_ = element[2]
          x = J.parse_int(element[3])
          y = J.parse_int(element[4])
          z = J.parse_int(element[5])
        end
      end
      if x ~= 0 then
        q:play_sound(player, "ItemSound.quest_tutorial")
        q:add_radar(player, x, y, z)
        qs:play_tutorial_voice("tutorial_voice_007")
        qs:set("ucMemo", "1")
        qs:set("Ex", "-5")
      end
    elseif event_id == 30 and player:level() < 10 and qs:get_int("Die") == 0 then
      qs:play_tutorial_voice("tutorial_voice_016")
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:set("Die", "1")
      qs:show_question_mark(8)
      qs:on_tutorial_client_event(0)
    elseif event_id == 800000 and player:level() < 6 and qs:get_int("sit") == 0 then
      qs:play_tutorial_voice("tutorial_voice_018")
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:set("sit", "1")
      qs:on_tutorial_client_event(0)
      html_ = "tutorial_21z.htm"
    elseif event_id == 40 then
      local sw3 = player:level()
      if sw3 == 5 then
        if qs:get_int("lvl") < 5 and not player:is_mage_class() or classId == 49 then
          qs:play_tutorial_voice("tutorial_voice_014")
          qs:show_question_mark(9)
          q:play_sound(player, "ItemSound.quest_tutorial")
          qs:set("lvl", "5")
        end
      elseif sw3 == 6 then
        if qs:get_int("lvl") < 6 and ClassId.level(player:class_id()) == 0 then
          qs:play_tutorial_voice("tutorial_voice_020")
          q:play_sound(player, "ItemSound.quest_tutorial")
          qs:show_question_mark(24)
          qs:set("lvl", "6")
        end
      elseif sw3 == 7 then
        if qs:get_int("lvl") < 7 and player:is_mage_class() and classId ~= 49 and ClassId.level(player:class_id()) == 0 then
          qs:play_tutorial_voice("tutorial_voice_019")
          q:play_sound(player, "ItemSound.quest_tutorial")
          qs:set("lvl", "7")
          qs:show_question_mark(11)
        end
      elseif sw3 == 15 then
        if qs:get_int("lvl") < 15 then
          q:play_sound(player, "ItemSound.quest_tutorial")
          qs:set("lvl", "15")
          qs:show_question_mark(33)
        end
      elseif sw3 == 19 then
        if qs:get_int("lvl") < 19 and ClassId.level(player:class_id()) == 0 then
          if classId == 0 or classId == 10 or classId == 18 or classId == 25 or classId == 31 or classId == 38 or classId == 44 or classId == 49 or classId == 53 then
            q:play_sound(player, "ItemSound.quest_tutorial")
            qs:set("lvl", "19")
            qs:show_question_mark(35)
          end
        end
      elseif sw3 == 35 then
        if qs:get_int("lvl") < 35 and ClassId.level(player:class_id()) == 1 then
          if classId == 1 or classId == 4 or classId == 7 or classId == 11 or classId == 15 or classId == 19 or classId == 22 or classId == 26 or classId == 29 or classId == 32 or classId == 35 or classId == 39 or classId == 42 or classId == 45 or classId == 47 or classId == 50 or classId == 54 or classId == 56 then
            q:play_sound(player, "ItemSound.quest_tutorial")
            qs:set("lvl", "35")
            qs:show_question_mark(34)
          end
        end
      end
    elseif event_id == 45 and player:level() < 10 and qs:get_int("HP") == 0 then
      qs:play_tutorial_voice("tutorial_voice_017")
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:set("HP", "1")
      qs:show_question_mark(10)
      qs:on_tutorial_client_event(800000)
    elseif event_id == 57 and player:level() < 6 and qs:get_int("Adena") == 0 then
      qs:play_tutorial_voice("tutorial_voice_012")
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:set("Adena", "1")
      qs:show_question_mark(23)
    elseif event_id == 6353 and player:level() < 6 and qs:get_int("Gemstone") == 0 then
      qs:play_tutorial_voice("tutorial_voice_013")
      q:play_sound(player, "ItemSound.quest_tutorial")
      qs:set("Gemstone", "1")
      qs:show_question_mark(5)
    elseif event_id == 1048576 and player:level() < 6 then
      qs:show_question_mark(5)
      qs:play_tutorial_voice("tutorial_voice_013")
      q:play_sound(player, "ItemSound.quest_tutorial")
    end
  elseif J.starts_with(event, "QM") then
    local x = 0
    local y = 0
    local z = 0
    local sw4 = J.parse_int(J.substring(event, 2))
    if sw4 == 1 then
      qs:play_tutorial_voice("tutorial_voice_007")
      qs:set("Ex", "-5")
      for _, element in ipairs(CEEa) do
        if classId == J.parse_int(element[1]) then
          html_ = element[2]
          x = J.parse_int(element[3])
          y = J.parse_int(element[4])
          z = J.parse_int(element[5])
        end
      end
      q:add_radar(player, x, y, z)
    elseif sw4 == 3 then
      html_ = "tutorial_09.htm"
      qs:on_tutorial_client_event(1048576)
    elseif sw4 == 5 then
      for _, element in ipairs(CEEa) do
        if classId == J.parse_int(element[1]) then
          -- html = element[1];
          x = J.parse_int(element[3])
          y = J.parse_int(element[4])
          z = J.parse_int(element[5])
        end
      end
      q:add_radar(player, x, y, z)
      html_ = "tutorial_11.htm"
    elseif sw4 == 7 then
      html_ = "tutorial_15.htm"
      qs:set("ucMemo", "3")
    elseif sw4 == 8 then
      html_ = "tutorial_18.htm"
    elseif sw4 == 9 then
      for _, element in ipairs(QMCa) do
        if classId == J.parse_int(element[1]) then
          html_ = element[2]
          x = J.parse_int(element[3])
          y = J.parse_int(element[4])
          z = J.parse_int(element[5])
        end
      end
      if x ~= 0 then
        q:add_radar(player, x, y, z)
      end
    elseif sw4 == 10 then
      html_ = "tutorial_19.htm"
    elseif sw4 == 11 then
      for _, element in ipairs(QMCa) do
        if classId == J.parse_int(element[1]) then
          html_ = element[2]
          x = J.parse_int(element[3])
          y = J.parse_int(element[4])
          z = J.parse_int(element[5])
        end
      end
      if x ~= 0 then
        q:add_radar(player, x, y, z)
      end
    elseif sw4 == 12 then
      html_ = "tutorial_15.htm"
      qs:set("ucMemo", "4")
    elseif sw4 == 17 then
      html_ = "tutorial_30.htm"
    elseif sw4 == 23 then
      html_ = "tutorial_24.htm"
    elseif sw4 == 24 then
      local entry = QMCb[classId]
      if entry ~= nil then
        html_ = entry
      end
    elseif sw4 == 26 then
      html_ = player:is_mage_class() and classId ~= 49 and "tutorial_newbie004b.htm" or "tutorial_newbie004a.htm"
    elseif sw4 == 33 then
      html_ = "tutorial_27.htm"
    elseif sw4 == 34 then
      html_ = "tutorial_28.htm"
    elseif sw4 == 35 then
      local entry = QMCc[classId]
      if entry ~= nil then
        html_ = entry
      end
    end
  end
  if html_ == "" then
    return nil
  end
  qs:show_tutorial_html(html_)
  return nil
end)
