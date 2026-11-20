-- Translated by questport from quests/Q00421_LittleWingsBigAdventure (Mobius CT_0 Interlude); owned by L2Go from here on.
-- iCond is an internal variable, used because cond isn't developped on that quest (only 3 states) :
-- <ul>
-- <li>1-3 leads initial mimyu behavior ;</li>
-- <li>used for leaves support as mask : 4, 8, 16 or 32 = 60 overall ;</li>
-- <li>63 becomes the "marker" to get back to mimyu (60 + 3), meaning you hitted the 4 trees ;</li>
-- <li>setted to 100 if mimyu check is ok.</li>
-- </ul>
--
-- Needs pets: player:summon(), the flute items (player:get_item,
-- item:enchant_level) and minions are not in the API yet; the quest is not
-- offered until then.
local J = require("lib.java")

local q = quest.new(421, "Q00421_LittleWingsBigAdventure", "Little Wing's Big Adventure")
q:requires("pets")

-- NPCs
local CRONOS = 30610
local MIMYU = 30747
-- Item
local FAIRY_LEAF = 4325

q:register_quest_items(FAIRY_LEAF)
q:add_start_npc(CRONOS)
q:add_talk_id(CRONOS, MIMYU)
q:add_attack_id(27185, 27186, 27187, 27188)
q:add_kill_id(27185, 27186, 27187, 27188)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30610-06.htm" then
    if q:get_quest_items_count(player, 3500) + q:get_quest_items_count(player, 3501) + q:get_quest_items_count(player, 3502) == 1 then
      -- Find the level of the flute.
      for i = 3500, 3502 do
        local item = player:get_item(i)
        if item ~= nil and item:enchant_level() >= 55 then
          st:start_quest()
          st:set("iCond", "1")
          st:set("summonOid", tostring(item:object_id()))
          return "30610-05.htm"
        end
      end
    end
    -- Exit quest if you got more than one flute, or the flute level doesn't meat requirements.
    st:exit_quest(true)
  elseif event == "30747-02.htm" then
    local summon = player:summon()
    if summon ~= nil then
      htmltext = summon:control_object_id() == st:get_int("summonOid") and "30747-04.htm" or "30747-03.htm"
    end
  elseif event == "30747-05.htm" then
    local summon = player:summon()
    if summon == nil or summon:control_object_id() ~= st:get_int("summonOid") then
      htmltext = "30747-06.htm"
    else
      st:set_cond(2, true)
      st:set("iCond", "3")
      q:give_items(player, FAIRY_LEAF, 4)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    -- Wrong level.
    if player:level() < 45 then
      htmltext = "30610-01.htm"
    elseif q:get_quest_items_count(player, 3500) + q:get_quest_items_count(player, 3501) + q:get_quest_items_count(player, 3502) ~= 1 then
      htmltext = "30610-02.htm"
    else
      -- Find the level of the hatchling.
      for i = 3500, 3502 do
        local item = player:get_item(i)
        if item ~= nil and item:enchant_level() >= 55 then
          return "30610-04.htm"
        end
      end
      -- Invalid level.
      htmltext = "30610-03.htm"
    end
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == CRONOS then
      htmltext = "30610-07.htm"
    elseif sw2 == MIMYU then
      local id = st:get_int("iCond")
      if id == 1 then
        htmltext = "30747-01.htm"
        st:set("iCond", "2")
      elseif id == 2 then
        local summon = player:summon()
        htmltext = summon ~= nil and (summon:control_object_id() == st:get_int("summonOid") and "30747-04.htm" or "30747-03.htm") or "30747-02.htm"
      elseif id == 3 then
        htmltext = "30747-07.htm"
      elseif id > 3 and id < 63 then
        htmltext = "30747-11.htm"
      elseif id == 63 then
        local summon = player:summon()
        if summon == nil then
          return "30747-12.htm"
        end
        if summon:control_object_id() ~= st:get_int("summonOid") then
          return "30747-14.htm"
        end
        htmltext = "30747-13.htm"
        st:set("iCond", "100")
      elseif id == 100 then
        local summon = player:summon()
        if summon ~= nil and summon:control_object_id() == st:get_int("summonOid") then
          return "30747-15.htm"
        end
        if q:get_quest_items_count(player, 3500) + q:get_quest_items_count(player, 3501) + q:get_quest_items_count(player, 3502) > 1 then
          return "30747-17.htm"
        end
        for i = 3500, 3502 do
          local item = player:get_item(i)
          if item ~= nil and item:object_id() == st:get_int("summonOid") then
            q:take_items(player, i, 1)
            q:give_items(player, i + 922, 1, item:enchant_level()) -- TODO: rebuild entirely pet system in order enchant is given a fuck. Supposed to give an item level XX for a flute level XX.
            st:exit_quest(true, true)
            return "30747-16.htm"
          end
        end
        -- Curse if the registered objectId is the wrong one (switch flutes).
        htmltext = "30747-18.htm"
        local skill = skills.get(4167, 1)
        if skill ~= nil and player:is_affected_by_skill(skill:id()) then
          skills.apply_effects(skill, npc, player)
        end
      end
    end
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  -- Minions scream no matter current quest state.
  if npc:has_minions() then
    for _, ghost in ipairs(npc:minions()) do
      if not ghost:is_dead() and rnd(100) < 1 then
        ghost:broadcast_say(ChatType.GENERAL, "We must protect the fairy tree!")
      end
    end
  end
  -- Condition required : 2.
  local st = q:get_quest_state(attacker, false)
  if st == nil or not st:is_cond(2) then
    return
  end
  -- A pet was the attacker, and the objectId is the good one.
  if isPet and attacker:summon():control_object_id() == st:get_int("summonOid") then
    -- Random luck is reached and you still have some leaves ; go further.
    if rnd(100) < 1 and q:has_quest_items(st:get_player(), FAIRY_LEAF) then
      local idMask = J.int(math.pow(2, npc:id() - 27182 - 1))
      local iCond = st:get_int("iCond")
      if J.bor(iCond, idMask) ~= iCond then
        st:set("iCond", tostring(J.bor(iCond, idMask)))
        npc:broadcast_say(ChatType.GENERAL, "Give me a Fairy Leaf...!")
        q:take_items(st:get_player(), FAIRY_LEAF, 1)
        npc:broadcast_say(ChatType.GENERAL, "Leave now, before you incur the wrath of the guardian ghost...")
        -- Four leafs have been used ; update quest state.
        if st:get_int("iCond") == 63 then
          st:set_cond(3, true)
        else
          q:play_sound(st:get_player(), QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)

q:on_kill(function(npc, killer, isPet)
  local originalKiller
  if isPet then
    originalKiller = killer:summon()
  else
    originalKiller = killer
  end
  -- Tree curses the killer.
  if rnd(100) < 30 and originalKiller ~= nil then
    local skill = skills.get(4243, 1)
    if skill ~= nil and originalKiller:is_affected_by_skill(skill:id()) then
      skills.apply_effects(skill, npc, originalKiller)
    end
  end
  -- Spawn 20 ghosts, attacking the killer.
  for i = 0, 19 do
    local newNpc = q:add_spawn(27189, {x = npc:x(), y = npc:y(), z = npc:z(), heading = rnd(65536)}, true, 300000)
    newNpc:set_running()
    newNpc:add_damage_hate(originalKiller, 0, 999)
    newNpc:attack(originalKiller)
  end
end)
