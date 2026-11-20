-- The ferry between Gludin and Rune (Mobius
-- vehicles/BoatGludinRune.java). Tickets 7905 and 7904. The Rune dock is
-- never busy (see lib/boat.lua).

local boat = require("lib.boat")
local say = boat.say

-- Time: 1151 s.
local GLUDIN_TO_RUNE = {
  { -95686, 155514, -3610, 150, 800 },
  { -98112, 159040, -3610, 150, 800 },
  { -104192, 160608, -3610, 200, 1800 },
  { -109952, 159616, -3610, 250, 1800 },
  { -112768, 154784, -3610, 290, 1800 },
  { -114688, 139040, -3610, 290, 1800 },
  { -115232, 134368, -3610, 290, 1800 },
  { -113888, 121696, -3610, 290, 1800 },
  { -107808, 104928, -3610, 290, 1800 },
  { -97152, 75520, -3610, 290, 800 },
  { -85536, 67264, -3610, 290, 1800 },
  { -64640, 55840, -3610, 290, 1800 },
  { -60096, 44672, -3610, 290, 1800 },
  { -52672, 37440, -3610, 290, 1800 },
  { -46144, 33184, -3610, 290, 1800 },
  { -36096, 24928, -3610, 290, 1800 },
  { -33792, 8448, -3610, 290, 1800 },
  { -23776, 3424, -3610, 290, 1000 },
  { -12000, -1760, -3610, 290, 1000 },
  { 672, 480, -3610, 290, 1800 },
  { 15488, 200, -3610, 290, 1000 },
  { 24736, 164, -3610, 290, 1000 },
  { 32192, -1156, -3610, 290, 1000 },
  { 39200, -8032, -3610, 270, 1000 },
  { 44320, -25152, -3610, 270, 1000 },
  { 40576, -31616, -3610, 250, 800 },
  { 36819, -35315, -3610, 220, 800 },
}

local RUNE_DOCK = { 34381, -37680, -3610, 200, 800 }

-- Time: 967 s.
local RUNE_TO_GLUDIN = {
  { 32750, -39300, -3610, 150, 800 },
  { 27440, -39328, -3610, 180, 1000 },
  { 21456, -34272, -3610, 200, 1000 },
  { 6608, -29520, -3610, 250, 800 },
  { 4160, -27828, -3610, 270, 800 },
  { 2432, -25472, -3610, 270, 1000 },
  { -8000, -16272, -3610, 220, 1000 },
  { -18976, -9760, -3610, 290, 800 },
  { -23776, 3408, -3610, 290, 800 },
  { -33792, 8432, -3610, 290, 800 },
  { -36096, 24912, -3610, 290, 800 },
  { -46144, 33184, -3610, 290, 800 },
  { -52688, 37440, -3610, 290, 800 },
  { -60096, 44672, -3610, 290, 800 },
  { -64640, 55840, -3610, 290, 800 },
  { -85552, 67248, -3610, 290, 800 },
  { -97168, 85264, -3610, 290, 800 },
  { -107824, 104912, -3610, 290, 800 },
  { -102151, 135704, -3610, 290, 800 },
  { -96686, 140595, -3610, 290, 800 },
  { -95686, 147717, -3610, 250, 800 },
  { -95686, 148218, -3610, 200, 800 },
}

local GLUDIN_DOCK = { -95686, 150514, -3610, 150, 800 }

local G, R = GLUDIN_DOCK, RUNE_DOCK

