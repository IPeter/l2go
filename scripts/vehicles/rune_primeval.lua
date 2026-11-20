-- The ferry between Rune and Primeval Isle (Mobius
-- vehicles/BoatRunePrimeval.java). Tickets 8925 and 8924. The Rune dock is
-- never busy (see lib/boat.lua).

local boat = require("lib.boat")
local say, with_horn = boat.say, boat.with_horn

-- Time: 239 s.
local RUNE_TO_PRIMEVAL = {
  { 32750, -39300, -3610, 180, 800 },
  { 27440, -39328, -3610, 250, 1000 },
  { 19616, -39360, -3610, 270, 1000 },
  { 3840, -38528, -3610, 270, 1000 },
  { 1664, -37120, -3610, 270, 1000 },
  { 896, -34560, -3610, 180, 1800 },
  { 832, -31104, -3610, 180, 180 },
  { 2240, -29132, -3610, 150, 1800 },
  { 4160, -27828, -3610, 150, 1800 },
  { 5888, -27279, -3610, 150, 1800 },
  { 7000, -27279, -3610, 150, 1800 },
  { 10342, -27279, -3610, 150, 1800 },
}

-- Time: 221 s.
local PRIMEVAL_TO_RUNE = {
  { 15528, -27279, -3610, 180, 800 },
  { 22304, -29664, -3610, 290, 800 },
  { 33824, -26880, -3610, 290, 800 },
  { 38848, -21792, -3610, 240, 1200 },
  { 43424, -22080, -3610, 180, 1800 },
  { 44320, -25152, -3610, 180, 1800 },
  { 40576, -31616, -3610, 250, 800 },
  { 36819, -35315, -3610, 220, 800 },
}

local RUNE_DOCK = { 34381, -37680, -3610, 220, 800 }

-- Mobius: the last point of RUNE_TO_PRIMEVAL.
local PRIMEVAL_DOCK = RUNE_TO_PRIMEVAL[#RUNE_TO_PRIMEVAL]

local R, P = RUNE_DOCK, PRIMEVAL_DOCK

boat.route {
  name = "RunePrimeval",
  boat = 5,
  start = { 34381, -37680, -3610, 40785 },
  dock = boat.RUNE_HARBOR,
  steps = {
    -- 0: the end of the path runs the next step.
    {
      undock = boat.RUNE_HARBOR,
      say = with_horn(say(R, P, "THE_FERRY_IS_NOW_DEPARTING_RUNE_HARBOR_FOR_PRIMEVAL_ISLE"), R),
      pay = { item = boat.TICKET_RUNE_TO_PRIMEVAL, count = 1, oust = { 34513, -38009, -3640 } },
      path = RUNE_TO_PRIMEVAL,
    },
    -- 1
    {
      say = with_horn(say(P, R, "THE_FERRY_HAS_ARRIVED_AT_PRIMEVAL_ISLE", "THE_FERRY_WILL_LEAVE_FOR_RUNE_HARBOR_AFTER_ANCHORING_FOR_THREE_MINUTES"), P),
      wait = 180000,
    },
    -- 2: the end of the path runs the next step.
    {
      say = with_horn(say(P, R, "THE_FERRY_IS_NOW_DEPARTING_PRIMEVAL_ISLE_FOR_RUNE_HARBOR"), P),
      pay = { item = boat.TICKET_PRIMEVAL_TO_RUNE, count = 1, oust = { 10447, -24982, -3664 } },
      path = PRIMEVAL_TO_RUNE,
    },
    -- 3
    {
      await = boat.await(boat.RUNE_HARBOR, R, P, "THE_FERRY_FROM_PRIMEVAL_ISLE_TO_RUNE_HARBOR_HAS_BEEN_DELAYED"),
      path = { RUNE_DOCK },
    },
    -- 4
    {
      dock = boat.RUNE_HARBOR,
      say = with_horn(say(R, P, "ARRIVED_AT_RUNE_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_PRIMEVAL_ISLE_AFTER_ANCHORING_FOR_THREE_MINUTES"), R),
      wait = 180000,
    },
  },
}
