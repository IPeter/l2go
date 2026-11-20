-- Translated by questport from quests/Q00108_JumbleTumbleDiamondFuss (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(108, "Q00108_JumbleTumbleDiamondFuss", "Jumble, Tumble, Diamond Fuss")

-- NPCs
local GOUPH = 30523
local REEP = 30516
local MURDOC = 30521
local AIRY = 30522
local BRUNON = 30526
local MARON = 30529
local TOROCCO = 30555
-- Monsters
local GOBLIN_BRIGAND_LEADER = 20323
local GOBLIN_BRIGAND_LIEUTENANT = 20324
local BLADE_BAT = 20480
-- Items
local GOUPH_CONTRACT = 1559
local REEP_CONTRACT = 1560
local ELVEN_WINE = 1561
local BRUNON_DICE = 1562
local BRUNON_CONTRACT = 1563
local AQUAMARINE = 1564
local CHRYSOBERYL = 1565
local GEM_BOX = 1566
local COAL_PIECE = 1567
local BRUNON_LETTER = 1568
local BERRY_TART = 1569
local BAT_DIAGRAM = 1570
local STAR_DIAMOND = 1571
-- Rewards
local SILVERSMITH_HAMMER = 1511
local SPIRITSHOT_FOR_BEGINNERS = 5790
local SOULSHOT_FOR_BEGINNERS = 5789
local ECHO_BATTLE = 4412
local ECHO_LOVE = 4413
local ECHO_SOLITUDE = 4414
local ECHO_FEAST = 4415
local ECHO_CELEBRATION = 4416
local LESSER_HEALING_POTION = 1060
-- Misc
local MAX_GEM_COUNT = 10
local GOBLIN_DROP_CHANCES = {}

GOBLIN_DROP_CHANCES[GOBLIN_BRIGAND_LEADER] = 0.8
GOBLIN_DROP_CHANCES[GOBLIN_BRIGAND_LIEUTENANT] = 0.6

q:register_quest_items(GOUPH_CONTRACT, REEP_CONTRACT, ELVEN_WINE, BRUNON_DICE, BRUNON_CONTRACT, AQUAMARINE, CHRYSOBERYL, GEM_BOX, COAL_PIECE, BRUNON_LETTER, BERRY_TART, BAT_DIAGRAM, STAR_DIAMOND)
q:add_start_npc(GOUPH)
q:add_talk_id(GOUPH, REEP, MURDOC, AIRY, BRUNON, MARON, TOROCCO)
q:add_kill_id(GOBLIN_BRIGAND_LEADER, GOBLIN_BRIGAND_LIEUTENANT, BLADE_BAT)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  local htmltext = event
  if st == nil then
    return htmltext
  end
  if event == "30523-03.htm" then
    st:start_quest()
    q:give_items(player, GOUPH_CONTRACT, 1)
  elseif event == "30555-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, REEP_CONTRACT, 1)
    q:give_items(player, ELVEN_WINE, 1)
  elseif event == "30526-02.htm" then
    st:set_cond(5, true)
    q:take_items(player, BRUNON_DICE, 1)
    q:give_items(player, BRUNON_CONTRACT, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DWARF then
      htmltext = "30523-00.htm"
    elseif player:level() < 10 then
      htmltext = "30523-01.htm"
    else
      htmltext = "30523-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GOUPH then
      if cond == 1 then
        htmltext = "30523-04.htm"
      elseif cond > 1 and cond < 7 then
        htmltext = "30523-05.htm"
      elseif cond == 7 then
        htmltext = "30523-06.htm"
        st:set_cond(8, true)
        q:take_items(player, GEM_BOX, 1)
        q:give_items(player, COAL_PIECE, 1)
      elseif cond > 7 and cond < 12 then
        htmltext = "30523-07.htm"
      elseif cond == 12 then
        htmltext = "30523-08.htm"
        q:take_items(player, STAR_DIAMOND, -1)
        q:give_items(player, SILVERSMITH_HAMMER, 1)
        q:give_items(player, LESSER_HEALING_POTION, 100)
        -- Give newbie reward if player is eligible
        if player:is_newbie() then
          local newPlayerRewardsReceived = player:var_int("NEWBIE_SHOTS_RECEIVED", 0)
          if newPlayerRewardsReceived < 2 then
            st:show_question_mark(26)
            if player:is_mage_class() then
              st:play_tutorial_voice("tutorial_voice_027")
              q:give_items(player, SPIRITSHOT_FOR_BEGINNERS, 3000)
              newPlayerRewardsReceived = newPlayerRewardsReceived + 1
              player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
            else
              st:play_tutorial_voice("tutorial_voice_026")
              q:give_items(player, SOULSHOT_FOR_BEGINNERS, 7000)
              newPlayerRewardsReceived = newPlayerRewardsReceived + 1
              player:set_var("NEWBIE_SHOTS_RECEIVED", newPlayerRewardsReceived)
            end
          end
        end
        q:give_items(player, ECHO_BATTLE, 10)
        q:give_items(player, ECHO_LOVE, 10)
        q:give_items(player, ECHO_SOLITUDE, 10)
        q:give_items(player, ECHO_FEAST, 10)
        q:give_items(player, ECHO_CELEBRATION, 10)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == REEP then
      if cond == 1 then
        htmltext = "30516-01.htm"
        st:set_cond(2, true)
        q:take_items(player, GOUPH_CONTRACT, 1)
        q:give_items(player, REEP_CONTRACT, 1)
      elseif cond > 1 then
        htmltext = "30516-02.htm"
      end
    elseif sw2 == TOROCCO then
      if cond == 2 then
        htmltext = "30555-01.htm"
      elseif cond == 3 then
        htmltext = "30555-03.htm"
      elseif cond == 7 then
        htmltext = "30555-04.htm"
      elseif cond > 7 then
        htmltext = "30555-05.htm"
      end
    elseif sw2 == MARON then
      if cond == 3 then
        htmltext = "30529-01.htm"
        st:set_cond(4, true)
        q:take_items(player, ELVEN_WINE, 1)
        q:give_items(player, BRUNON_DICE, 1)
      elseif cond == 4 then
        htmltext = "30529-02.htm"
      elseif cond > 4 then
        htmltext = "30529-03.htm"
      end
    elseif sw2 == BRUNON then
      if cond == 4 then
        htmltext = "30526-01.htm"
      elseif cond == 5 then
        htmltext = "30526-03.htm"
      elseif cond == 6 then
        htmltext = "30526-04.htm"
        st:set_cond(7, true)
        q:take_items(player, BRUNON_CONTRACT, 1)
        q:take_items(player, AQUAMARINE, -1)
        q:take_items(player, CHRYSOBERYL, -1)
        q:give_items(player, GEM_BOX, 1)
      elseif cond == 7 then
        htmltext = "30526-05.htm"
      elseif cond == 8 then
        htmltext = "30526-06.htm"
        st:set_cond(9, true)
        q:take_items(player, COAL_PIECE, 1)
        q:give_items(player, BRUNON_LETTER, 1)
      elseif cond == 9 then
        htmltext = "30526-07.htm"
      elseif cond > 9 then
        htmltext = "30526-08.htm"
      end
    elseif sw2 == MURDOC then
      if cond == 9 then
        htmltext = "30521-01.htm"
        st:set_cond(10, true)
        q:take_items(player, BRUNON_LETTER, 1)
        q:give_items(player, BERRY_TART, 1)
      elseif cond == 10 then
        htmltext = "30521-02.htm"
      elseif cond > 10 then
        htmltext = "30521-03.htm"
      end
    elseif sw2 == AIRY then
      if cond == 10 then
        htmltext = "30522-01.htm"
        st:set_cond(11, true)
        q:take_items(player, BERRY_TART, 1)
        q:give_items(player, BAT_DIAGRAM, 1)
      elseif cond == 11 then
        htmltext = rnd.bool() and "30522-02.htm" or "30522-04.htm"
      elseif cond == 12 then
        htmltext = "30522-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, killer, true) then
    local sw = npc:id()
    if sw == GOBLIN_BRIGAND_LEADER or sw == GOBLIN_BRIGAND_LIEUTENANT then
      repeat
        if qs:is_cond(5) and q:has_quest_items(killer, BRUNON_CONTRACT) then
          local dropChance = GOBLIN_DROP_CHANCES[npc:id()]
          local playSound = false
          if q:give_item_randomly(killer, npc, AQUAMARINE, 1, 1, MAX_GEM_COUNT, dropChance, false) then
            if q:get_quest_items_count(killer, CHRYSOBERYL) >= MAX_GEM_COUNT then
              qs:set_cond(6, true)
              break
            end
            playSound = true
          end
          if q:give_item_randomly(killer, npc, CHRYSOBERYL, 1, 1, MAX_GEM_COUNT, dropChance, false) then
            if q:get_quest_items_count(killer, AQUAMARINE) >= MAX_GEM_COUNT then
              qs:set_cond(6, true)
              break
            end
            playSound = true
          end
          if playSound then
            q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      until true
    elseif sw == BLADE_BAT then
      if qs:is_cond(11) and q:has_quest_items(killer, BAT_DIAGRAM) and q:give_item_randomly(killer, npc, STAR_DIAMOND, 1, 1, 1, 0.2, true) then
        q:take_items(killer, BAT_DIAGRAM, -1)
        qs:set_cond(12)
      end
    end
  end
end)
