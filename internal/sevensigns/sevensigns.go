// Package sevensigns holds the rules of the Seven Signs (Plan 14 M11l):
// Mobius's SevenSigns engine (the cabals, the seal stones, the scores, the
// seals and their owners, the weekly periods), the Festival of Darkness's
// scores and bonuses, and the Dimensional Rift's rooms and costs. It is pure:
// the game (internal/game) runs it on the world loop, sends the packets and
// saves the state.
package sevensigns

import (
	"math"
	"sort"
)

// Cabal is a side of the competition (Mobius SevenSigns.CABAL_*).
type Cabal int

// The cabals.
const (
	CabalNull Cabal = 0
	CabalDusk Cabal = 1
	CabalDawn Cabal = 2
)

// Seal is one of the three seals (Mobius SevenSigns.SEAL_*).
type Seal int

// The seals.
const (
	SealNull    Seal = 0
	SealAvarice Seal = 1
	SealGnosis  Seal = 2
	SealStrife  Seal = 3
)

// Seals are the three seals in Mobius's order.
var Seals = []Seal{SealAvarice, SealGnosis, SealStrife}

// Period is a period of the weekly cycle (Mobius SevenSigns.PERIOD_*).
type Period int

// The periods.
const (
	PeriodRecruiting     Period = 0
	PeriodCompetition    Period = 1
	PeriodResults        Period = 2
	PeriodSealValidation Period = 3
)

// Items and costs (Mobius SevenSigns).
const (
	RecordItemID = 5707 // RECORD_SEVEN_SIGNS_ID
	RecordCost   = 500  // RECORD_SEVEN_SIGNS_COST

	StoneBlueID  = 6360
	StoneGreenID = 6361
	StoneRedID   = 6362

	StoneBlueValue  = 3
	StoneGreenValue = 5
	StoneRedValue   = 10

	BlueContribPoints  = 3
	GreenContribPoints = 5
	RedContribPoints   = 10

	// AncientAdenaID is Inventory.ANCIENT_ADENA_ID.
	AncientAdenaID = 5575
)

// NPC IDs (Mobius SevenSigns).
const (
	OratorNpcID          = 31094
	PreacherNpcID        = 31093
	MammonMerchantID     = 31113
	MammonBlacksmithID   = 31126
	MammonMarketeerID    = 31092
	LilithNpcID          = 25283
	AnakimNpcID          = 25286
	CrestOfDawnID        = 31170
	CrestOfDuskID        = 31171
	VictorOfWarSkill     = 5074 // CommonSkill.THE_VICTOR_OF_WAR
	VanquishedOfWarSkill = 5075 // CommonSkill.THE_VANQUISHED_OF_WAR
)

// ShortName is getCabalShortName: "dawn", "dusk" or "No Cabal".
func (c Cabal) ShortName() string {
	switch c {
	case CabalDawn:
		return "dawn"
	case CabalDusk:
		return "dusk"
	}
	return "No Cabal"
}

// Name is getCabalName.
func (c Cabal) Name() string {
	switch c {
	case CabalDawn:
		return "Lords of Dawn"
	case CabalDusk:
		return "Revolutionaries of Dusk"
	}
	return "No Cabal"
}

