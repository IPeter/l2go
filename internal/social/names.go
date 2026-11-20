package social

import (
	"regexp"
	"strings"
	"unicode"
)

// NameCheck is the result of a clan, unit or alliance name check.
type NameCheck int

// The name check results.
const (
	// NameOK: the name may be used (uniqueness is the caller's check).
	NameOK NameCheck = iota
	// NameInvalid: not letters and digits only, under 2 characters (clan
	// and unit names), or not matching the template. Clans and units answer
	// CLAN_NAME_IS_INVALID, alliances INCORRECT_ALLIANCE_NAME_PLEASE_TRY_AGAIN.
	NameInvalid
	// NameBadLength: too long (clan and unit: over 16, answered with
	// CLAN_NAME_S_LENGTH_IS_INCORRECT) or, for alliances, outside 2–16
	// (INCORRECT_LENGTH_FOR_AN_ALLIANCE_NAME).
	NameBadLength
)

// The name length limits (ClanTable.createClan, Clan.createAlly).
const (
	MinNameLength = 2
	MaxNameLength = 16
)

// CompileNameTemplate compiles ClanNameTemplate (Server.ini) so that it
// must match the whole name, as Java's Matcher.matches does. Mobius's
// default is ".*". Go's RE2 syntax lacks some Java constructs
// (back-references, look-arounds); such templates fail to compile.
func CompileNameTemplate(template string) (*regexp.Regexp, error) {
	return regexp.Compile(`^(?:` + template + `)$`)
}

// alnumLength is StringUtil.isAlphaNumeric with the name's Java length:
// every UTF-16 unit a letter or digit (Character.isLetterOrDigit). A
// character outside the BMP is two surrogate units, neither a letter, so
// it fails; so do invalid UTF-8 bytes (decoded as U+FFFD, a symbol). ok is
// false for an empty name.
func alnumLength(name string) (n int, ok bool) {
	if name == "" {
		return 0, false
	}
	for _, r := range name {
		if r > 0xFFFF || (!unicode.IsLetter(r) && !unicode.IsDigit(r)) {
			return 0, false
		}
		n++
	}
	return n, true
}

// CheckClanName checks a clan or unit name as the village master does
// (VillageMaster create_clan / createSubPledge / renameSubPledge, then
// ClanTable.createClan): the template (nil skips it), letters and digits
// only, at least 2 → NameInvalid; more than 16 → NameBadLength.
func CheckClanName(name string, template *regexp.Regexp) NameCheck {
	if template != nil && !template.MatchString(name) {
		return NameInvalid
	}
	n, ok := alnumLength(name)
	if !ok || n < MinNameLength {
		return NameInvalid
	}
	if n > MaxNameLength {
		return NameBadLength
	}
	return NameOK
}

// CheckAllyName checks an alliance name as Clan.createAlly does: letters
// and digits only → NameInvalid, then 2–16 characters → NameBadLength.
// Mobius applies no template to alliance names.
func CheckAllyName(name string) NameCheck {
	n, ok := alnumLength(name)
	if !ok {
		return NameInvalid
	}
	if n < MinNameLength || n > MaxNameLength {
		return NameBadLength
	}
	return NameOK
}

// NameKey is the key two clan or alliance names collide on: Mobius compares
// them ignoring case (getClanByName, isAllyExists).
func NameKey(name string) string { return strings.ToLower(name) }
