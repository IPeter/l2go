-- Translated by questport from quests/Q00422_RepentYourSins (Mobius CT_0 Interlude); owned by L2Go from here on.
--
-- Needs pets (the Sin Eater): player:summon(), PK counts
-- (player:pk_kills/set_pk_kills/update_user_info) and the manacles' level
-- (player:get_item, item:enchant_level) are not in the API yet; the quest is
-- not offered until then.
local J = require("lib.java")

local q = quest.new(422, "Q00422_RepentYourSins", "Repent Your Sins")
q:requires("pets")

-- NPCs
local BLACK_JUDGE = 30981
local KATARI = 30668
local PIOTUR = 30597
local CASIAN = 30612
local JOAN = 30718
local PUSHKIN = 30300
-- Items
local RATMAN_SCAVENGER_SKULL = 4326
local TUREK_WAR_HOUND_TAIL = 4327
local TYRANT_KINGPIN_HEART = 4328
local TRISALIM_TARANTULA_VENOM_SAC = 4329
local QITEM_PENITENT_MANACLES = 4330
local MANUAL_OF_MANACLES = 4331
local PENITENT_MANACLES = 4425
local LEFT_PENITENT_MANACLES = 4426
local SILVER_NUGGET = 1873
local ADAMANTINE_NUGGET = 1877
local BLACKSMITH_FRAME = 1892
local COKES = 1879
local STEEL = 1880

local function findSinEaterLvl(player)
  return player:get_item(PENITENT_MANACLES):enchant_level()
end

