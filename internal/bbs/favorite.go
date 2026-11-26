package bbs

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// The favorites board (Mobius FavoriteBoard, table bbs_favorites).

// Column limits of bbs_favorites: a longer title or bypass is not stored
// (MySQL refuses it in strict mode).
const (
	MaxFavoriteTitle  = 50
	MaxFavoriteBypass = 127
)

// FavoriteDateFormat is how a favorite's date is shown (Mobius
// "yyyy-MM-dd HH:mm:ss").
const FavoriteDateFormat = "2006-01-02 15:04:05"

// Favorite is a bookmarked board page.
type Favorite struct {
	ID     int32
	Player int32
	Title  string
	Bypass string
	Added  time.Time
}

// Favorites are the favorites of every character. Loop-only.
type Favorites struct {
	byPlayer map[int32][]Favorite
	lastID   int32
}

// NewFavorites returns the favorites as loaded at start; new IDs follow the
// highest one (the auto-increment favId).
func NewFavorites(favs []Favorite) *Favorites {
	f := &Favorites{byPlayer: make(map[int32][]Favorite)}
	for _, fav := range favs {
		f.byPlayer[fav.Player] = append(f.byPlayer[fav.Player], fav)
		f.lastID = max(f.lastID, fav.ID)
	}
	return f
}

// Add bookmarks a page for player and returns the favorite.
func (f *Favorites) Add(player int32, title, bypass string, now time.Time) Favorite {
	f.lastID++
	fav := Favorite{ID: f.lastID, Player: player, Title: title, Bypass: bypass, Added: now}
	f.byPlayer[player] = append(f.byPlayer[player], fav)
	return fav
}

// Delete removes player's favorite with the ID and reports whether there
// was one.
func (f *Favorites) Delete(player, id int32) bool {
	list := f.byPlayer[player]
	i := slices.IndexFunc(list, func(fav Favorite) bool { return fav.ID == id })
	if i < 0 {
		return false
	}
	f.byPlayer[player] = slices.Delete(list, i, i+1)
	return true
}

// Count is the number of player's favorites.
func (f *Favorites) Count(player int32) int { return len(f.byPlayer[player]) }

// List returns player's favorites, newest first (Mobius ORDER BY favAddDate
// DESC; of two added at the same time the higher ID first).
func (f *Favorites) List(player int32) []Favorite {
	list := slices.Clone(f.byPlayer[player])
	slices.SortStableFunc(list, func(a, b Favorite) int {
		if c := b.Added.Compare(a.Added); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return list
}

// SplitBypass splits a remembered page into its title and bypass (Mobius
// "title&bypass", both trimmed); ok is false without the "&".
func SplitBypass(s string) (title, bypass string, ok bool) {
	title, bypass, ok = strings.Cut(s, "&")
	return JavaTrim(title), JavaTrim(bypass), ok
}
