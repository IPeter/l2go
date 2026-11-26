package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"

	"github.com/IPeter/l2go/internal/data"
)

// The olympiad domain writes the Olympiad's stadiums (Plan 14, M11h),
// loaded by data.LoadOlympiadStadiums: a table that exists only in Java,
// kept in input/olympiad.json and checked against the source it names
// (decision X6).
func init() { registerDomain("olympiad", genOlympiad) }

// olympiadManagerFile is the Java source of the stadium table.
const olympiadManagerFile = javaRoot + "mechanics/olympiad/OlympiadManager.java"

// olympiadInput is the stadium table as input/olympiad.json writes it.
//
//go:embed input/olympiad.json
var olympiadInput []byte

// olympiadTables is the content of input/olympiad.json.
type olympiadTables struct {
	Comment  string                 `json:"comment"`
	Source   []string               `json:"source"`
	Stadiums []data.OlympiadStadium `json:"stadiums"`
}

// genOlympiad adds olympiad/stadiums.json.
func genOlympiad(o options) error {
	stadiums, err := readOlympiadStadiums(o.src, olympiadInput)
	if err != nil {
		return err
	}
	if err := data.CheckOlympiadStadiums(stadiums); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.OlympiadStadiumsFile, stadiums); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "olympiad: %d stadiums\n", len(stadiums))
	return nil
}

// javaStadiumRe matches a stadium of the STADIUMS table.
var javaStadiumRe = regexp.MustCompile(`new\s+OlympiadStadium\(\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`)

// readOlympiadStadiums reads the input table and checks that it is the
// Java table, in order.
func readOlympiadStadiums(src fs.FS, input []byte) ([]data.OlympiadStadium, error) {
	var in olympiadTables
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("input/olympiad.json: %w", err)
	}
	if want := []string{olympiadManagerFile}; !slices.Equal(in.Source, want) {
		return nil, fmt.Errorf("input/olympiad.json: source %q, want %q", in.Source, want)
	}
	b, err := fs.ReadFile(src, olympiadManagerFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", olympiadManagerFile, err)
	}
	var java []data.OlympiadStadium
	for _, m := range javaStadiumRe.FindAllStringSubmatch(stripJavaComments(string(b)), -1) {
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		z, _ := strconv.Atoi(m[3])
		java = append(java, data.OlympiadStadium{X: x, Y: y, Z: z})
	}
	if len(java) == 0 {
		return nil, fmt.Errorf("%s: no stadium", olympiadManagerFile)
	}
	if !slices.Equal(java, in.Stadiums) {
		return nil, fmt.Errorf("%s: stadiums %v, input/olympiad.json has %v", olympiadManagerFile, java, in.Stadiums)
	}
	return in.Stadiums, nil
}
