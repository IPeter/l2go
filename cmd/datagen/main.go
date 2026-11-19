// Command datagen converts the parts of the L2J Mobius Interlude datapack
// that L2Go needs into the files of the data/ folder, and generates the Go
// enums of internal/data from the Mobius Java sources.
//
// Usage:
//
//	go run ./cmd/datagen [-mobius dir] [-out dir] [domain ...]
//	go run ./cmd/datagen -enums [-mobius dir] [-out dir] [domain ...]
//
// The first form writes data files (default folder: data). The second
// writes Go source files (default folder: internal/data, whose go:generate
// line runs it). Without domain names every domain of the mode runs; -list
// prints them.
//
// # Domains
//
// Each part of the data lives in its own file and registers itself from
// init:
//
//	func init() { registerDomain("players", genPlayers) }
//	func init() { registerEnums("items", genItemEnums) }
//
// A domain reads the Mobius project through options.src and adds its files
// to options.out (see output). A run writes all of its files or none: the
// domains fill an in-memory output, the data files are then loaded with
// internal/data (the domains of the same name, see data.LoadDomains), and
// only then written. A domain fails on anything it does not expect in the
// Mobius data, so a datapack change is noticed instead of silently dropped.
//
// The output is sorted and does not depend on file order or time, so
// running datagen again on the same input gives the same files.
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// Default output folders, relative to the repository root.
const (
	defaultDataOut  = "data"
	defaultEnumsOut = "internal/data"
)

// options is what a domain works with.
type options struct {
	// src is the Mobius Interlude project folder.
	src fs.FS
	// out collects the generated files.
	out *output
	// log receives warnings about the Mobius data.
	log io.Writer
}

// warnf reports something in the Mobius data that the domain worked
// around, such as a known inconsistency.
func (o options) warnf(format string, args ...any) {
	fmt.Fprintf(o.log, "warning: "+format+"\n", args...)
}

// domain is one registered generator.
type domain struct {
	name string
	// enums is set for generators of Go source files (-enums mode).
	enums bool
	run   func(o options) error
}

// registry holds the domains registered by init functions.
var registry []domain

// registerDomain registers a generator of data files. Call it from init.
// The name is also the internal/data domain that loads and checks the
// files, if there is one.
func registerDomain(name string, run func(o options) error) {
	register(domain{name: name, run: run})
}

// registerEnums registers a generator of Go source files for the -enums
// mode. Call it from init.
func registerEnums(name string, run func(o options) error) {
	register(domain{name: name, enums: true, run: run})
}

func register(d domain) {
	if d.name == "" || d.run == nil {
		panic("datagen: domain without a name or a run function")
	}
	for _, r := range registry {
		if r.name == d.name && r.enums == d.enums {
			panic("datagen: domain " + d.name + " is registered twice")
		}
	}
	registry = append(registry, d)
}

// domains returns the registered domains of a mode, sorted by name.
func domains(enums bool) []domain {
	var ds []domain
	for _, d := range registry {
		if d.enums == enums {
			ds = append(ds, d)
		}
	}
	slices.SortFunc(ds, func(a, b domain) int { return strings.Compare(a.name, b.name) })
	return ds
}

// selectDomains returns the named domains of a mode, or all of them if
// names is empty.
func selectDomains(enums bool, names []string) ([]domain, error) {
	all := domains(enums)
	if len(names) == 0 {
		return all, nil
	}
	var picked []domain
	for _, name := range names {
		i := slices.IndexFunc(all, func(d domain) bool { return d.name == name })
		if i < 0 {
			return nil, fmt.Errorf("unknown domain %q (have: %s)", name, domainNames(all))
		}
		if !slices.ContainsFunc(picked, func(d domain) bool { return d.name == name }) {
			picked = append(picked, all[i])
		}
	}
	slices.SortFunc(picked, func(a, b domain) int { return strings.Compare(a.name, b.name) })
	return picked, nil
}

func domainNames(ds []domain) string {
	names := make([]string, len(ds))
	for i, d := range ds {
		names[i] = d.name
	}
	return strings.Join(names, ", ")
}

func main() {
	mobius := flag.String("mobius", "L2J_Mobius/L2J_Mobius_CT_0_Interlude", "L2J Mobius Interlude project folder")
	out := flag.String("out", "", "output folder (default "+defaultDataOut+", or "+defaultEnumsOut+" with -enums)")
	enums := flag.Bool("enums", false, "generate the Go enum files instead of the data files")
	list := flag.Bool("list", false, "print the domains of the mode and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: datagen [-enums] [-mobius dir] [-out dir] [domain ...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *list {
		for _, d := range domains(*enums) {
			fmt.Println(d.name)
		}
		return
	}
	dir := *out
	if dir == "" {
		dir = defaultDataOut
		if *enums {
			dir = defaultEnumsOut
		}
	}
	if err := run(*mobius, dir, *enums, flag.Args(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "datagen:", err)
		os.Exit(1)
	}
}

// run generates the selected domains from mobiusDir into outDir and reports
// each file it writes to log.
func run(mobiusDir, outDir string, enums bool, names []string, log io.Writer) error {
	files, err := generate(os.DirFS(mobiusDir), enums, names, log)
	if err != nil {
		return err
	}
	return writeFiles(outDir, files, log)
}

// generate runs the selected domains of a mode (all if names is empty) on
// the Mobius project in src and returns their files, keyed by
// slash-separated path. Data files are loaded with internal/data before
// they are returned, so a run never writes data the server refuses.
func generate(src fs.FS, enums bool, names []string, log io.Writer) (map[string][]byte, error) {
	selected, err := selectDomains(enums, names)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no domains to run")
	}
	out := newOutput()
	o := options{src: src, out: out, log: log}
	for _, d := range selected {
		if err := d.run(o); err != nil {
			return nil, fmt.Errorf("%s: %w", d.name, err)
		}
	}
	if !enums {
		if err := check(out.files, selected); err != nil {
			return nil, err
		}
	}
	return out.files, nil
}

// check loads the generated data files with the internal/data domains of
// the same names, which also checks the references between them.
func check(files map[string][]byte, selected []domain) error {
	known := data.DomainNames()
	var names []string
	for _, d := range selected {
		if slices.Contains(known, d.name) {
			names = append(names, d.name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	fsys := make(fstest.MapFS, len(files))
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b, Mode: 0o644}
	}
	if _, err := data.LoadDomains(fsys, names...); err != nil {
		return fmt.Errorf("the generated data does not load: %w", err)
	}
	return nil
}
