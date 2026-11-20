-- Translated by questport from quests/Q00234_FatesWhisper (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(234, "Q00234_FatesWhisper", "Fate's Whisper")
-- L2Go: cond 7 needs a hit on Baium (29020) with the Pipette Knife, and
-- Baium is a grand boss, which L2Go does not spawn yet.
q:requires("grand_bosses")

-- Items
local REIRIA_SOUL_ORB = 4666
local KERMON_INFERNIUM_SCEPTER = 4667
local GOLKONDA_INFERNIUM_SCEPTER = 4668
local HALLATE_INFERNIUM_SCEPTER = 4669
local INFERNIUM_VARNISH = 4672
local REORIN_HAMMER = 4670
local REORIN_MOLD = 4671
local PIPETTE_KNIFE = 4665
local RED_PIPETTE_KNIFE = 4673
local CRYSTAL_B = 1460
-- Reward
local STAR_OF_DESTINY = 5011
-- Chest Spawn
local CHEST_SPAWN = {}
-- Weapons
local WEAPONS = {}

CHEST_SPAWN[25035] = 31027
CHEST_SPAWN[25054] = 31028
CHEST_SPAWN[25126] = 31029
CHEST_SPAWN[25220] = 31030
WEAPONS[79] = "Sword of Damascus"
WEAPONS[97] = "Lance"
WEAPONS[171] = "Deadman's Glory"
WEAPONS[175] = "Art of Battle Axe"
WEAPONS[210] = "Staff of Evil Spirits"
WEAPONS[234] = "Demon Dagger"
WEAPONS[268] = "Bellion Cestus"
WEAPONS[287] = "Bow of Peril"
WEAPONS[2626] = "Samurai Dual-sword"
WEAPONS[7883] = "Guardian Sword"
WEAPONS[7889] = "Wizard's Tear"
WEAPONS[7893] = "Kaim Vanul's Bones"
WEAPONS[7901] = "Star Buster"

q:register_quest_items(PIPETTE_KNIFE, RED_PIPETTE_KNIFE)
q:add_start_npc(31002)
q:add_talk_id(31002, 30182, 30847, 30178, 30833, 31028, 31029, 31030, 31027)
-- The 4 bosses which spawn chests
q:add_kill_id(25035, 25054, 25126, 25220)
-- Baium
q:add_attack_id(29020)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31002-03.htm" then
    st:start_quest()
  elseif event == "30182-01c.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, INFERNIUM_VARNISH, 1)
  elseif event == "30178-01a.htm" then
    st:set_cond(6, true)
  elseif event == "30833-01b.htm" then
    st:set_cond(7, true)
    q:give_items(player, PIPETTE_KNIFE, 1)
  elseif J.starts_with(event, "selectBGrade_") then
    if st:get_int("bypass") == 1 then
      return nil
    end
    local bGradeId = J.replace(event, "selectBGrade_", "")
    st:set("weaponId", bGradeId)
    htmltext = J.replace(q:get_htm(player, "31002-13.htm"), "%weaponname%", WEAPONS[st:get_int("weaponId")])
  elseif J.starts_with(event, "confirmWeapon") then
    st:set("bypass", "1")
    htmltext = J.replace(q:get_htm(player, "31002-14.htm"), "%weaponname%", WEAPONS[st:get_int("weaponId")])
  elseif J.starts_with(event, "selectAGrade_") then
    if st:get_int("bypass") == 1 then
      local itemId = st:get_int("weaponId")
      if q:has_quest_items(player, itemId) then
        local aGradeItemId = J.parse_int(J.replace(event, "selectAGrade_", ""))
        htmltext = J.replace(q:get_htm(player, "31002-12.htm"), "%weaponname%", items.get(aGradeItemId).name)
        q:take_items(player, itemId, 1)
        q:give_items(player, aGradeItemId, 1)
        q:give_items(player, STAR_OF_DESTINY, 1)
        player:social_action(3)
        st:exit_quest(false, true)
      else
        htmltext = J.replace(q:get_htm(player, "31002-15.htm"), "%weaponname%", WEAPONS[itemId])
      end
    else
      htmltext = "31002-16.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "31002-01.htm" or "31002-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == 31002 then
      if cond == 1 then
        if not q:has_quest_items(player, REIRIA_SOUL_ORB) then
          htmltext = "31002-04b.htm"
        else
          htmltext = "31002-05.htm"
          st:set_cond(2, true)
          q:take_items(player, REIRIA_SOUL_ORB, 1)
        end
      elseif cond == 2 then
        if not q:has_quest_items(player, KERMON_INFERNIUM_SCEPTER) or not q:has_quest_items(player, GOLKONDA_INFERNIUM_SCEPTER) or not q:has_quest_items(player, HALLATE_INFERNIUM_SCEPTER) then
          htmltext = "31002-05c.htm"
        else
          htmltext = "31002-06.htm"
          st:set_cond(3, true)
          q:take_items(player, GOLKONDA_INFERNIUM_SCEPTER, 1)
          q:take_items(player, HALLATE_INFERNIUM_SCEPTER, 1)
          q:take_items(player, KERMON_INFERNIUM_SCEPTER, 1)
        end
      elseif cond == 3 then
        if not q:has_quest_items(player, INFERNIUM_VARNISH) then
          htmltext = "31002-06b.htm"
        else
          htmltext = "31002-07.htm"
          st:set_cond(4, true)
          q:take_items(player, INFERNIUM_VARNISH, 1)
        end
      elseif cond == 4 then
        if not q:has_quest_items(player, REORIN_HAMMER) then
          htmltext = "31002-07b.htm"
        else
          htmltext = "31002-08.htm"
          st:set_cond(5, true)
          q:take_items(player, REORIN_HAMMER, 1)
        end
      elseif cond > 4 and cond < 8 then
        htmltext = "31002-08b.htm"
      elseif cond == 8 then
        htmltext = "31002-09.htm"
        st:set_cond(9, true)
        q:take_items(player, REORIN_MOLD, 1)
      elseif cond == 9 then
        if q:get_quest_items_count(player, CRYSTAL_B) < 984 then
          htmltext = "31002-09b.htm"
        else
          htmltext = "31002-BGradeList.htm"
          st:set_cond(10, true)
          q:take_items(player, CRYSTAL_B, 984)
        end
      elseif cond == 10 then
        -- If a weapon is selected
        if st:get_int("bypass") == 1 then
          -- If you got it in the inventory
          local itemId = st:get_int("weaponId")
          htmltext = J.replace(q:get_htm(player, q:has_quest_items(player, itemId) and "31002-AGradeList.htm" or "31002-15.htm"), "%weaponname%", WEAPONS[itemId])
        else
          htmltext = "31002-BGradeList.htm"
        end
      end
    elseif sw2 == 30182 then
      if cond == 3 then
        htmltext = not q:has_quest_items(player, INFERNIUM_VARNISH) and "30182-01.htm" or "30182-02.htm"
      end
    elseif sw2 == 30847 then
      if cond == 4 and not q:has_quest_items(player, REORIN_HAMMER) then
        htmltext = "30847-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, REORIN_HAMMER, 1)
      elseif cond >= 4 and q:has_quest_items(player, REORIN_HAMMER) then
        htmltext = "30847-02.htm"
      end
    elseif sw2 == 30178 then
      if cond == 5 then
        htmltext = "30178-01.htm"
      elseif cond > 5 then
        htmltext = "30178-02.htm"
      end
    elseif sw2 == 30833 then
      if cond == 6 then
        htmltext = "30833-01.htm"
      elseif cond == 7 then
        if q:has_quest_items(player, PIPETTE_KNIFE) and not q:has_quest_items(player, RED_PIPETTE_KNIFE) then
          htmltext = "30833-02.htm"
        else
          htmltext = "30833-03.htm"
          st:set_cond(8, true)
          q:take_items(player, RED_PIPETTE_KNIFE, 1)
          q:give_items(player, REORIN_MOLD, 1)
        end
      elseif cond > 7 then
        htmltext = "30833-04.htm"
      end
    elseif sw2 == 31027 then
      if cond == 1 and not q:has_quest_items(player, REIRIA_SOUL_ORB) then
        htmltext = "31027-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, REIRIA_SOUL_ORB, 1)
      else
        htmltext = "31027-02.htm"
      end
    elseif sw2 == 31028 or sw2 == 31029 or sw2 == 31030 then
      local itemId = npc:id() - 26361
      if cond == 2 and not q:has_quest_items(player, itemId) then
        htmltext = npc:id() .. "-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, itemId, 1)
      else
        htmltext = npc:id() .. "-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = attacker:get_quest_state(q:name())
  if st == nil or not st:is_cond(7) then
    return
  end
  if attacker:active_weapon() ~= nil and attacker:active_weapon() == PIPETTE_KNIFE and not q:has_quest_items(attacker, RED_PIPETTE_KNIFE) then
    q:play_sound(attacker, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(attacker, PIPETTE_KNIFE, 1)
    q:give_items(attacker, RED_PIPETTE_KNIFE, 1)
  end
end)

q:on_kill(function(npc, player, isPet)
  q:add_spawn(CHEST_SPAWN[npc:id()], {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 120000)
end)
