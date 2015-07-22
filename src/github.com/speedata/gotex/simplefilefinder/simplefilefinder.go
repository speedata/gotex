package simplefilefinder

import (
	"fmt"
	"os"
	"path/filepath"
)

var Basedir string
var filelist map[string]string

func init() {
	filelist = make(map[string]string)
}

func collectFiles(path string, info os.FileInfo, err error) error {
	if false {
		fmt.Println(path, info, err)
	}
	if filepath.Ext(path) == ".tfm" {
		filelist[info.Name()] = path
	}
	return nil
}

func Locate(filename string) string {
	if len(filelist) == 0 {
		filepath.Walk(Basedir, collectFiles)
	}
	return filelist[filename]
}
