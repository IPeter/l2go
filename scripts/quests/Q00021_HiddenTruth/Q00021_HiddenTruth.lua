-- Translated by questport from quests/Q00021_HiddenTruth (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(21, "Q00021_HiddenTruth", "Hidden Truth")

-- NPCs
local MYSTERIOUS_WIZARD = 31522
local TOMBSTONE = 31523
local VON_HELLMAN_DUKE = 31524
local VON_HELLMAN_PAGE = 31525
local BROKEN_BOOKSHELF = 31526
local AGRIPEL = 31348
local DOMINIC = 31350
local BENEDICT = 31349
local INNOCENTIN = 31328
-- Items
local CROSS_OF_EINHASAD = 7140
local CROSS_OF_EINHASAD_NEXT_QUEST = 7141
local PAGE_LOCS = {
  {x = 51992, y = -54424, z = -3160},
  {x = 52328, y = -53400, z = -3160},
  {x = 51928, y = -51656, z = -3096},
}
local _duke
local _page

local function spawnTheDuke(player)
  if _duke == nil then
    _duke = J.oid(q:add_spawn(VON_HELLMAN_DUKE, {x = 51432, y = -54570, z = -3136, heading = 0}, false, 0))
    J.obj(_duke):broadcast_say(ChatType.GENERAL, "Who awoke me?")
    q:start_quest_timer("dukeDespawn", 300000, J.obj(_duke), player, false)
  end
end

local function spawnThePage(player)
  if _page == nil then
    _page = J.oid(q:add_spawn(VON_HELLMAN_PAGE, {x = 51608, y = -54520, z = -3168, heading = 0}, false, 0))
    J.obj(_page):broadcast_say(ChatType.GENERAL, "My master has instructed me to be your guide, " .. player:name() .. ".")
    q:start_quest_timer("1", 4000, J.obj(_page), player, false)
    q:start_quest_timer("pageDespawn", 90000, J.obj(_page), player, false)
  end
end

q:register_quest_items(CROSS_OF_EINHASAD)
q:add_start_npc(MYSTERIOUS_WIZARD)
q:add_talk_id(MYSTERIOUS_WIZARD, TOMBSTONE, VON_HELLMAN_DUKE, VON_HELLMAN_PAGE, BROKEN_BOOKSHELF, AGRIPEL, DOMINIC, BENEDICT, INNOCENTIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31522-02.htm" then
    st:start_quest()
  elseif event == "31523-03.htm" then
    st:set_cond(2, true)
    spawnTheDuke(player)
  elseif event == "31524-06.htm" then
    st:set_cond(3, true)
    spawnThePage(player)
  elseif event == "31526-08.htm" then
    st:set_cond(5, true)
  elseif event == "31526-14.htm" then
    st:set_cond(6, true)
    q:give_items(player, CROSS_OF_EINHASAD, 1)
  elseif event == "1" then
    J.obj(_page):move_to(PAGE_LOCS[1])
    J.obj(_page):broadcast_say(ChatType.GENERAL, "Follow me...")
    q:start_quest_timer("2", 5000, J.obj(_page), player, false)
    return nil
  elseif event == "2" then
    J.obj(_page):move_to(PAGE_LOCS[2])
    q:start_quest_timer("3", 12000, J.obj(_page), player, false)
    return nil
  elseif event == "3" then
    J.obj(_page):move_to(PAGE_LOCS[3])
    q:start_quest_timer("4", 18000, J.obj(_page), player, false)
    return nil
  elseif event == "4" then
    st:set("end_walk", "1")
    J.obj(_page):broadcast_say(ChatType.GENERAL, "Please check this bookcase, " .. player:name() .. ".")
    q:start_quest_timer("5", 47000, J.obj(_page), player, false)
    return nil
  elseif event == "5" then
    J.obj(_page):broadcast_say(ChatType.GENERAL, "I'm confused! Maybe it's time to go back.")
    return nil
  elseif event == "31328-05.htm" then
    if q:has_quest_items(player, CROSS_OF_EINHASAD) then
      q:take_items(player, CROSS_OF_EINHASAD, 1)
      q:give_items(player, CROSS_OF_EINHASAD_NEXT_QUEST, 1)
      st:exit_quest(false, true)
    end
  elseif event == "dukeDespawn" then
    J.obj(_duke):delete()
    _duke = nil
    return nil
  elseif event == "pageDespawn" then
    J.obj(_page):delete()
    _page = nil
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 63 and "31522-03.htm" or "31522-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MYSTERIOUS_WIZARD then
      htmltext = "31522-05.htm"
    elseif sw2 == TOMBSTONE then
      if cond == 1 then
        htmltext = "31523-01.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "31523-04.htm"
        spawnTheDuke(player)
      elseif cond > 3 then
        htmltext = "31523-04.htm"
      end
    elseif sw2 == VON_HELLMAN_DUKE then
      if cond == 2 then
        htmltext = "31524-01.htm"
      elseif cond == 3 then
        htmltext = "31524-07.htm"
        spawnThePage(player)
      elseif cond > 3 then
        htmltext = "31524-07a.htm"
      end
    elseif sw2 == VON_HELLMAN_PAGE then
      if cond == 3 then
        if st:get_int("end_walk") == 1 then
          htmltext = "31525-02.htm"
          st:set_cond(4, true)
        else
          htmltext = "31525-01.htm"
        end
      elseif cond == 4 then
        htmltext = "31525-02.htm"
      end
    elseif sw2 == BROKEN_BOOKSHELF then
      if cond == 3 and st:get_int("end_walk") == 1 or cond == 4 then
        htmltext = "31526-01.htm"
        st:set_cond(5, true)
        if _page ~= nil then
          q:cancel_quest_timer("5", J.obj(_page), player)
          q:cancel_quest_timer("pageDespawn", J.obj(_page), player)
          J.obj(_page):delete()
          _page = nil
        end
        if _duke ~= nil then
          q:cancel_quest_timer("dukeDespawn", J.obj(_duke), player)
          J.obj(_duke):delete()
          _duke = nil
        end
      elseif cond == 5 then
        htmltext = "31526-10.htm"
      elseif cond > 5 then
        htmltext = "31526-15.htm"
      end
    elseif sw2 == AGRIPEL or sw2 == BENEDICT or sw2 == DOMINIC then
      if (cond == 6 or cond == 7) and q:has_quest_items(player, CROSS_OF_EINHASAD) then
        local npcId = npc:id()
        -- For cond 6, make checks until cond 7 is activated.
        if cond == 6 then
          local npcId1 = 0
          local npcId2 = 0
          if npcId == AGRIPEL then
            npcId1 = BENEDICT
            npcId2 = DOMINIC
          elseif npcId == BENEDICT then
            npcId1 = AGRIPEL
            npcId2 = DOMINIC
          elseif npcId == DOMINIC then
            npcId1 = AGRIPEL
            npcId2 = BENEDICT
          end
          if st:get_int(tostring(npcId1)) == 1 and st:get_int(tostring(npcId2)) == 1 then
            st:set_cond(7, true)
          else
            st:set(tostring(npcId), "1")
          end
        end
        htmltext = npcId .. "-01.htm"
      end
    elseif sw2 == INNOCENTIN then
      if cond == 7 and q:has_quest_items(player, CROSS_OF_EINHASAD) then
        htmltext = "31328-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    if npc:id() == INNOCENTIN then
      htmltext = "31328-06.htm"
    else
      htmltext = q:get_already_completed_msg(player)
    end
  end
  return htmltext
end)
