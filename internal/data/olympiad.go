package data

import (
	"fmt"
	"io/fs"
	"os"
)

// The Olympiad's stadiums (Plan 14, M11h): the 22 arenas of Mobius
// OlympiadManager.STADIUMS, a Java table that cmd/datagen copies from
// input/olympiad.json after checking it against the Java source (decision
// X6). The stadium zones are the OlympiadStadiumZone zones of the world
// data.

// OlympiadStadiumsFile is the path of the stadium file, relative to the
// data folder.
const OlympiadStadiumsFile = "olympiad/stadiums.json"

// OlympiadStadium is the centre of an arena: the nobles stand 900 units
// east and west of it, the spectators at it.
type OlympiadStadium struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

// LoadOlympiadStadiums reads and checks the stadium file in dir.
func LoadOlympiadStadiums(dir string) ([]OlympiadStadium, error) {
	s, err := LoadOlympiadStadiumsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return s, nil
}

// LoadOlympiadStadiumsFS reads and checks the stadium file in fsys.
func LoadOlympiadStadiumsFS(fsys fs.FS) ([]OlympiadStadium, error) {
	var list []OlympiadStadium
	if err := readJSON(fsys, OlympiadStadiumsFile, &list); err != nil {
		return nil, err
	}
	if err := CheckOlympiadStadiums(list); err != nil {
		return nil, err
	}
	return list, nil
}

// CheckOlympiadStadiums checks the stadiums: at least one, none at 0,0 and
// none listed twice.
func CheckOlympiadStadiums(list []OlympiadStadium) error {
	var p problems
	if len(list) == 0 {
		p.addf("%s: no stadium", OlympiadStadiumsFile)
	}
	seen := make(map[OlympiadStadium]bool, len(list))
	for i, s := range list {
		switch {
		case s.X == 0 && s.Y == 0:
			p.addf("%s: stadium %d is at 0,0", OlympiadStadiumsFile, i)
		case seen[s]:
			p.addf("%s: stadium %d is listed twice", OlympiadStadiumsFile, i)
		}
		seen[s] = true
	}
	return p.err()
}