// CabalOf returns the cabal of a stored short name ("dawn", "dusk"; any
// other is none), as getPlayerCabal reads it, ignoring case.
func CabalOf(short string) Cabal {
	switch {
	case equalFold(short, "dawn"):
		return CabalDawn
	case equalFold(short, "dusk"):
		return CabalDusk
	}
	return CabalNull
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// Name is getSealName: "Seal of Avarice", or "Avarice" when short.
func (s Seal) Name(short bool) string {
	name := ""
	if !short {
		name = "Seal of "
	}
	switch s {
	case SealAvarice:
		name += "Avarice"
	case SealGnosis:
		name += "Gnosis"
	case SealStrife:
		name += "Strife"
	}
	return name
}

// Name is getCurrentPeriodName.
func (p Period) Name() string {
	switch p {
	case PeriodRecruiting:
		return "Quest Event Initialization"
	case PeriodCompetition:
		return "Competition (Quest Event)"
	case PeriodResults:
		return "Quest Event Results"
	case PeriodSealValidation:
		return "Seal Validation"
	}
	return ""
}

// ContributionScore is calcContributionScore.
func ContributionScore(blue, green, red int) int {
	return blue*BlueContribPoints + green*GreenContribPoints + red*RedContribPoints
}

// AncientAdenaReward is calcAncientAdenaReward.
func AncientAdenaReward(blue, green, red int) int {
	return blue*StoneBlueValue + green*StoneGreenValue + red*StoneRedValue
}

// JavaRound is Java's Math.round(float): floor(f + 0.5) in float32.
func JavaRound(f float32) int {
	return int(math.Floor(float64(f + 0.5)))
}

// Player is a character's Seven Signs record (a Mobius seven_signs row).
// Cabal keeps the stored short name: "" after a cycle's reset.
type Player struct {
	CharID                             int32
	Cabal                              string
	Seal                               Seal
	RedStones, GreenStones, BlueStones int
	AncientAdena                       int
	Contribution                       int
}

// Status is the engine's state besides the players (Mobius
// seven_signs_status, without the festival's columns).
type Status struct {
	Cycle          int
	Period         Period
	PreviousWinner Cabal

	DawnStoneScore, DuskStoneScore       float64
	DawnFestivalScore, DuskFestivalScore int

	// SealOwners, DawnSealTotals and DuskSealTotals are indexed by Seal
	// (index 0 unused).
	SealOwners     [4]Cabal
	DawnSealTotals [4]int
	DuskSealTotals [4]int
}

// NewStatus is a new database's status (seven_signs_status.sql): cycle 1,
// the competition period.
func NewStatus() Status { return Status{Cycle: 1, Period: PeriodCompetition} }

// State is the Seven Signs engine's state: the status and every player who
// ever joined a cabal. Not safe for concurrent use.
type State struct {
	Status
	players map[int32]*Player
}

// NewState returns a state with the status and the players.
func NewState(st Status, players []Player) *State {
	s := &State{Status: st, players: make(map[int32]*Player, len(players))}
	for i := range players {
		p := players[i]
		s.players[p.CharID] = &p
	}
	return s
}

// Player returns a copy of a character's record and whether it has one.
func (s *State) Player(id int32) (Player, bool) {
	p, ok := s.players[id]
	if !ok {
		return Player{}, false
	}
	return *p, true
}

// Players returns copies of all records sorted by character ID.
func (s *State) Players() []Player {
	out := make([]Player, 0, len(s.players))
	for _, p := range s.players {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CharID < out[j].CharID })
	return out
}

// IsCompetition, IsSealValidation and IsResults are the period tests.
func (s *State) IsCompetition() bool    { return s.Period == PeriodCompetition }
func (s *State) IsSealValidation() bool { return s.Period == PeriodSealValidation }
func (s *State) IsResults() bool        { return s.Period == PeriodResults }

// stoneShare is the 500-point share of a stone score:
// Math.round((float) (stone / (total == 0 ? 1 : total)) * 500).
func stoneShare(stone, total float64) int {
	if float32(total) == 0 {
		total = 1
	}
	return JavaRound(float32(stone/total) * 500)
}

// CurrentScore is getCurrentScore: the cabal's share of 500 points by its
// stone score, plus its festival score.
func (s *State) CurrentScore(c Cabal) int {
	total := s.DawnStoneScore + s.DuskStoneScore
	switch c {
	case CabalDawn:
		return stoneShare(s.DawnStoneScore, total) + s.DawnFestivalScore
	case CabalDusk:
		return stoneShare(s.DuskStoneScore, total) + s.DuskFestivalScore
	}
	return 0
}

// StoneScore is getCurrentStoneScore.
func (s *State) StoneScore(c Cabal) float64 {
	switch c {
	case CabalDawn:
		return s.DawnStoneScore
	case CabalDusk:
		return s.DuskStoneScore
	}
	return 0
}

// FestivalScore is getCurrentFestivalScore.
func (s *State) FestivalScore(c Cabal) int {
	switch c {
	case CabalDawn:
		return s.DawnFestivalScore
	case CabalDusk:
		return s.DuskFestivalScore
	}
	return 0
}

// HighestCabal is getCabalHighestScore: the leading cabal, none on a tie.
func (s *State) HighestCabal() Cabal {
	dusk, dawn := s.CurrentScore(CabalDusk), s.CurrentScore(CabalDawn)
	switch {
	case dusk == dawn:
		return CabalNull
	case dusk > dawn:
		return CabalDusk
	}
	return CabalDawn
}

