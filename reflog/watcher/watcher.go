package watcher

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/nlepage/git-oo/reflog"
)

const logsDirName = "logs"

type Watcher struct {
	fswatcher *fsnotify.Watcher
	events    chan *reflog.Event
	lastLine  map[string]string
	logsDir   string
}

func New(gitDir string) (*Watcher, error) {
	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	events := make(chan *reflog.Event)

	w := &Watcher{
		fswatcher: fswatcher,
		events:    events,
		lastLine:  make(map[string]string),
		logsDir:   filepath.Join(gitDir, logsDirName),
	}

	if err := w.addDir(w.logsDir, false); err != nil {
		return nil, err
	}

	go w.watch()

	return w, nil
}

func (w *Watcher) Entries() <-chan *reflog.Event {
	return w.events
}

func (w *Watcher) Close() error {
	defer close(w.events)
	return w.fswatcher.Close()
}

func (w *Watcher) addDir(dir string, sendEvents bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return w.fswatcher.Add(path)
		}

		return w.readLastLines(path, sendEvents)
	})
}

func (w *Watcher) readLastLines(file string, sendEvents bool) error {
	lines := make([]string, 0, 10)

	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	s := bufio.NewScanner(r)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return err
	}

	if len(lines) == 0 {
		return nil
	}

	prevLastLine := w.lastLine[file]
	prevIndex := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == prevLastLine {
			prevIndex = i
			break
		}
	}

	w.lastLine[file] = lines[len(lines)-1]

	if sendEvents {
		reference, err := w.reference(file)
		if err != nil {
			return err
		}

		for _, line := range lines[prevIndex+1:] {
			event := &reflog.Event{
				Reference: reference,
				Type:      reflog.NewEntry,
			}
			if err := event.Entry.Parse(line); err != nil {
				return err
			}
			w.events <- event
		}
	}

	return nil
}

func (w *Watcher) watch() {
	for {
		select {
		case evt, ok := <-w.fswatcher.Events:
			if !ok {
				return
			}
			log.Printf("%s operation on file %s", evt.Op, evt.Name)

			switch {
			case evt.Has(fsnotify.Create):
				stat, err := os.Stat(evt.Name)
				if err != nil {
					log.Println(err)
					continue
				}
				if !stat.IsDir() {
					continue
				}
				if err := w.addDir(evt.Name, true); err != nil {
					log.Println(err)
				}
			case evt.Has(fsnotify.Write):
				if err := w.readLastLines(evt.Name, true); err != nil {
					log.Println(err)
				}
			case evt.Has(fsnotify.Remove):
				reference, err := w.reference(evt.Name)
				if err != nil {
					log.Println(err)
					continue
				}
				w.events <- &reflog.Event{
					Reference: reference,
					Type:      reflog.Remove,
				}
			}
		case err, ok := <-w.fswatcher.Errors:
			if !ok {
				return
			}
			log.Println(err)
		}
	}
}

func (w *Watcher) reference(file string) (string, error) {
	return filepath.Rel(w.logsDir, file)
}
