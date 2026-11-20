-- Translated by questport from quests/Q00123_TheLeaderAndTheFollower (Mobius CT_0 Interlude); owned by L2Go from here on.
-- The Leader And The Follower (123)
-- @author ivantotov
--
-- Needs the clan academy: player:apprentice(), player:sponsor() and
-- player:pledge_type() come with clans (M9). Until then the quest is not
-- offered (q:requires("clans")).
local J = require("lib.java")

local q = quest.new(123, "Q00123_TheLeaderAndTheFollower", "The Leader and the Follower")
q:requires("clans")

-- NPC
local HEAD_BLACKSMITH_NEWYEAR = 31961
-- Items
local CRYSTAL_D = 1458
local BRUIN_LIZARDMAN_BLOOD = 8549
local PICOT_ARANEIDS_LEG = 8550
-- Reward
local CLAN_OATH_HELM = 7850
local CLAN_OATH_ARMOR = 7851
local CLAN_OATH_GAUNTLETS_HEAVY_ARMOR = 7852
local CLAN_OATH_SABATON_HEAVY_ARMOR = 7853
local CLAN_OATH_BRIGANDINE = 7854
local CLAN_OATH_LEATHER_GLOVES_LIGHT_ARMOR = 7855
local CLAN_OATH_BOOTS_LIGHT_ARMOR = 7856
local CLAN_OATH_AKETON = 7857
local CLAN_OATH_PADDED_GLOVES_ROBE = 7858
local CLAN_OATH_SANDALS_ROBE = 7859
-- Quest Monster
local BRUIN_LIZARDMAN = 27321
local PICOT_ARANEID = 27322
-- Misc
local MIN_LEVEL = 19
local CRYSTAL_COUNT_1 = 922
local CRYSTAL_COUNT_2 = 771

q:add_start_npc(HEAD_BLACKSMITH_NEWYEAR)
q:add_talk_id(HEAD_BLACKSMITH_NEWYEAR)
q:add_kill_id(BRUIN_LIZARDMAN, PICOT_ARANEID)
q:register_quest_items(BRUIN_LIZARDMAN_BLOOD, PICOT_ARANEIDS_LEG)