// SealOwner is getSealOwner.
func (s *State) SealOwner(seal Seal) Cabal {
	if seal < SealAvarice || seal > SealStrife {
		return CabalNull
	}
	return s.SealOwners[seal]
}

// SealProportion is getSealProportion: how many members of the cabal chose
// the seal.
func (s *State) SealProportion(seal Seal, c Cabal) int {
	if seal < SealAvarice || seal > SealStrife {
		return 0
	}
	switch c {
	case CabalDusk:
		return s.DuskSealTotals[seal]
	case CabalDawn:
		return s.DawnSealTotals[seal]
	}
	return 0
}

// TotalMembers is getTotalMembers.
func (s *State) TotalMembers(c Cabal) int {
	name := c.ShortName()
	n := 0
	for _, p := range s.players {
		if p.Cabal == name {
			n++
		}
	}
	return n
}

// PlayerCabal is getPlayerCabal.
func (s *State) PlayerCabal(id int32) Cabal {
	if p, ok := s.players[id]; ok {
		return CabalOf(p.Cabal)
	}
	return CabalNull
}

// PlayerSeal is getPlayerSeal.
func (s *State) PlayerSeal(id int32) Seal {
	if p, ok := s.players[id]; ok {
		return p.Seal
	}
	return SealNull
}

// PlayerStoneContrib is getPlayerStoneContrib: the stones turned in.
func (s *State) PlayerStoneContrib(id int32) int {
	if p, ok := s.players[id]; ok {
		return p.RedStones + p.GreenStones + p.BlueStones
	}
	return 0
}

// PlayerContribScore is getPlayerContribScore.
func (s *State) PlayerContribScore(id int32) int {
	if p, ok := s.players[id]; ok {
		return p.Contribution
	}
	return 0
}

// PlayerAdenaCollect is getPlayerAdenaCollect.
func (s *State) PlayerAdenaCollect(id int32) int {
	if p, ok := s.players[id]; ok {
		return p.AncientAdena
	}
	return 0
}

// SetPlayerInfo is setPlayerInfo: the character joins the cabal for the
// seal (a new record, or the old one re-registered), and the seal's count
// of the cabal grows (Dawn when the record says dawn, Dusk otherwise).
func (s *State) SetPlayerInfo(id int32, c Cabal, seal Seal) {
	p, ok := s.players[id]
	if !ok {
		p = &Player{CharID: id}
		s.players[id] = p
	}
	p.Cabal = c.ShortName()
	p.Seal = seal
	if seal < SealAvarice || seal > SealStrife {
		return
	}
	if p.Cabal == "dawn" {
		s.DawnSealTotals[seal]++
	} else {
		s.DuskSealTotals[seal]++
	}
}

// TakeAncientAdenaReward is getAncientAdenaReward(objectId, true): the
// ancient adena the character may claim; its stones and ancient adena are
// cleared. 0 without a record.
func (s *State) TakeAncientAdenaReward(id int32) int {
	p, ok := s.players[id]
	if !ok {
		return 0
	}
	n := p.AncientAdena
	p.RedStones, p.GreenStones, p.BlueStones, p.AncientAdena = 0, 0, 0, 0
	return n
}

// AddStoneContrib is addPlayerStoneContrib: the stones count for the
// character's cabal; it returns the contribution they are worth, or -1 (and
// changes nothing) when the contribution score would pass max. A character
// without a record gets -1 (Mobius would fail).
func (s *State) AddStoneContrib(id int32, blue, green, red, max int) int {
	p, ok := s.players[id]
	if !ok {
		return -1
	}
	score := ContributionScore(blue, green, red)
	total := p.Contribution + score
	if total > max {
		return -1
	}
	p.RedStones += red
	p.GreenStones += green
	p.BlueStones += blue
	p.AncientAdena += AncientAdenaReward(blue, green, red)
	p.Contribution = total
	switch CabalOf(p.Cabal) {
	case CabalDawn:
		s.DawnStoneScore += float64(score)
	case CabalDusk:
		s.DuskStoneScore += float64(score)
	}
	return score
}

