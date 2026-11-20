package gameserver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"unicode"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
)

// startingClasses are the classes a new character may have, in the order
// the creation screen lists them (Mobius NewCharacter): human, elf, dark
// elf and orc fighter and mystic, then dwarf fighter.
var startingClasses = []int{0, 10, 18, 25, 31, 38, 44, 49, 53}

// Character creation limits (Mobius CharacterCreate).
const (
	maxNameLength      = 16 // UTF-16 code units, as Java counts
	maxFace            = 2
	maxHairStyleMale   = 4
	maxHairStyleFemale = 6
	maxHairColor       = 3
)

// startingTemplates returns the templates of the starting classes that d
// has, in startingClasses order.
func startingTemplates(d *data.Data) []data.Template {
	var ts []data.Template
	for _, id := range startingClasses {
		if t, ok := d.Template(id); ok {
			ts = append(ts, t)
		}
	}
	return ts
}

// startingTemplate returns the template of classID if it is a starting
// class.
func (s *Server) startingTemplate(classID int32) (data.Template, bool) {
	for _, t := range s.templates {
		if t.ClassID == int(classID) {
			return t, true
		}
	}
	return data.Template{}, false
}

// checkName applies the protocol's name rules: 1-16 characters, letters
// and digits only. Like Java's Character.isLetterOrDigit on UTF-16 code
// units, characters outside the Basic Multilingual Plane are refused.
func checkName(name string) (serverpackets.CharCreateFailReason, bool) {
	n := 0
	for _, r := range name {
		n += max(utf16.RuneLen(r), 1)
	}
	if n == 0 || n > maxNameLength {
		return serverpackets.CharCreateFailNameTooLong, false
	}
	for _, r := range name {
		if utf16.RuneLen(r) != 1 || (!unicode.IsLetter(r) && !unicode.IsDigit(r)) {
			return serverpackets.CharCreateFailIncorrectName, false
		}
	}
	return 0, true
}

// checkAppearance applies the appearance ranges. It returns a description
// of the first value out of range, or "" if all are valid.
func checkAppearance(req clientpackets.CharacterCreate) string {
	maxHairStyle := int32(maxHairStyleMale)
	if req.Female() {
		maxHairStyle = maxHairStyleFemale
	}
	switch {
	case req.Face < 0 || req.Face > maxFace:
		return fmt.Sprintf("face %d is not in 0-%d", req.Face, maxFace)
	case req.HairStyle < 0 || req.HairStyle > maxHairStyle:
		return fmt.Sprintf("hair style %d is not in 0-%d", req.HairStyle, maxHairStyle)
	case req.HairColor < 0 || req.HairColor > maxHairColor:
		return fmt.Sprintf("hair color %d is not in 0-%d", req.HairColor, maxHairColor)
	}
	return ""
}

