-- The Innadril pleasure boat (Mobius vehicles/BoatInnadrilTour.java): a
-- round trip from the Innadril dock, without a ticket.

local boat = require("lib.boat")
local say, with_horn = boat.say, boat.with_horn

-- Time: 1867 s.
local TOUR = {
  { 105129, 226240, -3610, 150, 800 },
  { 90604, 238797, -3610, 150, 800 },
  { 74853, 237943, -3610, 150, 800 },
  { 68207, 235399, -3610, 150, 800 },
  { 63226, 230487, -3610, 150, 800 },
  { 61843, 224797, -3610, 150, 800 },
  { 61822, 203066, -3610, 150, 800 },
  { 59051, 197685, -3610, 150, 800 },
  { 54048, 195298, -3610, 150, 800 },
  { 41609, 195687, -3610, 150, 800 },
  { 35821, 200284, -3610, 150, 800 },
  { 35567, 205265, -3610, 150, 800 },
  { 35617, 222471, -3610, 150, 800 },
  { 37932, 226588, -3610, 150, 800 },
  { 42932, 229394, -3610, 150, 800 },
  { 74324, 245231, -3610, 150, 800 },
  { 81872, 250314, -3610, 150, 800 },
  { 101692, 249882, -3610, 150, 800 },
  { 107907, 256073, -3610, 150, 800 },
  { 112317, 257133, -3610, 150, 800 },
  { 126273, 255313, -3610, 150, 800 },
  { 128067, 250961, -3610, 150, 800 },
  { 128520, 238249, -3610, 150, 800 },
  { 126428, 235072, -3610, 150, 800 },
  { 121843, 234656, -3610, 150, 800 },
  { 120096, 234268, -3610, 150, 800 },
  { 118572, 233046, -3610, 150, 800 },
  { 117671, 228951, -3610, 150, 800 },
  { 115936, 226540, -3610, 150, 800 },
  { 113628, 226240, -3610, 150, 800 },
  { 111300, 226240, -3610, 150, 800 },
  { 111264, 226240, -3610, 150, 800 },
}

-- Mobius: the last point of TOUR.
local DOCK = TOUR[#TOUR]

boat.route {
  name = "InnadrilTour",
  boat = 4,
  start = { 111264, 226240, -3610, 32768 },
  steps = {
    -- 0
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_LEAVE_IN_FIVE_MINUTES"), wait = 240000 },
    -- 1
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_LEAVE_IN_ONE_MINUTE"), wait = 40000 },
    -- 2
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_BE_LEAVING_SOON"), wait = 20000 },
    -- 3
    {
      say = with_horn(say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_IS_LEAVING"), DOCK),
      pay = { item = 0, count = 1, oust = { 107092, 219098, -3952 } },
      path = TOUR,
      wait = 650000,
    },
    -- 4
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_ARRIVE_IN_APPROXIMATELY_20_MINUTES"), wait = 300000 },
    -- 5
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_ARRIVE_IN_APPROXIMATELY_15_MINUTES"), wait = 300000 },
    -- 6
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_ARRIVE_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 7
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_ARRIVE_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 8: the end of the path runs the next step.
    { say = say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_WILL_ARRIVE_IN_APPROXIMATELY_1_MINUTE") },
    -- 9
    {
      say = with_horn(say(DOCK, DOCK, "THE_INNADRIL_PLEASURE_BOAT_HAS_ARRIVED_IT_WILL_ANCHOR_FOR_TEN_MINUTES"), DOCK),
      wait = 300000,
    },
  },
}