// AddFestivalScore is addFestivalScore: the cabal gains amount, and the
// rival loses it when it has that much.
func (s *State) AddFestivalScore(c Cabal, amount int) {
	if c == CabalDusk {
		s.DuskFestivalScore += amount
		if s.DawnFestivalScore >= amount {
			s.DawnFestivalScore -= amount
		}
		return
	}
	s.DawnFestivalScore += amount
	if s.DuskFestivalScore >= amount {
		s.DuskFestivalScore -= amount
	}
}

// ResetPlayers is resetPlayerData: every record loses its cabal, seal and
// contribution score (stones and ancient adena stay).
func (s *State) ResetPlayers() {
	for _, p := range s.players {
		p.Cabal = ""
		p.Seal = SealNull
		p.Contribution = 0
	}
}

// ResetSeals is resetSeals.
func (s *State) ResetSeals() {
	s.DawnSealTotals = [4]int{}
	s.DuskSealTotals = [4]int{}
}

// percentOf is Math.round(((float) part / total) * 100) with total at least
// 1 (calcNewSealOwners).
func percentOf(part, total int) int {
	if total == 0 {
		total = 1
	}
	return JavaRound(float32(part) / float32(total) * 100)
}

// NewSealOwner is the rule of calcNewSealOwners for one seal: a seal not
// owned, or owned by the loser, goes to the winner with 35% of its members;
// the owner keeps it with 10% (on a tie, or when it won, or when the winner
// has less than 35%).
func NewSealOwner(prev, winner Cabal, dawnPercent, duskPercent int) Cabal {
	switch prev {
	case CabalNull:
		switch winner {
		case CabalDawn:
			if dawnPercent >= 35 {
				return CabalDawn
			}
		case CabalDusk:
			if duskPercent >= 35 {
				return CabalDusk
			}
		}
	case CabalDawn:
		switch winner {
		case CabalNull, CabalDawn:
			if dawnPercent >= 10 {
				return CabalDawn
			}
		case CabalDusk:
			if duskPercent >= 35 {
				return CabalDusk
			}
			if dawnPercent >= 10 {
				return CabalDawn
			}
		}
	case CabalDusk:
		switch winner {
		case CabalNull, CabalDusk:
			if duskPercent >= 10 {
				return CabalDusk
			}
		case CabalDawn:
			if dawnPercent >= 35 {
				return CabalDawn
			}
			if duskPercent >= 10 {
				return CabalDusk
			}
		}
	}
	return CabalNull
}

// SealPercents returns the share of each cabal's members that chose the
// seal, as calcNewSealOwners rounds it.
func (s *State) SealPercents(seal Seal) (dawn, dusk int) {
	return percentOf(s.SealProportion(seal, CabalDawn), s.TotalMembers(CabalDawn)),
		percentOf(s.SealProportion(seal, CabalDusk), s.TotalMembers(CabalDusk))
}

// CalcNewSealOwners is calcNewSealOwners: each seal's owner by the winner
// and the members' choices. It returns the new owner of each seal, in
// Mobius's order (avarice, gnosis, strife).
func (s *State) CalcNewSealOwners() [4]Cabal {
	winner := s.HighestCabal()
	for _, seal := range Seals {
		dawn, dusk := s.SealPercents(seal)
		s.SealOwners[seal] = NewSealOwner(s.SealOwners[seal], winner, dawn, dusk)
	}
	return s.SealOwners
}

// SealObtainedMsg is the system message calcNewSealOwners sends everyone
// for a seal's new owner, 0 for none.
func SealObtainedMsg(seal Seal, owner Cabal) int {
	base := 0
	switch owner {
	case CabalDawn:
		base = MsgDawnObtainedAvarice
	case CabalDusk:
		base = MsgDuskObtainedAvarice
	default:
		return 0
	}
	switch seal {
	case SealAvarice:
		return base
	case SealGnosis:
		return base + 1
	case SealStrife:
		return base + 2
	}
	return 0
}

// MaxTax is CastleManager.validateTaxes's limit for the Seal of Strife's
// owner: 25 % for Dawn, 5 % for Dusk, 15 % otherwise.
func MaxTax(strifeOwner Cabal) int {
	switch strifeOwner {
	case CabalDusk:
		return 5
	case CabalDawn:
		return 25
	}
	return 15
}
