package sevensigns

// The Record of the Seven Signs (Mobius SSQStatus): what each page shows,
// computed as the packet computes it. The game turns a Record into the
// SSQStatus packet.

// RecordPage1 is the first page: the period and the player's and the
// cabals' scores.
type RecordPage1 struct {
	Cycle                                int
	PeriodMsg, PeriodEndMsg              int
	Cabal                                Cabal
	Seal                                 Seal
	StonesTurnedIn, AncientAdenaToCollect int
	DuskStone, DuskFestival, DuskTotal   int
	DuskPercent                          int
	DawnStone, DawnFestival, DawnTotal   int
	DawnPercent                          int
}

// Page1 computes the first page for the character id.
func (s *State) Page1(id int32) RecordPage1 {
	r := RecordPage1{Cycle: s.Cycle, Cabal: s.PlayerCabal(id), Seal: s.PlayerSeal(id),
		StonesTurnedIn: s.PlayerStoneContrib(id), AncientAdenaToCollect: s.PlayerAdenaCollect(id)}
	switch s.Period {
	case PeriodRecruiting:
		r.PeriodMsg = MsgInitialPeriod
	case PeriodCompetition:
		r.PeriodMsg = MsgQuestEventPeriod
	case PeriodResults:
		r.PeriodMsg = MsgCalculatingPeriod
	case PeriodSealValidation:
		r.PeriodMsg = MsgSealValidationPeriod
	}
	if s.Period.IsMajor() {
		r.PeriodEndMsg = MsgUntilNextMonday
	} else {
		r.PeriodEndMsg = MsgUntilToday
	}
	total := s.DuskStoneScore + s.DawnStoneScore
	if total != 0 {
		r.DuskStone = JavaRound(float32(s.DuskStoneScore) / float32(total) * 500)
		r.DawnStone = JavaRound(float32(s.DawnStoneScore) / float32(total) * 500)
	}
	r.DuskFestival, r.DawnFestival = s.DuskFestivalScore, s.DawnFestivalScore
	r.DuskTotal, r.DawnTotal = s.CurrentScore(CabalDusk), s.CurrentScore(CabalDawn)
	if all := r.DuskTotal + r.DawnTotal; all != 0 {
		r.DawnPercent = JavaRound(float32(r.DawnTotal) / float32(all) * 100)
		r.DuskPercent = JavaRound(float32(r.DuskTotal) / float32(all) * 100)
	}
	return r
}

// SealVote is a seal on the third page: its owner and the share of each
// cabal's members that chose it.
type SealVote struct {
	Seal                     Seal
	Owner                    Cabal
	DuskPercent, DawnPercent int
}

// Page3 computes the third page (the members' choices).
func (s *State) Page3() []SealVote {
	dawnMembers, duskMembers := s.TotalMembers(CabalDawn), s.TotalMembers(CabalDusk)
	pct := func(part, total int) int {
		if total == 0 {
			return 0
		}
		return JavaRound(float32(part) / float32(total) * 100)
	}
	out := make([]SealVote, 0, len(Seals))
	for _, seal := range Seals {
		out = append(out, SealVote{Seal: seal, Owner: s.SealOwner(seal),
			DuskPercent: pct(s.SealProportion(seal, CabalDusk), duskMembers),
			DawnPercent: pct(s.SealProportion(seal, CabalDawn), dawnMembers)})
	}
	return out
}

// SealPrediction is a seal on the fourth page: who would own it now, and
// why.
type SealPrediction struct {
	Seal  Seal
	Owner Cabal
	Msg   int
}