q:on_event(function(event, npc, player)
  local htmltext = nil
  -- Manage Sponsor's quest events.
  if player:apprentice() > 0 then
    local apprentice = world.get_player(player:apprentice())
    if apprentice == nil then
      return htmltext
    end
    local q123 = apprentice:get_quest_state("Q00123_TheLeaderAndTheFollower")
    if event == "sponsor" then
      if not util.in_range(Config.ALT_PARTY_RANGE, npc, apprentice, true) then
        htmltext = "31961-09.html"
      else
        if q123 == nil or not q123:is_memo_state(2) and not q123:is_memo_state(3) then
          htmltext = "31961-14.html"
        elseif q123:is_memo_state(2) then
          htmltext = "31961-08.html"
        elseif q123:is_memo_state(3) then
          htmltext = "31961-12.html"
        end
      end
    elseif event == "31961-10.html" then
      if util.in_range(Config.ALT_PARTY_RANGE, npc, apprentice, true) and q123 ~= nil and q123:is_memo_state(2) then
        local sw = q123:get_memo_state_ex(1)
        if sw == 1 then
          if q:get_quest_items_count(player, CRYSTAL_D) >= CRYSTAL_COUNT_1 then
            q:take_items(player, CRYSTAL_D, CRYSTAL_COUNT_1)
            q123:set_memo_state(3)
            q123:set_cond(6, true)
            htmltext = event
          else
            htmltext = "31961-11.html"
          end
        elseif sw == 2 or sw == 3 then
          if q:get_quest_items_count(player, CRYSTAL_D) >= CRYSTAL_COUNT_2 then
            q:take_items(player, CRYSTAL_D, CRYSTAL_COUNT_2)
            q123:set_memo_state(3)
            q123:set_cond(6, true)
            htmltext = event
          else
            htmltext = "31961-11a.html"
          end
        end
      end
    end
    return htmltext
  end
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  if event == "31961-03.htm" then
    if qs:is_created() then
      qs:start_quest()
      qs:set_memo_state(1)
      htmltext = event
    end
  elseif event == "31961-05a.html" or event == "31961-05b.html" or event == "31961-05c.html" or event == "31961-05g.html" then
    htmltext = event
  elseif event == "31961-05d.html" then
    if qs:is_memo_state(1) and q:get_quest_items_count(player, BRUIN_LIZARDMAN_BLOOD) >= 10 then
      q:take_items(player, BRUIN_LIZARDMAN_BLOOD, -1)
      qs:set_memo_state(2)
      qs:set_memo_state_ex(1, 1)
      qs:set_cond(3, true)
      htmltext = event
    end
  elseif event == "31961-05e.html" then
    if qs:is_memo_state(1) and q:get_quest_items_count(player, BRUIN_LIZARDMAN_BLOOD) >= 10 then
      q:take_items(player, BRUIN_LIZARDMAN_BLOOD, -1)
      qs:set_memo_state(2)
      qs:set_memo_state_ex(1, 2)
      qs:set_cond(4, true)
      htmltext = event
    end
  elseif event == "31961-05f.html" then
    if qs:is_memo_state(1) and q:get_quest_items_count(player, BRUIN_LIZARDMAN_BLOOD) >= 10 then
      q:take_items(player, BRUIN_LIZARDMAN_BLOOD, -1)
      qs:set_memo_state(2)
      qs:set_memo_state_ex(1, 3)
      qs:set_cond(5, true)
      htmltext = event
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local q118 = player:get_quest_state("Q00118_ToLeadAndBeLed")
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    if q118 ~= nil and q118:is_started() then
      htmltext = "31961-02b.htm"
    elseif q118 ~= nil and q118:is_completed() then
      htmltext = "31961-02a.html"
    elseif player:level() >= MIN_LEVEL and player:pledge_type() == -1 and player:sponsor() > 0 then
      htmltext = "31961-01.htm"
    else
      htmltext = "31961-02.htm"
    end
  elseif sw == State.STARTED then
    if qs:is_memo_state(1) then
      if q:get_quest_items_count(player, BRUIN_LIZARDMAN_BLOOD) < 10 then
        htmltext = "31961-04.html"
      else
        htmltext = "31961-05.html"
      end
    elseif qs:is_memo_state(2) then
      if player:sponsor() == 0 then
        if qs:get_memo_state_ex(1) == 1 then
          htmltext = "31961-06a.html"
        elseif qs:get_memo_state_ex(1) == 2 then
          htmltext = "31961-06b.html"
        elseif qs:get_memo_state_ex(1) == 3 then
          htmltext = "31961-06c.html"
        end
      else
        local c0 = world.get_player(player:sponsor())
        if c0 ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, c0, true) then
          htmltext = "31961-07.html"
        else
          if qs:get_memo_state_ex(1) == 1 then
            htmltext = "31961-06.html"
          elseif qs:get_memo_state_ex(1) == 2 then
            htmltext = "31961-06d.html"
          elseif qs:get_memo_state_ex(1) == 3 then
            htmltext = "31961-06e.html"
          end
        end
      end
    elseif qs:is_memo_state(3) then
      qs:set_memo_state(4)
      qs:set_cond(7, true)
      htmltext = "31961-15.html"
    elseif qs:is_memo_state(4) then
      if q:get_quest_items_count(player, PICOT_ARANEIDS_LEG) < 8 then
        htmltext = "31961-16.html"
      else
        if qs:get_memo_state_ex(1) == 1 then
          q:give_items(player, CLAN_OATH_HELM, 1)
          q:give_items(player, CLAN_OATH_ARMOR, 1)
          q:give_items(player, CLAN_OATH_GAUNTLETS_HEAVY_ARMOR, 1)
          q:give_items(player, CLAN_OATH_SABATON_HEAVY_ARMOR, 1)
          q:take_items(player, PICOT_ARANEIDS_LEG, -1)
        elseif qs:get_memo_state_ex(1) == 2 then
          q:give_items(player, CLAN_OATH_HELM, 1)
          q:give_items(player, CLAN_OATH_BRIGANDINE, 1)
          q:give_items(player, CLAN_OATH_LEATHER_GLOVES_LIGHT_ARMOR, 1)
          q:give_items(player, CLAN_OATH_BOOTS_LIGHT_ARMOR, 1)
          q:take_items(player, PICOT_ARANEIDS_LEG, -1)
        elseif qs:get_memo_state_ex(1) == 3 then
          q:give_items(player, CLAN_OATH_HELM, 1)
          q:give_items(player, CLAN_OATH_AKETON, 1)
          q:give_items(player, CLAN_OATH_PADDED_GLOVES_ROBE, 1)
          q:give_items(player, CLAN_OATH_SANDALS_ROBE, 1)
          q:take_items(player, PICOT_ARANEIDS_LEG, -1)
        end
        qs:exit_quest(false, true)
        htmltext = "31961-17.html"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_started() then
    local sw = npc:id()
    if sw == BRUIN_LIZARDMAN then
      if qs:is_memo_state(1) and q:give_item_randomly(killer, npc, BRUIN_LIZARDMAN_BLOOD, 1, 1, 10, 7, true) then
        qs:set_cond(2)
      end
    elseif sw == PICOT_ARANEID then
      if qs:is_memo_state(4) and killer:sponsor() > 0 then
        local c0 = world.get_player(killer:sponsor())
        if c0 ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, c0, true) and q:give_item_randomly(killer, npc, PICOT_ARANEIDS_LEG, 1, 1, 8, 7, true) then
          qs:set_cond(8)
        end
      end
    end
  end
end)
