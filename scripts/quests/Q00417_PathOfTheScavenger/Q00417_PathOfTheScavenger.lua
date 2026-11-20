-- Translated by questport from quests/Q00417_PathOfTheScavenger (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(417, "Q00417_PathOfTheScavenger", "Path to become a Scavenger")

-- NPCs
local RAUT = 30316
local SHARI = 30517
local MION = 30519
local PIPPI = 30524
local BRONK = 30525
local ZIMENF = 30538
local TOMA = 30556
local TORAI = 30557
local YASHENI = 31958
-- Monsters
local HUNTER_TARANTULA = 20403
local PLUNDER_TARANTULA = 20508
local HUNTER_BEAR = 20777
local HONEY_BEAR = 27058
-- Items
local RING_OF_RAVEN = 1642
local PIPPI_LETTER = 1643
local RAUT_TELEPORT_SCROLL = 1644
local SUCCUBUS_UNDIES = 1645
local MION_LETTER = 1646
local BRONK_INGOT = 1647
local SHARI_AXE = 1648
local ZIMENF_POTION = 1649
local BRONK_PAY = 1650
local SHARI_PAY = 1651
local ZIMENF_PAY = 1652
local BEAR_PICTURE = 1653
local TARANTULA_PICTURE = 1654
local HONEY_JAR = 1655
local BEAD = 1656
local BEAD_PARCEL_1 = 1657
local BEAD_PARCEL_2 = 8543

q:register_quest_items(PIPPI_LETTER, RAUT_TELEPORT_SCROLL, SUCCUBUS_UNDIES, MION_LETTER, BRONK_INGOT, SHARI_AXE, ZIMENF_POTION, BRONK_PAY, SHARI_PAY, ZIMENF_PAY, BEAR_PICTURE, TARANTULA_PICTURE, HONEY_JAR, BEAD, BEAD_PARCEL_1, BEAD_PARCEL_2)
q:add_start_npc(PIPPI)
q:add_talk_id(RAUT, SHARI, MION, PIPPI, BRONK, ZIMENF, TOMA, TORAI, YASHENI)
q:add_kill_id(HUNTER_TARANTULA, PLUNDER_TARANTULA, HUNTER_BEAR, HONEY_BEAR)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30524-05.htm" then
    if player:class_id() ~= ClassId.DWARVEN_FIGHTER then
      htmltext = player:class_id() == ClassId.SCAVENGER and "30524-02a.htm" or "30524-08.htm"
    elseif player:level() < 19 then
      htmltext = "30524-02.htm"
    elseif q:has_quest_items(player, RING_OF_RAVEN) then
      htmltext = "30524-04.htm"
    else
      st:start_quest()
      q:give_items(player, PIPPI_LETTER, 1)
    end
  elseif event == "30519_1" then
    local random = rnd(3)
    htmltext = "30519-0" .. (random + 2) .. ".htm"
    st:set_cond(2, true)
    q:take_items(player, PIPPI_LETTER, -1)
    q:give_items(player, ZIMENF_POTION - random, 1)
  elseif event == "30519_2" then
    local random = rnd(3)
    htmltext = "30519-0" .. (random + 2) .. ".htm"
    q:take_items(player, BRONK_PAY, -1)
    q:take_items(player, SHARI_PAY, -1)
    q:take_items(player, ZIMENF_PAY, -1)
    q:give_items(player, ZIMENF_POTION - random, 1)
  elseif event == "30519-07.htm" then
    st:set("id", tostring(st:get_int("id") + 1))
  elseif event == "30519-09.htm" then
    local id = st:get_int("id")
    if J.idiv(id, 10) < 2 then
      htmltext = "30519-07.htm"
      st:set("id", tostring(id + 1))
    elseif J.idiv(id, 10) == 2 then
      st:set("id", tostring(id + 1))
    elseif J.idiv(id, 10) >= 3 then
      htmltext = "30519-10.htm"
      st:set_cond(4, true)
      q:take_items(player, SHARI_AXE, -1)
      q:take_items(player, ZIMENF_POTION, -1)
      q:take_items(player, BRONK_INGOT, -1)
      q:give_items(player, MION_LETTER, 1)
    end
  elseif event == "30519-11.htm" then
    if rnd.bool() then
      htmltext = "30519-06.htm"
    end
  elseif event == "30556-05b.htm" then
    st:set_cond(9, true)
    q:take_items(player, BEAD, -1)
    q:take_items(player, TARANTULA_PICTURE, 1)
    q:give_items(player, BEAD_PARCEL_1, 1)
  elseif event == "30556-06b.htm" then
    st:set_cond(12, true)
    q:take_items(player, BEAD, -1)
    q:take_items(player, TARANTULA_PICTURE, 1)
    q:give_items(player, BEAD_PARCEL_2, 1)
  elseif event == "30316-02.htm" or event == "30316-03.htm" then
    st:set_cond(10, true)
    q:take_items(player, BEAD_PARCEL_1, 1)
    q:give_items(player, RAUT_TELEPORT_SCROLL, 1)
  elseif event == "30557-03.htm" then
    st:set_cond(11, true)
    q:take_items(player, RAUT_TELEPORT_SCROLL, 1)
    q:give_items(player, SUCCUBUS_UNDIES, 1)
  elseif event == "31958-02.htm" then
    q:take_items(player, BEAD_PARCEL_2, 1)
    q:give_items(player, RING_OF_RAVEN, 1)
    q:add_exp_and_sp(player, 3200, 7080)
    player:social_action(3)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30524-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == PIPPI then
      if cond == 1 then
        htmltext = "30524-06.htm"
      elseif cond > 1 then
        htmltext = "30524-07.htm"
      end
    elseif sw2 == MION then
      if q:has_quest_items(player, PIPPI_LETTER) then
        htmltext = "30519-01.htm"
      elseif q:has_at_least_one_quest_item(player, BRONK_INGOT, SHARI_AXE, ZIMENF_POTION) then
        local id = st:get_int("id")
        if J.idiv(id, 10) == 0 then
          htmltext = "30519-05.htm"
        else
          htmltext = "30519-08.htm"
        end
      elseif q:has_at_least_one_quest_item(player, BRONK_PAY, SHARI_PAY, ZIMENF_PAY) then
        local id = st:get_int("id")
        if id < 50 then
          htmltext = "30519-12.htm"
        else
          htmltext = "30519-15.htm"
          st:set_cond(4, true)
          q:take_items(player, BRONK_PAY, -1)
          q:take_items(player, SHARI_PAY, -1)
          q:take_items(player, ZIMENF_PAY, -1)
          q:give_items(player, MION_LETTER, 1)
        end
      elseif cond == 4 then
        htmltext = "30519-13.htm"
      elseif cond > 4 then
        htmltext = "30519-14.htm"
      end
    elseif sw2 == SHARI then
      if q:has_quest_items(player, SHARI_AXE) then
        local id = st:get_int("id")
        if id < 20 then
          htmltext = "30517-01.htm"
        else
          htmltext = "30517-02.htm"
          st:set_cond(3, true)
        end
        st:set("id", tostring(id + 10))
        q:take_items(player, SHARI_AXE, 1)
        q:give_items(player, SHARI_PAY, 1)
      elseif q:has_quest_items(player, SHARI_PAY) then
        htmltext = "30517-03.htm"
      end
    elseif sw2 == BRONK then
      if q:has_quest_items(player, BRONK_INGOT) then
        local id = st:get_int("id")
        if id < 20 then
          htmltext = "30525-01.htm"
        else
          htmltext = "30525-02.htm"
          st:set_cond(3, true)
        end
        st:set("id", tostring(id + 10))
        q:take_items(player, BRONK_INGOT, 1)
        q:give_items(player, BRONK_PAY, 1)
      elseif q:has_quest_items(player, BRONK_PAY) then
        htmltext = "30525-03.htm"
      end
    elseif sw2 == ZIMENF then
      if q:has_quest_items(player, ZIMENF_POTION) then
        local id = st:get_int("id")
        if id < 20 then
          htmltext = "30538-01.htm"
        else
          htmltext = "30538-02.htm"
          st:set_cond(3, true)
        end
        st:set("id", tostring(id + 10))
        q:take_items(player, ZIMENF_POTION, 1)
        q:give_items(player, ZIMENF_PAY, 1)
      elseif q:has_quest_items(player, ZIMENF_PAY) then
        htmltext = "30538-03.htm"
      end
    elseif sw2 == TOMA then
      if cond == 4 then
        htmltext = "30556-01.htm"
        st:set_cond(5, true)
        q:take_items(player, MION_LETTER, 1)
        q:give_items(player, BEAR_PICTURE, 1)
      elseif cond == 5 then
        htmltext = "30556-02.htm"
      elseif cond == 6 then
        htmltext = "30556-03.htm"
        st:set_cond(7, true)
        q:take_items(player, HONEY_JAR, -1)
        q:take_items(player, BEAR_PICTURE, 1)
        q:give_items(player, TARANTULA_PICTURE, 1)
      elseif cond == 7 then
        htmltext = "30556-04.htm"
      elseif cond == 8 then
        htmltext = "30556-05a.htm"
      elseif cond == 9 then
        htmltext = "30556-06a.htm"
      elseif cond == 10 or cond == 11 then
        htmltext = "30556-07.htm"
      elseif cond == 12 then
        htmltext = "30556-06c.htm"
      end
    elseif sw2 == RAUT then
      if cond == 9 then
        htmltext = "30316-01.htm"
      elseif cond == 10 then
        htmltext = "30316-04.htm"
      elseif cond == 11 then
        htmltext = "30316-05.htm"
        q:take_items(player, SUCCUBUS_UNDIES, 1)
        q:give_items(player, RING_OF_RAVEN, 1)
        q:add_exp_and_sp(player, 3200, 7080)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == TORAI then
      if cond == 10 then
        htmltext = "30557-01.htm"
      end
    elseif sw2 == YASHENI then
      if cond == 12 then
        htmltext = "31958-01.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == HUNTER_BEAR then
    if st:is_cond(5) then
      local step = st:get_int("step")
      if step > 20 then
        if (step - 20) * 10 >= rnd(100) then
          q:add_spawn(HONEY_BEAR, npc, false, 300000)
          st:unset("step")
        else
          st:set("step", tostring(step + 1))
        end
      else
        st:set("step", tostring(step + 1))
      end
    end
  -- L2Go: Mobius counts these kills only when the killer spoiled the monster
  -- (npc.getSpoilerObjectId() == player.getObjectId()). The game keeps the
  -- spoiler, but the script API cannot read it yet, so the check is left out
  -- for now and any kill counts; put it back once npc:spoiler_object_id()
  -- exists.
  elseif sw == HONEY_BEAR then
    if st:is_cond(5) then
      if q:get_quest_items_count(player, HONEY_JAR) < 5 then
        q:give_items(player, HONEY_JAR, 1)
        if q:get_quest_items_count(player, HONEY_JAR) >= 5 then
          st:set_cond(6, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == HUNTER_TARANTULA or sw == PLUNDER_TARANTULA then
    if st:is_cond(7) and rnd(100) < (npc:id() == HUNTER_TARANTULA and 33 or 60) then
      if q:get_quest_items_count(player, BEAD) < 20 then
        q:give_items(player, BEAD, 1)
        if q:get_quest_items_count(player, BEAD) >= 20 then
          st:set_cond(8, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
