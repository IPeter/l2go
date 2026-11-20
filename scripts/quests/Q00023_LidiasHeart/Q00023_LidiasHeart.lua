-- Translated by questport from quests/Q00023_LidiasHeart (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(23, "Q00023_LidiasHeart", "Lidia's Heart")

-- NPCs
local INNOCENTIN = 31328
local BROKEN_BOOKSHELF = 31526
local GHOST_OF_VON_HELLMANN = 31524
local TOMBSTONE = 31523
local VIOLET = 31386
local BOX = 31530
-- NPC instance
local _ghost = nil
-- Items
local FOREST_OF_DEADMAN_MAP = 7063
local SILVER_KEY = 7149
local LIDIA_HAIRPIN = 7148
local LIDIA_DIARY = 7064
local SILVER_SPEAR = 7150

q:register_quest_items(FOREST_OF_DEADMAN_MAP, SILVER_KEY, LIDIA_DIARY, SILVER_SPEAR)
q:add_start_npc(INNOCENTIN)
q:add_talk_id(INNOCENTIN, BROKEN_BOOKSHELF, GHOST_OF_VON_HELLMANN, VIOLET, BOX, TOMBSTONE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31328-02.htm" then
    st:start_quest()
    q:give_items(player, FOREST_OF_DEADMAN_MAP, 1)
    q:give_items(player, SILVER_KEY, 1)
  elseif event == "31328-06.htm" then
    st:set_cond(2, true)
  elseif event == "31526-05.htm" then
    if not q:has_quest_items(player, LIDIA_HAIRPIN) then
      q:give_items(player, LIDIA_HAIRPIN, 1)
      if q:has_quest_items(player, LIDIA_DIARY) then
        st:set_cond(4, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif event == "31526-11.htm" then
    if not q:has_quest_items(player, LIDIA_DIARY) then
      q:give_items(player, LIDIA_DIARY, 1)
      if q:has_quest_items(player, LIDIA_HAIRPIN) then
        st:set_cond(4, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif event == "31328-11.htm" then
    if st:get_cond() < 5 then
      st:set_cond(5, true)
    end
  elseif event == "31328-19.htm" then
    st:set_cond(6, true)
  elseif event == "31524-04.htm" then
    st:set_cond(7, true)
    q:take_items(player, LIDIA_DIARY, 1)
  elseif event == "31523-02.htm" then
    if _ghost == nil then
      _ghost = J.oid(q:add_spawn(31524, {x = 51432, y = -54570, z = -3136, heading = 0}, false, 60000))
      J.obj(_ghost):broadcast_say(ChatType.GENERAL, "Who awoke me?")
      q:start_quest_timer("ghost_cleanup", 58000, nil, player, false)
    end
  elseif event == "31523-05.htm" then
    -- Don't launch twice the same task...
    if q:get_quest_timer("tomb_digger", nil, player) == nil then
      q:start_quest_timer("tomb_digger", 10000, nil, player, false)
    end
  elseif event == "tomb_digger" then
    htmltext = "31523-06.htm"
    st:set_cond(8, true)
    q:give_items(player, SILVER_KEY, 1)
  elseif event == "31530-02.htm" then
    st:set_cond(10, true)
    q:take_items(player, SILVER_KEY, 1)
    q:give_items(player, SILVER_SPEAR, 1)
  elseif event == "ghost_cleanup" then
    _ghost = nil
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local st2 = player:get_quest_state("Q00022_TragedyInVonHellmannForest")
    if st2 ~= nil and st2:is_completed() then
      if player:level() >= 64 then
        htmltext = "31328-01.htm"
      else
        htmltext = "31328-00a.htm"
      end
    else
      htmltext = "31328-00.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == INNOCENTIN then
      if cond == 1 then
        htmltext = "31328-03.htm"
      elseif cond == 2 then
        htmltext = "31328-07.htm"
      elseif cond == 4 then
        htmltext = "31328-08.htm"
      elseif cond == 5 then
        htmltext = "31328-10.htm"
      elseif cond > 5 then
        htmltext = "31328-21.htm"
      end
    elseif sw2 == BROKEN_BOOKSHELF then
      if cond == 2 then
        htmltext = "31526-00.htm"
        st:set_cond(3, true)
      elseif cond == 3 then
        if not q:has_quest_items(player, LIDIA_DIARY) then
          htmltext = not q:has_quest_items(player, LIDIA_HAIRPIN) and "31526-02.htm" or "31526-06.htm"
        elseif not q:has_quest_items(player, LIDIA_HAIRPIN) then
          htmltext = "31526-12.htm"
        end
      elseif cond > 3 then
        htmltext = "31526-13.htm"
      end
    elseif sw2 == GHOST_OF_VON_HELLMANN then
      if cond == 6 then
        htmltext = "31524-01.htm"
      elseif cond > 6 then
        htmltext = "31524-05.htm"
      end
    elseif sw2 == TOMBSTONE then
      if cond == 6 then
        htmltext = _ghost == nil and "31523-01.htm" or "31523-03.htm"
      elseif cond == 7 then
        htmltext = "31523-04.htm"
      elseif cond > 7 then
        htmltext = "31523-06.htm"
      end
    elseif sw2 == VIOLET then
      if cond == 8 then
        htmltext = "31386-01.htm"
        st:set_cond(9, true)
      elseif cond == 9 then
        htmltext = "31386-02.htm"
      elseif cond == 10 then
        if q:has_quest_items(player, SILVER_SPEAR) then
          htmltext = "31386-03.htm"
          q:take_items(player, SILVER_SPEAR, 1)
          q:give_adena(player, 100000, true)
          st:exit_quest(false, true)
        else
          htmltext = "31386-02.htm"
          st:set_cond(9)
        end
      end
    elseif sw2 == BOX then
      if cond == 9 then
        htmltext = "31530-01.htm"
      elseif cond == 10 then
        htmltext = "31530-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    if npc:id() == VIOLET then
      htmltext = "31386-04.htm"
    else
      htmltext = q:get_already_completed_msg(player)
    end
  end
  return htmltext
end)
