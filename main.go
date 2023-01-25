package main

import (
	"log"
	"os"

	"github.com/nlepage/gitoo/reflog"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	watcher, err := reflog.NewWatcher(wd)
	if err != nil {
		log.Fatalln(err)
	}
	defer watcher.Close()

	for entry := range watcher.Entries() {
		log.Printf("%#v\n", entry)
	}
}