// createCharacter checks a CharacterCreate request, runs the script hook
// and saves the new character. It returns false and the reason to send if
// the character is refused or cannot be saved.
//
// The checks run in Mobius order: name, appearance, class, characters per
// account, name taken. The script hook runs after them, so scripts only see
// characters the server would accept.
func (s *Server) createCharacter(ctx context.Context, c *client, req clientpackets.CharacterCreate) (serverpackets.CharCreateFailReason, bool) {
	logger := c.log().With("name", req.Name, "class", req.ClassID)
	if reason, ok := checkName(req.Name); !ok {
		logger.Info("character name refused", "reason", reason)
		return reason, false
	}
	if problem := checkAppearance(req); problem != "" {
		s.hackAttempts.Add(1)
		logger.Warn("character creation refused: possible client hack", "problem", problem)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	tmpl, ok := s.startingTemplate(req.ClassID)
	if !ok {
		s.hackAttempts.Add(1)
		logger.Warn("character creation refused: not a starting class")
		return serverpackets.CharCreateFailCreationFailed, false
	}

	// Count, name check and save happen under one lock per account, so two
	// connections of the same account cannot both pass the limit.
	account := c.Account()
	unlock := s.accountLocks.lock(account)
	defer unlock()

	n, err := s.chars.CountByAccount(ctx, account)
	if err != nil {
		logger.Error("character creation failed: cannot count characters", "err", err)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	if limit := s.gs.MaxCharactersPerAccount; limit > 0 && n >= limit {
		logger.Info("character creation refused: too many characters", "count", n, "limit", limit)
		return serverpackets.CharCreateFailTooManyCharacters, false
	}
	exists, err := s.chars.NameExists(ctx, req.Name)
	if err != nil {
		logger.Error("character creation failed: cannot check the name", "err", err)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	if exists {
		logger.Info("character creation refused: name exists")
		return serverpackets.CharCreateFailNameExists, false
	}

	if reason, ok := s.allowCharacterCreate(ctx, c, req, tmpl); !ok {
		return reason, false
	}

	ch, err := s.newCharacter(ctx, account, req, tmpl)
	if err != nil {
		logger.Error("character creation failed", "err", err)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	// The account's first character is its newbie (Mobius
	// Player.create: accountCharNumber == 0); AltNewCharAlwaysIsNewbie is
	// applied by the game (Player.IsNewbie).
	ch.Newbie = n == 0
	if err := s.chars.Create(ctx, ch); err != nil {
		if errors.Is(err, storage.ErrExists) {
			logger.Info("character creation refused: name exists")
			return serverpackets.CharCreateFailNameExists, false
		}
		logger.Error("character creation failed: cannot save", "err", err)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	logger.Info("character created", "objectId", ch.ObjectID, "race", ch.Race,
		"x", ch.X, "y", ch.Y, "z", ch.Z, "items", len(ch.Items))
	return 0, true
}

// allowCharacterCreate runs the character_create script hook on the world
// loop, which owns the scripts engine. It returns false and the code to
// send if a script rejects the character, or if the hook could not run (the
// loop or the engine stopped). Handler errors are logged by the engine and
// do not block creation. A game without a scripts engine allows everything.
func (s *Server) allowCharacterCreate(ctx context.Context, c *client, req clientpackets.CharacterCreate, tmpl data.Template) (serverpackets.CharCreateFailReason, bool) {
	ev := scripting.CharacterCreate{
		Account:   c.Account(),
		Name:      req.Name,
		Race:      tmpl.Race,
		Sex:       int(sexOf(req)),
		ClassID:   tmpl.ClassID,
		HairStyle: int(req.HairStyle),
		HairColor: int(req.HairColor),
		Face:      int(req.Face),
	}.Event()
	// res may be read only when OnLoop returns nil: after an error the
	// function may still run later.
	var res scripting.Result
	err := s.game.OnLoop(ctx, func() {
		if eng := s.game.Scripts(); eng != nil {
			res = eng.Invoke(ctx, ev)
		}
	})
	switch {
	case err != nil:
	case ctx.Err() != nil: // handlers may have been cut short
		err = ctx.Err()
	case errors.Is(res.Err, scripting.ErrStopped):
		err = res.Err
	}
	if err != nil {
		c.log().Warn("character creation refused: the script hook did not run", "name", req.Name, "err", err)
		return serverpackets.CharCreateFailCreationFailed, false
	}
	if !res.Rejected {
		return 0, true
	}
	code, ok := serverpackets.CharCreateFailReasonByName(res.Reason)
	if !ok {
		c.log().Warn("unknown character creation reason, using creation_failed", "reason", res.Reason)
		code = serverpackets.CharCreateFailCreationFailed
	}
	c.log().Info("character creation rejected by a script", "name", req.Name, "reason", res.Reason)
	return code, false
}

// sexOf returns the stored sex: 0 male, 1 female.
func sexOf(req clientpackets.CharacterCreate) int32 {
	if req.Female() {
		return 1
	}
	return 0
}

// vitalsGame is a game that knows the HP, MP and CP of new characters
// (*game.Server).
type vitalsGame interface {
	NewCharacterVitals(classID, level int, items []storage.Item) (game.CharacterVitals, error)
}

var _ vitalsGame = (*game.Server)(nil)

// skillsGame is a game that knows the skills of new characters
// (*game.Server).
type skillsGame interface {
	NewCharacterSkills(classID int) []storage.CharacterSkill
}

var _ skillsGame = (*game.Server)(nil)

// startingSkills returns the skills a new character of the class starts
// with (the level-1 skills of its tree, Mobius CharacterCreate), none when
// the game has no skills.
func (s *Server) startingSkills(classID int) []storage.CharacterSkill {
	if g, ok := s.game.(skillsGame); ok {
		return g.NewCharacterSkills(classID)
	}
	return nil
}

// shortcutGame is a game that knows the shortcut bar of new characters
// (*game.Server).
type shortcutGame interface {
	NewCharacterShortcuts(c *storage.Character) []storage.Shortcut
}

var _ shortcutGame = (*game.Server)(nil)

// startingShortcuts returns the shortcut bar of a new character with its
// items and skills (Mobius InitialShortcutData.registerAllShortcuts), none
// when the game has no initial shortcuts.
func (s *Server) startingShortcuts(c *storage.Character) []storage.Shortcut {
	if g, ok := s.game.(shortcutGame); ok {
		return g.NewCharacterShortcuts(c)
	}
	return nil
}

// questGame is the optional part of the game that gives a new character its
// starting quests (Mobius startTutorialQuest).
type questGame interface {
	NewCharacterQuests() []storage.Quest
}

var _ questGame = (*game.Server)(nil)

// startingQuests returns the quests a new character starts with, none when
// the game has no quests.
func (s *Server) startingQuests() []storage.Quest {
	if g, ok := s.game.(questGame); ok {
		return g.NewCharacterQuests()
	}
	return nil
}

// startingVitals returns the HP, MP and CP of a new level 1 character of
// class tmpl with items, the worn ones in their slots: full HP and MP and
// CP 0, from the game's stats (with the CON and MEN bonuses, as Mobius
// CharacterCreate sets them). A game that cannot compute them leaves the
// template's level 1 values.
func (s *Server) startingVitals(tmpl data.Template, items []storage.Item) game.CharacterVitals {
	if g, ok := s.game.(vitalsGame); ok {
		v, err := g.NewCharacterVitals(tmpl.ClassID, 1, items)
		if err == nil {
			return v
		}
		s.logger.Warn("cannot compute the starting HP, MP and CP, using the template's", "class", tmpl.ClassID, "err", err)
	}
	l := tmpl.Level1
	return game.CharacterVitals{CurHP: l.HP, MaxHP: l.HP, CurMP: l.MP, MaxMP: l.MP, MaxCP: l.CP}
}

// newCharacter builds a level 1 character of class tmpl with its starting
// items, at a random creation point of the template. HP and MP are full and
// CP is 0, as in Mobius (startingVitals).
func (s *Server) newCharacter(ctx context.Context, account string, req clientpackets.CharacterCreate, tmpl data.Template) (*storage.Character, error) {
	start := s.data.InitialItems(tmpl.ClassID)
	ids, err := s.ids.NextIDs(ctx, 1+len(start))
	if err != nil {
		return nil, fmt.Errorf("object IDs: %w", err)
	}
	if len(tmpl.CreationPoints) == 0 {
		return nil, fmt.Errorf("class %d has no creation point", tmpl.ClassID)
	}
	loc := tmpl.CreationPoints[rand.IntN(len(tmpl.CreationPoints))] //nolint:gosec // game randomness, not security

	items := make([]storage.Item, len(start))
	for i, e := range start {
		items[i] = storage.Item{
			ObjectID: ids[1+i],
			ItemID:   int32(e.ItemID), //nolint:gosec // item IDs fit in int32
			Count:    e.Count,
			Mana:     storage.NoMana,
		}
		// A shadow item starts with its full mana (Mobius
		// ItemManager.createItem).
		if t, ok := s.data.Item(e.ItemID); ok && t.Duration > 0 {
			items[i].Mana = int32(min(t.Duration, math.MaxInt32)) //nolint:gosec // clamped
		}
	}
	doll := newPaperdoll(items)
	for i, e := range start {
		if !e.Equipped {
			continue
		}
		it, ok := s.data.Item(e.ItemID)
		if !ok || !doll.equip(i, it) {
			s.logger.Warn("starting item cannot be equipped, keeping it in the inventory",
				"class", tmpl.ClassID, "item", e.ItemID, "bodyPart", it.BodyPart)
		}
	}

	vitals := s.startingVitals(tmpl, items)
	c := &storage.Character{
		ObjectID:    ids[0],
		Account:     account,
		Name:        req.Name,
		Race:        int32(tmpl.Race), //nolint:gosec // small game data values
		Sex:         sexOf(req),
		ClassID:     int32(tmpl.ClassID), //nolint:gosec // small game data values
		BaseClassID: int32(tmpl.ClassID), //nolint:gosec // small game data values
		HairStyle:   req.HairStyle,
		HairColor:   req.HairColor,
		Face:        req.Face,
		Level:       1,
		CurHP:       vitals.CurHP,
		MaxHP:       vitals.MaxHP,
		CurMP:       vitals.CurMP,
		MaxMP:       vitals.MaxMP,
		CurCP:       vitals.CurCP,
		MaxCP:       vitals.MaxCP,
		X:           int32(loc.X), //nolint:gosec // world coordinates fit in int32
		Y:           int32(loc.Y), //nolint:gosec // world coordinates fit in int32
		Z:           int32(loc.Z), //nolint:gosec // world coordinates fit in int32
		AccessLevel: storage.AccessLevelPlayer,
		Items:       items,
		Skills:      s.startingSkills(tmpl.ClassID),
		Quests:      s.startingQuests(),
		// Mobius stores lastAccess at creation, so the new character is
		// the active one of the list until another is played.
		LastAccess: s.now(),
	}
	// The shortcuts point at the items and need the skills (Mobius
	// registerAllShortcuts comes last).
	c.Shortcuts = s.startingShortcuts(c)
	return c, nil
}

// keyedMutex is a set of mutexes by key, created on demand and dropped when
// nobody holds or waits for them. The zero value is ready to use.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int // holders and waiters
}

// lock locks the mutex of key and returns the function that unlocks it.
func (k *keyedMutex) lock(key string) (unlock func()) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[string]*keyedLock)
	}
	l := k.locks[key]
	if l == nil {
		l = &keyedLock{}
		k.locks[key] = l
	}
	l.refs++
	k.mu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