q:register_quest_items(RATMAN_SCAVENGER_SKULL, TUREK_WAR_HOUND_TAIL, TYRANT_KINGPIN_HEART, TRISALIM_TARANTULA_VENOM_SAC, MANUAL_OF_MANACLES, PENITENT_MANACLES, QITEM_PENITENT_MANACLES)
q:add_start_npc(BLACK_JUDGE)
q:add_talk_id(BLACK_JUDGE, KATARI, PIOTUR, CASIAN, JOAN, PUSHKIN)
q:add_kill_id(20039, 20494, 20193, 20561)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "Start" then
    st:start_quest()
    if player:level() <= 20 then
      htmltext = "30981-03.htm"
      st:set_cond(2)
    elseif player:level() >= 20 and player:level() <= 30 then
      htmltext = "30981-04.htm"
      st:set_cond(3)
    elseif player:level() >= 30 and player:level() <= 40 then
      htmltext = "30981-05.htm"
      st:set_cond(4)
    else
      htmltext = "30981-06.htm"
      st:set_cond(5)
    end
  elseif event == "30981-11.htm" then
    if not q:has_quest_items(player, PENITENT_MANACLES) then
      local cond = st:get_cond()
      -- Case you return back the qitem to Black Judge. She rewards you with the pet item.
      if cond == 15 then
        st:set_cond(16)
        st:set("level", tostring(player:level()))
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, QITEM_PENITENT_MANACLES, -1)
        q:give_items(player, PENITENT_MANACLES, 1)
      elseif cond == 16 then
        st:set("level", tostring(player:level()))
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, LEFT_PENITENT_MANACLES, -1)
        q:give_items(player, PENITENT_MANACLES, 1)
      end
    end
  elseif event == "30981-19.htm" then
    if q:has_quest_items(player, LEFT_PENITENT_MANACLES) then
      st:set_state(State.STARTED)
      st:set_cond(16, true)
    end
  elseif event == "Pk" then
    local pet = player:summon()
    -- If Sin Eater is currently summoned, show a warning.
    if pet ~= nil and pet:id() == 12564 then
      htmltext = "30981-16.htm"
    elseif findSinEaterLvl(player) > st:get_int("level") then
      q:take_items(player, PENITENT_MANACLES, 1)
      q:give_items(player, LEFT_PENITENT_MANACLES, 1)
      local removePkAmount = rnd(10) + 1
      -- Player's PKs are lower than random amount ; finish the quest.
      if player:pk_kills() <= removePkAmount then
        htmltext = "30981-15.htm"
        st:exit_quest(true, true)
        player:set_pk_kills(0)
        player:update_user_info()
      else
        htmltext = "30981-14.htm"
        st:set("level", tostring(player:level()))
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        player:set_pk_kills(player:pk_kills() - removePkAmount)
        player:update_user_info()
      end
    end
  elseif event == "Quit" then
    htmltext = "30981-20.htm"
    q:take_items(player, RATMAN_SCAVENGER_SKULL, -1)
    q:take_items(player, TUREK_WAR_HOUND_TAIL, -1)
    q:take_items(player, TYRANT_KINGPIN_HEART, -1)
    q:take_items(player, TRISALIM_TARANTULA_VENOM_SAC, -1)
    q:take_items(player, MANUAL_OF_MANACLES, -1)
    q:take_items(player, PENITENT_MANACLES, -1)
    q:take_items(player, QITEM_PENITENT_MANACLES, -1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_already_completed_msg(player)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:pk_kills() >= 1 then
      htmltext = q:has_quest_items(player, LEFT_PENITENT_MANACLES) and "30981-18.htm" or "30981-02.htm"
    else
      htmltext = "30981-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BLACK_JUDGE then
      if cond <= 9 then
        htmltext = "30981-07.htm"
      elseif cond > 9 and cond < 14 then
        htmltext = "30981-08.htm"
        st:set_cond(14, true)
        q:give_items(player, MANUAL_OF_MANACLES, 1)
      elseif cond == 14 then
        htmltext = "30981-09.htm"
      elseif cond == 15 then
        htmltext = "30981-10.htm"
      elseif cond == 16 then
        if q:has_quest_items(player, PENITENT_MANACLES) then
          htmltext = findSinEaterLvl(player) > st:get_int("level") and "30981-13.htm" or "30981-12.htm"
        else
          htmltext = "30981-18.htm"
        end
      end
    elseif sw2 == KATARI then
      if cond == 2 then
        htmltext = "30668-01.htm"
        st:set_cond(6, true)
      elseif cond == 6 then
        if q:get_quest_items_count(player, RATMAN_SCAVENGER_SKULL) < 10 then
          htmltext = "30668-02.htm"
        else
          htmltext = "30668-03.htm"
          st:set_cond(10, true)
          q:take_items(player, RATMAN_SCAVENGER_SKULL, -1)
        end
      elseif cond == 10 then
        htmltext = "30668-04.htm"
      end
    elseif sw2 == PIOTUR then
      if cond == 3 then
        htmltext = "30597-01.htm"
        st:set_cond(7, true)
      elseif cond == 7 then
        if q:get_quest_items_count(player, TUREK_WAR_HOUND_TAIL) < 10 then
          htmltext = "30597-02.htm"
        else
          htmltext = "30597-03.htm"
          st:set_cond(11, true)
          q:take_items(player, TUREK_WAR_HOUND_TAIL, -1)
        end
      elseif cond == 11 then
        htmltext = "30597-04.htm"
      end
    elseif sw2 == CASIAN then
      if cond == 4 then
        htmltext = "30612-01.htm"
        st:set_cond(8, true)
      elseif cond == 8 then
        if not q:has_quest_items(player, TYRANT_KINGPIN_HEART) then
          htmltext = "30612-02.htm"
        else
          htmltext = "30612-03.htm"
          st:set_cond(12, true)
          q:take_items(player, TYRANT_KINGPIN_HEART, -1)
        end
      elseif cond == 12 then
        htmltext = "30612-04.htm"
      end
    elseif sw2 == JOAN then
      if cond == 5 then
        htmltext = "30718-01.htm"
        st:set_cond(9, true)
      elseif cond == 9 then
        if q:get_quest_items_count(player, TRISALIM_TARANTULA_VENOM_SAC) < 3 then
          htmltext = "30718-02.htm"
        else
          htmltext = "30718-03.htm"
          st:set_cond(13, true)
          q:take_items(player, TRISALIM_TARANTULA_VENOM_SAC, -1)
        end
      elseif cond == 13 then
        htmltext = "30718-04.htm"
      end
    elseif sw2 == PUSHKIN then
      if cond == 14 and q:get_quest_items_count(player, MANUAL_OF_MANACLES) == 1 then
        if q:get_quest_items_count(player, SILVER_NUGGET) < 10 or q:get_quest_items_count(player, STEEL) < 5 or q:get_quest_items_count(player, ADAMANTINE_NUGGET) < 2 or q:get_quest_items_count(player, COKES) < 10 or q:get_quest_items_count(player, BLACKSMITH_FRAME) < 1 then
          htmltext = "30300-02.htm"
        else
          htmltext = "30300-01.htm"
          st:set_cond(15, true)
          q:take_items(player, MANUAL_OF_MANACLES, 1)
          q:take_items(player, SILVER_NUGGET, 10)
          q:take_items(player, ADAMANTINE_NUGGET, 2)
          q:take_items(player, COKES, 10)
          q:take_items(player, STEEL, 5)
          q:take_items(player, BLACKSMITH_FRAME, 1)
          q:give_items(player, QITEM_PENITENT_MANACLES, 1)
        end
      elseif q:has_at_least_one_quest_item(player, QITEM_PENITENT_MANACLES, PENITENT_MANACLES, LEFT_PENITENT_MANACLES) then
        htmltext = "30300-03.htm"
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
  if sw == 20039 then
    if st:is_cond(6) and q:get_quest_items_count(player, RATMAN_SCAVENGER_SKULL) < 10 then
      q:give_items(player, RATMAN_SCAVENGER_SKULL, 1)
      if q:get_quest_items_count(player, RATMAN_SCAVENGER_SKULL) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20494 then
    if st:is_cond(7) and q:get_quest_items_count(player, TUREK_WAR_HOUND_TAIL) < 10 then
      q:give_items(player, TUREK_WAR_HOUND_TAIL, 1)
      if q:get_quest_items_count(player, TUREK_WAR_HOUND_TAIL) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20193 then
    if st:is_cond(8) and not q:has_quest_items(player, TYRANT_KINGPIN_HEART) then
      q:give_items(player, TYRANT_KINGPIN_HEART, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == 20561 then
    if st:is_cond(9) and q:get_quest_items_count(player, TRISALIM_TARANTULA_VENOM_SAC) < 3 then
      q:give_items(player, TRISALIM_TARANTULA_VENOM_SAC, 1)
      if q:get_quest_items_count(player, TRISALIM_TARANTULA_VENOM_SAC) < 3 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)
