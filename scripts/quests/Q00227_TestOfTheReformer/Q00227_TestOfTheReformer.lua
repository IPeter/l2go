-- Translated by questport from quests/Q00227_TestOfTheReformer (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(227, "Q00227_TestOfTheReformer", "Test of the Reformer")

-- NPCs
local PUPINA = 30118
local SLA = 30666
local RAMUS = 30667
local KATARI = 30668
local KAKAN = 30669
local NYAKURI = 30670
local OL_MAHUM_PILGRIM = 30732
-- Monsters
local MISERY_SKELETON = 20022
local SKELETON_ARCHER = 20100
local SKELETON_MARKSMAN = 20102
local SKELETON_LORD = 20104
local SILENT_HORROR = 20404
local NAMELESS_REVENANT = 27099
local ARURAUNE = 27128
local OL_MAHUM_INSPECTOR = 27129
local OL_MAHUM_BETRAYER = 27130
local CRIMSON_WEREWOLF = 27131
local KRUDEL_LIZARDMAN = 27132
-- Items
local BOOK_OF_REFORM = 2822
local LETTER_OF_INTRODUCTION = 2823
local SLA_LETTER = 2824
local GREETINGS = 2825
local OL_MAHUM_MONEY = 2826
local KATARI_LETTER = 2827
local NYAKURI_LETTER = 2828
local UNDEAD_LIST = 2829
local RAMUS_LETTER = 2830
local RIPPED_DIARY = 2831
local HUGE_NAIL = 2832
local LETTER_OF_BETRAYER = 2833
local BONE_FRAGMENT_4 = 2834
local BONE_FRAGMENT_5 = 2835
local BONE_FRAGMENT_6 = 2836
local BONE_FRAGMENT_7 = 2837
local BONE_FRAGMENT_8 = 2838
local BONE_FRAGMENT_9 = 2839
local KAKAN_LETTER = 3037
-- Rewards
local MARK_OF_REFORMER = 2821
local DIMENSIONAL_DIAMOND = 7562
-- Checks & Instances
local _timer = 0
-- Misc
local _olMahumInspector
local _olMahumPilgrim
local _olMahumBetrayer
local _crimsonWerewolf = false
local _krudelLizardman = false
-- Allowed skills when attacking Crimson Werewolf
local ALLOWED_SKILLS = {1031, 1069, 1147, 1164, 1168, 1177, 1184, 1201, 1206, 4192}
local DISRUPT_UNDEAD = 1031

q:register_quest_items(BOOK_OF_REFORM, LETTER_OF_INTRODUCTION, SLA_LETTER, GREETINGS, OL_MAHUM_MONEY, KATARI_LETTER, NYAKURI_LETTER, UNDEAD_LIST, RAMUS_LETTER, RIPPED_DIARY, HUGE_NAIL, LETTER_OF_BETRAYER, BONE_FRAGMENT_4, BONE_FRAGMENT_5, BONE_FRAGMENT_6, BONE_FRAGMENT_7, BONE_FRAGMENT_8, BONE_FRAGMENT_9, KAKAN_LETTER)
q:add_start_npc(PUPINA)
q:add_talk_id(PUPINA, SLA, RAMUS, KATARI, KAKAN, NYAKURI, OL_MAHUM_PILGRIM)
q:add_attack_id(NAMELESS_REVENANT, CRIMSON_WEREWOLF)
q:add_kill_id(MISERY_SKELETON, SKELETON_ARCHER, SKELETON_MARKSMAN, SKELETON_LORD, SILENT_HORROR, NAMELESS_REVENANT, ARURAUNE, OL_MAHUM_INSPECTOR, OL_MAHUM_BETRAYER, CRIMSON_WEREWOLF, KRUDEL_LIZARDMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30118-04.htm" then
    st:start_quest()
    q:give_items(player, BOOK_OF_REFORM, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30118-04b.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30118-06.htm" then
    st:set_cond(4, true)
    q:take_items(player, BOOK_OF_REFORM, 1)
    q:take_items(player, HUGE_NAIL, 1)
    q:give_items(player, LETTER_OF_INTRODUCTION, 1)
  elseif event == "30666-04.htm" then
    st:set_cond(5, true)
    q:take_items(player, LETTER_OF_INTRODUCTION, 1)
    q:give_items(player, SLA_LETTER, 1)
  elseif event == "30669-03.htm" then
    if not st:is_cond(12) then
      st:set_cond(12, true)
    end
    if not _crimsonWerewolf then
      -- It may only be hunted with low level spells, so it never resists magic whatever magic success formula is configured.
      -- L2Go: Mobius also sets the werewolf's MAGIC_SUCCESS_RES stat to 0;
      -- the script API cannot set NPC stats yet.
      q:add_spawn(CRIMSON_WEREWOLF, {x = -9382, y = -89852, z = -2333, heading = 0}, false, 299000)
      _crimsonWerewolf = true
      -- Resets Crimson Werewolf
      q:start_quest_timer("werewolf_cleanup", 300000, nil, player, false)
    end
  elseif event == "30670-03.htm" then
    st:set_cond(15, true)
    if not _krudelLizardman then
      q:add_spawn(KRUDEL_LIZARDMAN, {x = 126019, y = -179983, z = -1781, heading = 0}, false, 299000)
      _krudelLizardman = true
      -- Resets Krudel Lizardman
      q:start_quest_timer("lizardman_cleanup", 300000, nil, player, false)
    end
  elseif event == "werewolf_despawn" then
    npc:abort_attack()
    npc:broadcast_say(ChatType.GENERAL, "Cowardly guy!")
    npc:delete()
    _crimsonWerewolf = false
    q:cancel_quest_timer("werewolf_cleanup", nil, player)
    return nil
  elseif event == "ol_mahums_despawn" then
    _timer = _timer + 1
    if st:is_cond(8) or _timer >= 60 then
      -- L2Go: J.obj gives nil for an NPC already gone (killed and decayed);
      -- Java's deleteMe on it does nothing.
      if _olMahumPilgrim ~= nil then
        local pilgrim = J.obj(_olMahumPilgrim)
        if pilgrim ~= nil then
          pilgrim:delete()
        end
        _olMahumPilgrim = nil
      end
      if _olMahumInspector ~= nil then
        local inspector = J.obj(_olMahumInspector)
        if inspector ~= nil then
          inspector:delete()
        end
        _olMahumInspector = nil
      end
      q:cancel_quest_timer("ol_mahums_despawn", nil, player)
      _timer = 0
    end
    return nil
  elseif event == "betrayer_despawn" then
    if _olMahumBetrayer ~= nil then
      local betrayer = J.obj(_olMahumBetrayer)
      if betrayer ~= nil then
        betrayer:delete()
      end
      _olMahumBetrayer = nil
    end
    return nil
  elseif event == "werewolf_cleanup" then
    _crimsonWerewolf = false
    return nil
  elseif event == "lizardman_cleanup" then
    _krudelLizardman = false
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() == ClassId.CLERIC or player:class_id() == ClassId.SHILLIEN_ORACLE then
      htmltext = player:level() < 39 and "30118-01.htm" or "30118-03.htm"
    else
      htmltext = "30118-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == PUPINA then
      if cond < 3 then
        htmltext = "30118-04a.htm"
      elseif cond == 3 then
        htmltext = "30118-05.htm"
      elseif cond > 3 then
        htmltext = "30118-07.htm"
      end
    elseif sw2 == SLA then
      if cond == 4 then
        htmltext = "30666-01.htm"
      elseif cond > 4 and cond < 10 then
        htmltext = "30666-05.htm"
      elseif cond == 10 then
        htmltext = "30666-06.htm"
        st:set_cond(11, true)
        q:take_items(player, OL_MAHUM_MONEY, 1)
        q:give_items(player, GREETINGS, 3)
      elseif cond > 10 and cond < 20 then
        htmltext = "30666-06.htm"
      elseif cond == 20 then
        htmltext = "30666-07.htm"
        q:take_items(player, KATARI_LETTER, 1)
        q:take_items(player, KAKAN_LETTER, 1)
        q:take_items(player, NYAKURI_LETTER, 1)
        q:take_items(player, RAMUS_LETTER, 1)
        q:give_items(player, MARK_OF_REFORMER, 1)
        q:add_exp_and_sp(player, 164032, 17500)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == KATARI then
      if cond == 5 or cond == 6 then
        htmltext = "30668-01.htm"
        if cond == 5 then
          st:set_cond(6, true)
          q:take_items(player, SLA_LETTER, 1)
        end
        if _olMahumPilgrim == nil and _olMahumInspector == nil then
          _olMahumPilgrim = J.oid(q:add_spawn(OL_MAHUM_PILGRIM, {x = -4015, y = 40141, z = -3664, heading = 0}, false, 0))
          _olMahumInspector = J.oid(q:add_spawn(OL_MAHUM_INSPECTOR, {x = -4034, y = 40201, z = -3665, heading = 0}, false, 0))
          -- Resets Ol Mahums' instances
          q:start_quest_timer("ol_mahums_despawn", 5000, nil, player, true)
          J.obj(_olMahumInspector):add_damage_hate(J.obj(_olMahumPilgrim), 0, 99999)
          J.obj(_olMahumInspector):attack(J.obj(_olMahumPilgrim))
        end
      elseif cond == 7 then
        htmltext = "30668-01.htm"
        if _olMahumPilgrim == nil then
          _olMahumPilgrim = J.oid(q:add_spawn(OL_MAHUM_PILGRIM, {x = -4015, y = 40141, z = -3664, heading = 0}, false, 0))
          -- Resets Ol Mahums' instances
          q:start_quest_timer("ol_mahums_despawn", 5000, nil, player, true)
        end
      elseif cond == 8 then
        htmltext = "30668-02.htm"
        if _olMahumBetrayer == nil then
          _olMahumBetrayer = J.oid(q:add_spawn(OL_MAHUM_BETRAYER, {x = -4106, y = 40174, z = -3660, heading = 0}, false, 0))
          J.obj(_olMahumBetrayer):set_running()
          J.obj(_olMahumBetrayer):move_to({x = -7732, y = 36787, z = -3709})
          -- Resets Ol Mahum Betrayer's instance
          q:start_quest_timer("betrayer_despawn", 40000, nil, player, false)
        end
      elseif cond == 9 then
        htmltext = "30668-03.htm"
        st:set_cond(10, true)
        q:take_items(player, LETTER_OF_BETRAYER, 1)
        q:give_items(player, KATARI_LETTER, 1)
      elseif cond > 9 then
        htmltext = "30668-04.htm"
      end
    elseif sw2 == OL_MAHUM_PILGRIM then
      if cond == 7 then
        htmltext = "30732-01.htm"
        st:set_cond(8, true)
        q:give_items(player, OL_MAHUM_MONEY, 1)
      end
    elseif sw2 == KAKAN then
      if cond == 11 or cond == 12 then
        htmltext = "30669-01.htm"
      elseif cond == 13 then
        htmltext = "30669-04.htm"
        st:set_cond(14, true)
        q:take_items(player, GREETINGS, 1)
        q:give_items(player, KAKAN_LETTER, 1)
      elseif cond > 13 then
        htmltext = "30669-04.htm"
      end
    elseif sw2 == NYAKURI then
      if cond == 14 or cond == 15 then
        htmltext = "30670-01.htm"
      elseif cond == 16 then
        htmltext = "30670-04.htm"
        st:set_cond(17, true)
        q:take_items(player, GREETINGS, 1)
        q:give_items(player, NYAKURI_LETTER, 1)
      elseif cond > 16 then
        htmltext = "30670-04.htm"
      end
    elseif sw2 == RAMUS then
      if cond == 17 then
        htmltext = "30667-01.htm"
        st:set_cond(18, true)
        q:take_items(player, GREETINGS, 1)
        q:give_items(player, UNDEAD_LIST, 1)
      elseif cond == 18 then
        htmltext = "30667-02.htm"
      elseif cond == 19 then
        htmltext = "30667-03.htm"
        st:set_cond(20, true)
        q:take_items(player, BONE_FRAGMENT_4, 1)
        q:take_items(player, BONE_FRAGMENT_5, 1)
        q:take_items(player, BONE_FRAGMENT_6, 1)
        q:take_items(player, BONE_FRAGMENT_7, 1)
        q:take_items(player, BONE_FRAGMENT_8, 1)
        q:take_items(player, UNDEAD_LIST, 1)
        q:give_items(player, RAMUS_LETTER, 1)
      elseif cond > 19 then
        htmltext = "30667-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet, skill)
  -- Crimson Werewolf leaves as soon as a player hits it with anything but the allowed skills.
  if npc:id() == CRIMSON_WEREWOLF and not isPet and not npc:is_script_value(1) and (skill == nil or not J.contains(ALLOWED_SKILLS, skill:id())) then
    npc:set_script_value(1)
    q:start_quest_timer("werewolf_despawn", 1000, npc, attacker, false)
    return
  end
  local st = q:get_quest_state(attacker, false)
  if st == nil then
    return
  end
  local cond = st:get_cond()
  local sw = npc:id()
  if sw == NAMELESS_REVENANT then
    -- Only a kill right after Disrupt Undead counts, any other hit clears it.
    if cond == 1 or cond == 2 then
      npc:set_script_value(skill ~= nil and skill:id() == DISRUPT_UNDEAD and 1 or 2)
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local cond = st:get_cond()
  local sw = npc:id()
  if sw == NAMELESS_REVENANT then
    if (cond == 1 or cond == 2) and npc:is_script_value(1) and q:get_quest_items_count(player, RIPPED_DIARY) < 7 then
      q:give_items(player, RIPPED_DIARY, 1)
      if q:get_quest_items_count(player, RIPPED_DIARY) >= 7 then
        st:set_cond(2, true)
        q:take_items(player, RIPPED_DIARY, -1)
        q:add_spawn(ARURAUNE, npc, false, 300000)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ARURAUNE then
    if cond == 2 then
      st:set_cond(3, true)
      q:give_items(player, HUGE_NAIL, 1)
      npc:broadcast_say(ChatType.GENERAL, "The concealed truth will always be revealed...!")
    end
  elseif sw == OL_MAHUM_INSPECTOR then
    if cond == 6 then
      st:set_cond(7, true)
    end
  elseif sw == OL_MAHUM_BETRAYER then
    if cond == 8 then
      st:set_cond(9, true)
      q:give_items(player, LETTER_OF_BETRAYER, 1)
      q:cancel_quest_timer("betrayer_despawn", nil, player)
      _olMahumBetrayer = nil
    end
  elseif sw == CRIMSON_WEREWOLF then
    if cond == 12 then
      st:set_cond(13, true)
      q:cancel_quest_timer("werewolf_cleanup", nil, player)
      _crimsonWerewolf = false
    end
  elseif sw == KRUDEL_LIZARDMAN then
    if cond == 15 then
      st:set_cond(16, true)
      q:cancel_quest_timer("lizardman_cleanup", nil, player)
      _krudelLizardman = false
    end
  elseif sw == SILENT_HORROR then
    if cond == 18 and not q:has_quest_items(player, BONE_FRAGMENT_4) then
      q:give_items(player, BONE_FRAGMENT_4, 1)
      if q:has_quest_items(player, BONE_FRAGMENT_5, BONE_FRAGMENT_6, BONE_FRAGMENT_7, BONE_FRAGMENT_8) then
        st:set_cond(19, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SKELETON_LORD then
    if cond == 18 and not q:has_quest_items(player, BONE_FRAGMENT_5) then
      q:give_items(player, BONE_FRAGMENT_5, 1)
      if q:has_quest_items(player, BONE_FRAGMENT_4, BONE_FRAGMENT_6, BONE_FRAGMENT_7, BONE_FRAGMENT_8) then
        st:set_cond(19, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SKELETON_MARKSMAN then
    if cond == 18 and not q:has_quest_items(player, BONE_FRAGMENT_6) then
      q:give_items(player, BONE_FRAGMENT_6, 1)
      if q:has_quest_items(player, BONE_FRAGMENT_4, BONE_FRAGMENT_5, BONE_FRAGMENT_7, BONE_FRAGMENT_8) then
        st:set_cond(19, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MISERY_SKELETON then
    if cond == 18 and not q:has_quest_items(player, BONE_FRAGMENT_7) then
      q:give_items(player, BONE_FRAGMENT_7, 1)
      if q:has_quest_items(player, BONE_FRAGMENT_4, BONE_FRAGMENT_5, BONE_FRAGMENT_6, BONE_FRAGMENT_8) then
        st:set_cond(19, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SKELETON_ARCHER then
    if cond == 18 and not q:has_quest_items(player, BONE_FRAGMENT_8) then
      q:give_items(player, BONE_FRAGMENT_8, 1)
      if q:has_quest_items(player, BONE_FRAGMENT_4, BONE_FRAGMENT_5, BONE_FRAGMENT_6, BONE_FRAGMENT_7) then
        st:set_cond(19, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
