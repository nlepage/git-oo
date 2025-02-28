package reflog

import (
	"bufio"
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const logsDirName = "logs"

type watcher struct {
	ctx       context.Context
	fswatcher *fsnotify.Watcher
	events    chan *Event
	lastLine  map[string]string
	logsDir   string
}

func Watch(ctx context.Context, gitDir string) (<-chan *Event, error) {
	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	events := make(chan *Event)

	w := &watcher{
		ctx:       ctx,
		fswatcher: fswatcher,
		events:    events,
		lastLine:  make(map[string]string),
		logsDir:   filepath.Join(gitDir, logsDirName),
	}

	if err := w.addDir(w.logsDir, false); err != nil {
		return nil, err
	}

	go w.watch()

	return events, nil
}

func (w *watcher) close() error {
	defer close(w.events)
	return w.fswatcher.Close()
}

func (w *watcher) addDir(dir string, sendEvents bool) error {
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

func (w *watcher) readLastLines(file string, sendEvents bool) error {
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
			event := &Event{
				Reference: reference,
				Type:      NewEntry,
			}
			if err := event.Entry.Parse(line); err != nil {
				return err
			}
			w.events <- event
		}
	}

	return nil
}

func (w *watcher) watch() {
	for {
		select {
		case evt := <-w.fswatcher.Events:
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
				w.events <- &Event{
					Reference: reference,
					Type:      Remove,
				}
			}
		case err := <-w.fswatcher.Errors:
			log.Println(err)
		case <-w.ctx.Done():
			if err := w.close(); err != nil {
				log.Println(err)
			}
			return
		}
	}
}

func (w *watcher) reference(file string) (string, error) {
	return filepath.Rel(w.logsDir, file)
}
