-- Translated by questport from quests/Q00333_HuntOfTheBlackLion (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(333, "Q00333_HuntOfTheBlackLion", "Hunt of the Black Lion")

-- NPCs
local SOPHYA = 30735
local REDFOOT = 30736
local RUPIO = 30471
local UNDRIAS = 30130
local LOCKIRIN = 30531
local MORGAN = 30737
-- Needs for start
local BLACK_LION_MARK = 1369
-- Quest items
local LION_CLAW = 3675
local LION_EYE = 3676
local GUILD_COIN = 3677
local UNDEAD_ASH = 3848
local BLOODY_AXE_INSIGNIA = 3849
local DELU_FANG = 3850
local STAKATO_TALON = 3851
local SOPHYA_LETTER_1 = 3671
local SOPHYA_LETTER_2 = 3672
local SOPHYA_LETTER_3 = 3673
local SOPHYA_LETTER_4 = 3674
local CARGO_BOX_1 = 3440
local CARGO_BOX_2 = 3441
local CARGO_BOX_3 = 3442
local CARGO_BOX_4 = 3443
local GLUDIO_APPLE = 3444
local CORN_MEAL = 3445
local WOLF_PELTS = 3446
local MOONSTONE = 3447
local GLUDIO_WHEAT_FLOWER = 3448
local SPIDERSILK_ROPE = 3449
local ALEXANDRITE = 3450
local SILVER_TEA = 3451
local GOLEM_PART = 3452
local FIRE_EMERALD = 3453
local SILK_FROCK = 3454
local PORCELAN_URN = 3455
local IMPERIAL_DIAMOND = 3456
local STATUE_SHILIEN_HEAD = 3457
local STATUE_SHILIEN_TORSO = 3458
local STATUE_SHILIEN_ARM = 3459
local STATUE_SHILIEN_LEG = 3460
local COMPLETE_STATUE = 3461
local TABLET_FRAGMENT_1 = 3462
local TABLET_FRAGMENT_2 = 3463
local TABLET_FRAGMENT_3 = 3464
local TABLET_FRAGMENT_4 = 3465
local COMPLETE_TABLET = 3466
-- Neutral items
local ADENA = 57
local SWIFT_ATTACK_POTION = 735
local SCROLL_OF_ESCAPE = 736
local HEALING_POTION = 1061
local SOULSHOT_D = 1463
local SPIRITSHOT_D = 2510
-- Tabs: Part, NpcId, ItemId, Item Chance, Box Id, Box Chance
local DROPLIST = {
  {SOPHYA_LETTER_1, 20160, UNDEAD_ASH, 500000, CARGO_BOX_1, 90000},
  {SOPHYA_LETTER_1, 20171, UNDEAD_ASH, 500000, CARGO_BOX_1, 60000},
  {SOPHYA_LETTER_1, 20197, UNDEAD_ASH, 500000, CARGO_BOX_1, 70000},
  {SOPHYA_LETTER_1, 20198, UNDEAD_ASH, 500000, CARGO_BOX_1, 80000},
  {SOPHYA_LETTER_1, 20200, UNDEAD_ASH, 500000, CARGO_BOX_1, 100000},
  {SOPHYA_LETTER_1, 20201, UNDEAD_ASH, 500000, CARGO_BOX_1, 110000},
  {SOPHYA_LETTER_2, 20207, BLOODY_AXE_INSIGNIA, 500000, CARGO_BOX_2, 60000},
  {SOPHYA_LETTER_2, 20208, BLOODY_AXE_INSIGNIA, 500000, CARGO_BOX_2, 70000},
  {SOPHYA_LETTER_2, 20209, BLOODY_AXE_INSIGNIA, 500000, CARGO_BOX_2, 80000},
  {SOPHYA_LETTER_2, 20210, BLOODY_AXE_INSIGNIA, 500000, CARGO_BOX_2, 90000},
  {SOPHYA_LETTER_2, 20211, BLOODY_AXE_INSIGNIA, 500000, CARGO_BOX_2, 100000},
  {SOPHYA_LETTER_3, 20251, DELU_FANG, 500000, CARGO_BOX_3, 100000},
  {SOPHYA_LETTER_3, 20252, DELU_FANG, 500000, CARGO_BOX_3, 110000},
  {SOPHYA_LETTER_3, 20253, DELU_FANG, 500000, CARGO_BOX_3, 120000},
  {SOPHYA_LETTER_4, 20157, STAKATO_TALON, 500000, CARGO_BOX_4, 100000},
  {SOPHYA_LETTER_4, 20230, STAKATO_TALON, 500000, CARGO_BOX_4, 110000},
  {SOPHYA_LETTER_4, 20232, STAKATO_TALON, 500000, CARGO_BOX_4, 120000},
  {SOPHYA_LETTER_4, 20234, STAKATO_TALON, 500000, CARGO_BOX_4, 130000},
}

q:register_quest_items(LION_CLAW, LION_EYE, GUILD_COIN, UNDEAD_ASH, BLOODY_AXE_INSIGNIA, DELU_FANG, STAKATO_TALON, SOPHYA_LETTER_1, SOPHYA_LETTER_2, SOPHYA_LETTER_3, SOPHYA_LETTER_4)
q:add_start_npc(SOPHYA)
q:add_talk_id(SOPHYA, REDFOOT, RUPIO, UNDRIAS, LOCKIRIN, MORGAN)
for _, i in ipairs(DROPLIST) do
  q:add_kill_id(i[2])
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30735-04.htm" then
    st:start_quest()
  elseif event == "30735-10.htm" then
    if not q:has_quest_items(player, SOPHYA_LETTER_1) then
      q:give_items(player, SOPHYA_LETTER_1, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30735-11.htm" then
    if not q:has_quest_items(player, SOPHYA_LETTER_2) then
      q:give_items(player, SOPHYA_LETTER_2, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30735-12.htm" then
    if not q:has_quest_items(player, SOPHYA_LETTER_3) then
      q:give_items(player, SOPHYA_LETTER_3, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30735-13.htm" then
    if not q:has_quest_items(player, SOPHYA_LETTER_4) then
      q:give_items(player, SOPHYA_LETTER_4, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30735-16.htm" then
    if q:get_quest_items_count(player, LION_CLAW) > 9 then
      q:take_items(player, LION_CLAW, 10)
      local eyes = q:get_quest_items_count(player, LION_EYE)
      if eyes < 5 then
        htmltext = "30735-17a.htm"
        q:give_items(player, LION_EYE, 1)
        local random = rnd(100)
        if random < 25 then
          q:give_items(player, HEALING_POTION, 20)
        elseif random < 50 then
          q:give_items(player, player:is_mage_class() and SPIRITSHOT_D or SOULSHOT_D, player:is_mage_class() and 50 or 100)
        elseif random < 75 then
          q:give_items(player, SCROLL_OF_ESCAPE, 20)
        else
          q:give_items(player, SWIFT_ATTACK_POTION, 3)
        end
      elseif eyes < 9 then
        htmltext = "30735-18b.htm"
        q:give_items(player, LION_EYE, 1)
        local random = rnd(100)
        if random < 25 then
          q:give_items(player, HEALING_POTION, 25)
        elseif random < 50 then
          q:give_items(player, player:is_mage_class() and SPIRITSHOT_D or SOULSHOT_D, player:is_mage_class() and 100 or 200)
        elseif random < 75 then
          q:give_items(player, SCROLL_OF_ESCAPE, 20)
        else
          q:give_items(player, SWIFT_ATTACK_POTION, 3)
        end
      else
        htmltext = "30735-19b.htm"
        local random = rnd(100)
        if random < 25 then
          q:give_items(player, HEALING_POTION, 50)
        elseif random < 50 then
          q:give_items(player, player:is_mage_class() and SPIRITSHOT_D or SOULSHOT_D, player:is_mage_class() and 200 or 400)
        elseif random < 75 then
          q:give_items(player, SCROLL_OF_ESCAPE, 30)
        else
          q:give_items(player, SWIFT_ATTACK_POTION, 4)
        end
      end
    end
  elseif event == "30735-20.htm" then
    q:take_items(player, SOPHYA_LETTER_1, -1)
    q:take_items(player, SOPHYA_LETTER_2, -1)
    q:take_items(player, SOPHYA_LETTER_3, -1)
    q:take_items(player, SOPHYA_LETTER_4, -1)
  elseif event == "30735-26.htm" then
    q:take_items(player, LION_CLAW, -1)
    q:take_items(player, LION_EYE, -1)
    q:take_items(player, GUILD_COIN, -1)
    q:take_items(player, BLACK_LION_MARK, -1)
    q:take_items(player, SOPHYA_LETTER_1, -1)
    q:take_items(player, SOPHYA_LETTER_2, -1)
    q:take_items(player, SOPHYA_LETTER_3, -1)
    q:take_items(player, SOPHYA_LETTER_4, -1)
    q:give_items(player, ADENA, 12400)
    st:exit_quest(true, true)
  elseif event == "30736-03.htm" then
    local cargo1 = q:has_quest_items(player, CARGO_BOX_1)
    local cargo2 = q:has_quest_items(player, CARGO_BOX_2)
    local cargo3 = q:has_quest_items(player, CARGO_BOX_3)
    local cargo4 = q:has_quest_items(player, CARGO_BOX_4)
    if (cargo1 or cargo2 or cargo3 or cargo4) and player:adena() > 649 then
      q:take_items(player, ADENA, 650)
      if cargo1 then
        q:take_items(player, CARGO_BOX_1, 1)
      elseif cargo2 then
        q:take_items(player, CARGO_BOX_2, 1)
      elseif cargo3 then
        q:take_items(player, CARGO_BOX_3, 1)
      else
        q:take_items(player, CARGO_BOX_4, 1)
      end
      local i0 = rnd(100)
      local i1 = rnd(100)
      if i0 < 40 then
        if i1 < 33 then
          htmltext = "30736-04a.htm"
          q:give_items(player, GLUDIO_APPLE, 1)
        elseif i1 < 66 then
          htmltext = "30736-04b.htm"
          q:give_items(player, CORN_MEAL, 1)
        else
          htmltext = "30736-04c.htm"
          q:give_items(player, WOLF_PELTS, 1)
        end
      elseif i0 < 60 then
        if i1 < 33 then
          htmltext = "30736-04d.htm"
          q:give_items(player, MOONSTONE, 1)
        elseif i1 < 66 then
          htmltext = "30736-04e.htm"
          q:give_items(player, GLUDIO_WHEAT_FLOWER, 1)
        else
          htmltext = "30736-04f.htm"
          q:give_items(player, SPIDERSILK_ROPE, 1)
        end
      elseif i0 < 70 then
        if i1 < 33 then
          htmltext = "30736-04g.htm"
          q:give_items(player, ALEXANDRITE, 1)
        elseif i1 < 66 then
          htmltext = "30736-04h.htm"
          q:give_items(player, SILVER_TEA, 1)
        else
          htmltext = "30736-04i.htm"
          q:give_items(player, GOLEM_PART, 1)
        end
      elseif i0 < 75 then
        if i1 < 33 then
          htmltext = "30736-04j.htm"
          q:give_items(player, FIRE_EMERALD, 1)
        elseif i1 < 66 then
          htmltext = "30736-04k.htm"
          q:give_items(player, SILK_FROCK, 1)
        else
          htmltext = "30736-04l.htm"
          q:give_items(player, PORCELAN_URN, 1)
        end
      elseif i0 < 76 then
        htmltext = "30736-04m.htm"
        q:give_items(player, IMPERIAL_DIAMOND, 1)
      elseif rnd.bool() then
        htmltext = "30736-04n.htm"
        if i1 < 25 then
          q:give_items(player, STATUE_SHILIEN_HEAD, 1)
        elseif i1 < 50 then
          q:give_items(player, STATUE_SHILIEN_TORSO, 1)
        elseif i1 < 75 then
          q:give_items(player, STATUE_SHILIEN_ARM, 1)
        else
          q:give_items(player, STATUE_SHILIEN_LEG, 1)
        end
      else
        htmltext = "30736-04o.htm"
        if i1 < 25 then
          q:give_items(player, TABLET_FRAGMENT_1, 1)
        elseif i1 < 50 then
          q:give_items(player, TABLET_FRAGMENT_2, 1)
        elseif i1 < 75 then
          q:give_items(player, TABLET_FRAGMENT_3, 1)
        else
          q:give_items(player, TABLET_FRAGMENT_4, 1)
        end
      end
    else
      htmltext = "30736-05.htm"
    end
  elseif event == "30736-07.htm" then
    local state = st:get_int("state")
    if player:adena() > 200 + state * 200 then
      if state < 3 then
        local i0 = rnd(100)
        if i0 < 5 then
          htmltext = "30736-08a.htm"
        elseif i0 < 10 then
          htmltext = "30736-08b.htm"
        elseif i0 < 15 then
          htmltext = "30736-08c.htm"
        elseif i0 < 20 then
          htmltext = "30736-08d.htm"
        elseif i0 < 25 then
          htmltext = "30736-08e.htm"
        elseif i0 < 30 then
          htmltext = "30736-08f.htm"
        elseif i0 < 35 then
          htmltext = "30736-08g.htm"
        elseif i0 < 40 then
          htmltext = "30736-08h.htm"
        elseif i0 < 45 then
          htmltext = "30736-08i.htm"
        elseif i0 < 50 then
          htmltext = "30736-08j.htm"
        elseif i0 < 55 then
          htmltext = "30736-08k.htm"
        elseif i0 < 60 then
          htmltext = "30736-08l.htm"
        elseif i0 < 65 then
          htmltext = "30736-08m.htm"
        elseif i0 < 70 then
          htmltext = "30736-08n.htm"
        elseif i0 < 75 then
          htmltext = "30736-08o.htm"
        elseif i0 < 80 then
          htmltext = "30736-08p.htm"
        elseif i0 < 85 then
          htmltext = "30736-08q.htm"
        elseif i0 < 90 then
          htmltext = "30736-08r.htm"
        elseif i0 < 95 then
          htmltext = "30736-08s.htm"
        else
          htmltext = "30736-08t.htm"
        end
        q:take_items(player, ADENA, 200 + state * 200)
        st:set("state", tostring(state + 1))
      else
        htmltext = "30736-08.htm"
      end
    end
  elseif event == "30471-03.htm" then
    if q:has_quest_items(player, STATUE_SHILIEN_HEAD, STATUE_SHILIEN_TORSO, STATUE_SHILIEN_ARM, STATUE_SHILIEN_LEG) then
      q:take_items(player, STATUE_SHILIEN_HEAD, 1)
      q:take_items(player, STATUE_SHILIEN_TORSO, 1)
      q:take_items(player, STATUE_SHILIEN_ARM, 1)
      q:take_items(player, STATUE_SHILIEN_LEG, 1)
      if rnd.bool() then
        htmltext = "30471-04.htm"
        q:give_items(player, COMPLETE_STATUE, 1)
      else
        htmltext = "30471-05.htm"
      end
    end
  elseif event == "30471-06.htm" then
    if q:has_quest_items(player, TABLET_FRAGMENT_1, TABLET_FRAGMENT_2, TABLET_FRAGMENT_3, TABLET_FRAGMENT_4) then
      q:take_items(player, TABLET_FRAGMENT_1, 1)
      q:take_items(player, TABLET_FRAGMENT_2, 1)
      q:take_items(player, TABLET_FRAGMENT_3, 1)
      q:take_items(player, TABLET_FRAGMENT_4, 1)
      if rnd.bool() then
        htmltext = "30471-07.htm"
        q:give_items(player, COMPLETE_TABLET, 1)
      else
        htmltext = "30471-08.htm"
      end
    end
  elseif event == "30130-04.htm" then
    if q:has_quest_items(player, COMPLETE_STATUE) then
      q:take_items(player, COMPLETE_STATUE, 1)
      q:give_items(player, ADENA, 30000)
    end
  elseif event == "30531-04.htm" then
    if q:has_quest_items(player, COMPLETE_TABLET) then
      q:take_items(player, COMPLETE_TABLET, 1)
      q:give_items(player, ADENA, 30000)
    end
  elseif event == "30737-06.htm" then
    local cargo1 = q:has_quest_items(player, CARGO_BOX_1)
    local cargo2 = q:has_quest_items(player, CARGO_BOX_2)
    local cargo3 = q:has_quest_items(player, CARGO_BOX_3)
    local cargo4 = q:has_quest_items(player, CARGO_BOX_4)
    if cargo1 or cargo2 or cargo3 or cargo4 then
      if cargo1 then
        q:take_items(player, CARGO_BOX_1, 1)
      elseif cargo2 then
        q:take_items(player, CARGO_BOX_2, 1)
      elseif cargo3 then
        q:take_items(player, CARGO_BOX_3, 1)
      else
        q:take_items(player, CARGO_BOX_4, 1)
      end
      local coins = q:get_quest_items_count(player, GUILD_COIN)
      if coins < 40 then
        htmltext = "30737-03.htm"
        q:give_items(player, ADENA, 100)
      elseif coins < 80 then
        htmltext = "30737-04.htm"
        q:give_items(player, ADENA, 200)
      else
        htmltext = "30737-05.htm"
        q:give_items(player, ADENA, 300)
      end
      if coins < 80 then
        q:give_items(player, GUILD_COIN, 1)
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 25 then
      htmltext = "30735-01.htm"
    elseif not q:has_quest_items(player, BLACK_LION_MARK) then
      htmltext = "30735-02.htm"
    else
      htmltext = "30735-03.htm"
    end
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SOPHYA then
      if not q:has_at_least_one_quest_item(player, SOPHYA_LETTER_1, SOPHYA_LETTER_2, SOPHYA_LETTER_3, SOPHYA_LETTER_4) then
        htmltext = "30735-14.htm"
      else
        if not q:has_at_least_one_quest_item(player, UNDEAD_ASH, BLOODY_AXE_INSIGNIA, DELU_FANG, STAKATO_TALON) then
          htmltext = q:has_at_least_one_quest_item(player, CARGO_BOX_1, CARGO_BOX_2, CARGO_BOX_3, CARGO_BOX_4) and "30735-15a.htm" or "30735-15.htm"
        else
          local count = q:get_quest_items_count(player, UNDEAD_ASH) + q:get_quest_items_count(player, BLOODY_AXE_INSIGNIA) + q:get_quest_items_count(player, DELU_FANG) + q:get_quest_items_count(player, STAKATO_TALON)
          q:take_items(player, UNDEAD_ASH, -1)
          q:take_items(player, BLOODY_AXE_INSIGNIA, -1)
          q:take_items(player, DELU_FANG, -1)
          q:take_items(player, STAKATO_TALON, -1)
          q:give_items(player, ADENA, count * 35)
          if count >= 20 and count < 50 then
            q:give_items(player, LION_CLAW, 1)
          elseif count >= 50 and count < 100 then
            q:give_items(player, LION_CLAW, 2)
          elseif count >= 100 then
            q:give_items(player, LION_CLAW, 3)
          end
          htmltext = q:has_at_least_one_quest_item(player, CARGO_BOX_1, CARGO_BOX_2, CARGO_BOX_3, CARGO_BOX_4) and "30735-23.htm" or "30735-22.htm"
        end
      end
    elseif sw2 == REDFOOT then
      htmltext = q:has_at_least_one_quest_item(player, CARGO_BOX_1, CARGO_BOX_2, CARGO_BOX_3, CARGO_BOX_4) and "30736-02.htm" or "30736-01.htm"
    elseif sw2 == RUPIO then
      if q:has_quest_items(player, STATUE_SHILIEN_HEAD, STATUE_SHILIEN_TORSO, STATUE_SHILIEN_ARM, STATUE_SHILIEN_LEG) or q:has_quest_items(player, TABLET_FRAGMENT_1, TABLET_FRAGMENT_2, TABLET_FRAGMENT_3, TABLET_FRAGMENT_4) then
        htmltext = "30471-02.htm"
      else
        htmltext = "30471-01.htm"
      end
    elseif sw2 == UNDRIAS then
      if not q:has_quest_items(player, COMPLETE_STATUE) then
        htmltext = q:has_quest_items(player, STATUE_SHILIEN_HEAD, STATUE_SHILIEN_TORSO, STATUE_SHILIEN_ARM, STATUE_SHILIEN_LEG) and "30130-02.htm" or "30130-01.htm"
      else
        htmltext = "30130-03.htm"
      end
    elseif sw2 == LOCKIRIN then
      if not q:has_quest_items(player, COMPLETE_TABLET) then
        htmltext = q:has_quest_items(player, TABLET_FRAGMENT_1, TABLET_FRAGMENT_2, TABLET_FRAGMENT_3, TABLET_FRAGMENT_4) and "30531-02.htm" or "30531-01.htm"
      else
        htmltext = "30531-03.htm"
      end
    elseif sw2 == MORGAN then
      htmltext = q:has_at_least_one_quest_item(player, CARGO_BOX_1, CARGO_BOX_2, CARGO_BOX_3, CARGO_BOX_4) and "30737-02.htm" or "30737-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  for _, info in ipairs(DROPLIST) do
    if q:has_quest_items(player, info[1]) and npc:id() == info[2] then
      local playSound = false
      if rnd(1000000) < info[4] then
        q:give_items(player, info[3], 1)
        playSound = true
      end
      if rnd(1000000) < info[6] then
        q:give_items(player, info[5], 1)
        playSound = true
      end
      if playSound then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
      break
    end
  end
end)