// Page4 computes the fourth page: the predicted winner and the predicted
// owner of each seal (SSQStatus's copy of calcNewSealOwners, with its
// messages).
func (s *State) Page4() (Cabal, []SealPrediction) {
	winner := s.HighestCabal()
	out := make([]SealPrediction, 0, len(Seals))
	for _, seal := range Seals {
		dawn, dusk := s.SealPercents(seal)
		owner := s.SealOwner(seal)
		p := SealPrediction{Seal: seal}
		switch owner {
		case CabalNull:
			switch winner {
			case CabalNull:
				p.Msg = MsgTieNobodyAwarded
			case CabalDawn, CabalDusk:
				pctW := dawn
				if winner == CabalDusk {
					pctW = dusk
				}
				if pctW >= 35 {
					p.Owner, p.Msg = winner, MsgNotOwnedAnd35Percent
				} else {
					p.Msg = MsgNotOwnedAndBelow35
				}
			}
		default:
			mine, theirs := dawn, dusk
			rival := CabalDusk
			if owner == CabalDusk {
				mine, theirs, rival = dusk, dawn, CabalDawn
			}
			switch winner {
			case CabalNull:
				if mine >= 10 {
					p.Owner, p.Msg = owner, MsgOwnedAnd10Percent
				} else {
					p.Msg = MsgTieNobodyAwarded
				}
			case owner:
				if mine >= 10 {
					p.Owner, p.Msg = owner, MsgOwnedAnd10Percent
				} else {
					p.Msg = MsgOwnedButBelow10Percent
				}
			case rival:
				switch {
				case theirs >= 35:
					p.Owner, p.Msg = rival, MsgNotOwnedAnd35Percent
				case mine >= 10:
					p.Owner, p.Msg = owner, MsgOwnedAnd10Percent
				default:
					p.Msg = MsgOwnedButBelow10Percent
				}
			}
		}
		out = append(out, p)
	}
	return winner, out
}

// MustLeaveDungeon is the rule of teleLosingCabalFromDungeons (cabal is the
// character's stored cabal name, hasRecord whether it has a record) at a
// period change: in the results and seal validation periods, everyone who
// is not of the winner (by short name, so nobody on a tie); otherwise
// everyone without a record or with a cabal (Mobius's test, which is the
// reverse of the enter-world one).
func MustLeaveDungeon(period Period, cabal string, hasRecord bool, winner Cabal) bool {
	if period == PeriodSealValidation || period == PeriodResults {
		return !hasRecord || cabal != winner.ShortName()
	}
	return !hasRecord || cabal != ""
}

// MustLeaveDungeonOnEnter is Player.onPlayerEnter's rule: in the results
// and seal validation periods, everyone not of the leading cabal; otherwise
// everyone without a cabal.
func MustLeaveDungeonOnEnter(period Period, cabal, winner Cabal) bool {
	if period == PeriodSealValidation || period == PeriodResults {
		return cabal != winner
	}
	return cabal == CabalNull
}

// DungeonAccess is DungeonGatekeeper's rule for the necropolises (seal
// avarice) and the catacombs (seal gnosis): whether the character may enter,
// and the message a refused one gets (0 for none). open is
// AltOpenNecropolis or AltOpenCatacumbs.
func DungeonAccess(open, sealValidation bool, winner, playerCabal, sealOwner Cabal) (bool, int) {
	if open {
		return true, 0
	}
	if !sealValidation {
		return playerCabal != CabalNull, 0
	}
	switch {
	case winner == CabalDawn && (playerCabal != CabalDawn || sealOwner != CabalDawn):
		return false, MsgOnlyDawn
	case winner == CabalDusk && (playerCabal != CabalDusk || sealOwner != CabalDusk):
		return false, MsgOnlyDusk
	case winner == CabalNull && playerCabal != CabalNull:
		return true, 0
	case playerCabal == CabalNull:
		return false, 0
	}
	return true, 0
}

// StrifeSkill is the skill the Seal of Strife gives a character in the seal
// validation period (giveCPMult and EnterWorld): The Victor of War for the
// owner's members, The Vanquished of War for the others with a cabal, 0
// for none.
func StrifeSkill(strifeOwner, playerCabal Cabal) int {
	switch {
	case playerCabal == CabalNull || strifeOwner == CabalNull:
		return 0
	case playerCabal == strifeOwner:
		return VictorOfWarSkill
	}
	return VanquishedOfWarSkill
}

// DawnTicket and RookieTicket are checkIsDawnPostingTicket and
// checkIsRookiePostingTicket: the mercenary tickets only Dawn's seal allows,
// and the ones Dusk's still allows.
func DawnTicket(id int) bool {
	return (id > 6114 && id < 6175) || (id > 6801 && id < 6812) || (id > 7997 && id < 8008) || (id > 7940 && id < 7951) ||
		(id > 6294 && id < 6307) || (id > 6831 && id < 6834) || (id > 8027 && id < 8030) || (id > 7970 && id < 7973)
}

// RookieTicket see DawnTicket.
func RookieTicket(id int) bool {
	return (id > 6174 && id < 6295) || (id > 6811 && id < 6832) || (id > 7950 && id < 7971) || (id > 8007 && id < 8028)
}
