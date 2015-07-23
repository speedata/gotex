package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/speedata/gotex/dvitype"
)

func main() {

	// -dpi=REAL              set resolution to REAL pixels per inch; default 300.0
	// -magnification=NUMBER  override existing magnification with NUMBER
	// -max-pages=NUMBER      process NUMBER pages; default one million
	// -output-level=NUMBER   verbosity level, from 0 to 4; default 4
	// -page-start=PAGE-SPEC  start at PAGE-SPEC, for example `2' or `5.*.-2'
	// -show-opcodes          show numeric opcodes (in decimal)
	// -help                  display this help and exit
	// -version               output version information and exit

	curdir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	var outmode = flag.Int("output-level", 4, "verbosity level, from 0 to 4; default 4")
	var pagespec = flag.String("page-start", "*", "start at PAGE-SPEC, for example `2' or `5.*.-2'")
	var maxpages = flag.Int("max-pages", 1000000, "process NUMBER pages; default one million")
	var basedir = flag.String("basedir", curdir, "Set the root directory with TFM files")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "dvitype: Need exactly one file argument.")
		fmt.Fprintln(os.Stderr, "Try `dvitype --help' for more information.")
		os.Exit(1)
	}
	dvifile, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	d := dvitype.New(dvifile)
	d.OutMode = *outmode
	d.PageSpec = *pagespec
	d.MaxPages = *maxpages
	d.Basedir = *basedir
	d.Run()
}
