package serverpackets

import "testing"

// Mobius SSQStatus.writeImpl, page 1: c 0xF5, c page, c period, d cycle,
// d period msg, d period end msg, c cabal, c seal, d stones, d ancient
// adena, then per cabal (Dusk, Dawn) d stone score, d festival score,
// d total, c percent.
func TestSSQStatusPage1(t *testing.T) {
	s := SSQStatus{Page: 1, Period: 1, Cycle: 7, PeriodMsg: 1176, PeriodEndMsg: 1286, Cabal: 2, Seal: 3,
		StonesTurnedIn: 15, AncientAdenaToCollect: 120,
		DuskStoneScore: 100, DuskFestivalScore: 60, DuskTotal: 160, DuskPercent: 31,
		DawnStoneScore: 400, DawnFestivalScore: 0, DawnTotal: 400, DawnPercent: 71}
	var g m3Golden
	g.c(0xf5).c(1).c(1).d(7).d(1176).d(1286).c(2).c(3).d(15).d(120)
	g.d(100).d(60).d(160).c(31)
	g.d(400).d(0).d(400).c(71)
	m3Check(t, "SSQStatus page 1", NewSSQStatusPacket(s), g.b)
}

// Page 2: h 1, c count, per festival c id, d level score, d dusk score,
// c n, S names, d dawn score, c n, S names. An empty party is one empty
// name, as Mobius splits "" into one.
func TestSSQStatusPage2(t *testing.T) {
	s := SSQStatus{Page: 2, Period: 3, Festivals: []SSQFestival{
		{ID: 1, LevelScore: 60, DuskScore: 75, DuskMembers: []string{"Ann", "Bo"}, DawnScore: 0, DawnMembers: []string{""}},
		{ID: 5, LevelScore: 150, DuskScore: 0, DuskMembers: nil, DawnScore: 300, DawnMembers: []string{"Zed"}},
	}}
	var g m3Golden
	g.c(0xf5).c(2).c(3).h(1).c(2)
	g.c(1).d(60).d(75).c(2).s("Ann").s("Bo").d(0).c(1).s("")
	g.c(5).d(150).d(0).c(0).d(300).c(1).s("Zed")
	m3Check(t, "SSQStatus page 2", NewSSQStatusPacket(s), g.b)
}

// Page 3: c 10, c 35, c count, per seal c seal, c owner, c dusk %, c dawn %.
func TestSSQStatusPage3(t *testing.T) {
	s := SSQStatus{Page: 3, Period: 0, Seals: []SSQSealVote{
		{Seal: 1, Owner: 2, DuskPercent: 0, DawnPercent: 50},
		{Seal: 2, Owner: 0, DuskPercent: 100, DawnPercent: 0},
		{Seal: 3, Owner: 1, DuskPercent: 33, DawnPercent: 25},
	}}
	var g m3Golden
	g.c(0xf5).c(3).c(0).c(10).c(35).c(3)
	g.c(1).c(2).c(0).c(50)
	g.c(2).c(0).c(100).c(0)
	g.c(3).c(1).c(33).c(25)
	m3Check(t, "SSQStatus page 3", NewSSQStatusPacket(s), g.b)
}

// Page 4: c winner, c count, per seal c seal, c owner, h message.
func TestSSQStatusPage4(t *testing.T) {
	s := SSQStatus{Page: 4, Period: 1, Winner: 2, Predictions: []SSQSealPrediction{
		{Seal: 1, Owner: 2, Msg: 1290},
		{Seal: 2, Owner: 0, Msg: 1292},
		{Seal: 3, Owner: 1, Msg: 1289},
	}}
	var g m3Golden
	g.c(0xf5).c(4).c(1).c(2).c(3)
	g.c(1).c(2).h(1290)
	g.c(2).c(0).h(1292)
	g.c(3).c(1).h(1289)
	m3Check(t, "SSQStatus page 4", NewSSQStatusPacket(s), g.b)
}

// Another page has the header only.
func TestSSQStatusOtherPage(t *testing.T) {
	m3Check(t, "SSQStatus page 0", NewSSQStatusPacket(SSQStatus{Page: 0, Period: 2, Cycle: 9}), unhex(t, "f5 00 02"))
}