boat.route {
  name = "GludinRune",
  boat = 3,
  start = { -95686, 150514, -3610, 16723 },
  dock = boat.GLUDIN_HARBOR,
  steps = {
    -- 0
    { say = say(G, R, "DEPARTURE_FOR_RUNE_HARBOR_WILL_TAKE_PLACE_IN_FIVE_MINUTES"), wait = 240000 },
    -- 1
    { say = say(G, R, "DEPARTURE_FOR_RUNE_HARBOR_WILL_TAKE_PLACE_IN_ONE_MINUTE"), wait = 40000 },
    -- 2
    { say = say(G, R, "MAKE_HASTE_WE_WILL_BE_DEPARTING_FOR_GLUDIN_HARBOR_SHORTLY_2"), wait = 20000 },
    -- 3
    {
      undock = boat.GLUDIN_HARBOR,
      say = say(G, R, "WE_ARE_NOW_DEPARTING_FOR_GLUDIN_HARBOR_HOLD_ON_AND_ENJOY_THE_RIDE"),
      horn = G,
      pay = { item = boat.TICKET_GLUDIN_TO_RUNE, count = 1, oust = { -90015, 150422, -3610 } },
      path = GLUDIN_TO_RUNE,
      wait = 250000,
    },
    -- 4
    { say = say(R, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_RUNE_HARBOR_IN_APPROXIMATELY_15_MINUTES"), wait = 300000 },
    -- 5
    { say = say(R, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_RUNE_HARBOR_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 6
    { say = say(R, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_RUNE_HARBOR_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 7: the end of the path runs the next step.
    { say = say(R, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_RUNE_HARBOR_IN_APPROXIMATELY_1_MINUTE") },
    -- 8
    {
      await = boat.await(boat.RUNE_HARBOR, R, G, "THE_FERRY_FROM_GLUDIN_HARBOR_TO_RUNE_HARBOR_HAS_BEEN_DELAYED"),
      path = { RUNE_DOCK },
    },
    -- 9
    {
      dock = boat.RUNE_HARBOR,
      say = say(R, G, "ARRIVED_AT_RUNE_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_GLUDIN_HARBOR_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = R,
      wait = 300000,
    },
    -- 10
    { say = say(R, G, "DEPARTURE_FOR_GLUDIN_HARBOR_WILL_TAKE_PLACE_IN_FIVE_MINUTES"), wait = 240000 },
    -- 11
    { say = say(R, G, "DEPARTURE_FOR_GLUDIN_HARBOR_WILL_TAKE_PLACE_IN_ONE_MINUTE"), wait = 40000 },
    -- 12
    { say = say(R, G, "MAKE_HASTE_WE_WILL_BE_DEPARTING_FOR_GLUDIN_HARBOR_SHORTLY"), wait = 20000 },
    -- 13
    {
      undock = boat.RUNE_HARBOR,
      say = say(R, G, "WE_ARE_NOW_DEPARTING_FOR_GLUDIN_HARBOR_HOLD_ON_AND_ENJOY_THE_RIDE"),
      horn = R,
      pay = { item = boat.TICKET_RUNE_TO_GLUDIN, count = 1, oust = { 34513, -38009, -3640 } },
      path = RUNE_TO_GLUDIN,
      wait = 60000,
    },
    -- 14
    { say = say(G, R, "THE_FERRY_FROM_RUNE_HARBOR_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_15_MINUTES"), wait = 300000 },
    -- 15
    { say = say(G, R, "THE_FERRY_FROM_RUNE_HARBOR_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 16
    { say = say(G, R, "THE_FERRY_FROM_RUNE_HARBOR_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 17: the end of the path runs the next step.
    { say = say(G, R, "THE_FERRY_FROM_RUNE_HARBOR_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_1_MINUTE") },
    -- 18
    {
      await = boat.await(boat.GLUDIN_HARBOR, G, R, "THE_FERRY_FROM_RUNE_HARBOR_TO_GLUDIN_HARBOR_HAS_BEEN_DELAYED"),
      path = { GLUDIN_DOCK },
    },
    -- 19
    {
      dock = boat.GLUDIN_HARBOR,
      say = say(G, R, "THE_FERRY_HAS_ARRIVED_AT_GLUDIN_HARBOR", "DEPARTURE_FOR_RUNE_HARBOR_WILL_TAKE_PLACE_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = G,
      wait = 300000,
    },
  },
}
