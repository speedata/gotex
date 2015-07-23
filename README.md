# gotex
Playground for TeX-related tools written in Go

# dvitype
The first one in the long series. Currently a direct translation from the Pascal source, not Goish in any terms. And not yet tested well enough.

## How to build

You need a Go compiler which you can get from https://golang.org/. Then type

    $ export GOPATH=$PWD
    $ go get github.com/speedata/gotex/dvitype/dvitype

This installs the dvitype binary in `$GOPATH/bin`. To use it run

    $ bin/dvitype -basedir /opt/texlive2014/texmf-dist/fonts/tfm/ test.dvi

You need to change the `-basedir` option of course. The default `basedir` setting is the current directory. The current file finder searches recursively from the given base dir.

