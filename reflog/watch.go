package reflog

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const logsDirName = "logs"

type Watcher struct {
	fswatcher *fsnotify.Watcher
	entries   chan *Entry
	lastLine  map[string]string
	logsDir   string
}

func NewWatcher(gitDir string) (*Watcher, error) {
	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	entries := make(chan *Entry)

	w := &Watcher{
		fswatcher: fswatcher,
		entries:   entries,
		lastLine:  make(map[string]string),
		logsDir:   filepath.Join(gitDir, logsDirName),
	}

	if err := w.addDir(w.logsDir, false); err != nil {
		return nil, err
	}

	go w.watch()

	return w, nil
}

func (w *Watcher) Entries() <-chan *Entry {
	return w.entries
}

func (w *Watcher) Close() error {
	defer close(w.entries)
	return w.fswatcher.Close()
}

func (w *Watcher) addDir(dir string, sendEntries bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return w.fswatcher.Add(path)
		}

		return w.readLastLines(path, sendEntries)
	})
}

func (w *Watcher) readLastLines(file string, sendEntries bool) error {
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
	prevIndex := 0
	for i := len(lines) - 1; i > 0; i-- {
		if lines[i] == prevLastLine {
			prevIndex = i
			break
		}
	}

	w.lastLine[file] = lines[len(lines)-1]

	if sendEntries {
		reference, err := filepath.Rel(w.logsDir, file)
		if err != nil {
			return err
		}

		for _, line := range lines[prevIndex+1:] {
			entry, err := parseEntry(reference, line)
			if err != nil {
				return err
			}
			w.entries <- entry
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
			}
		case err, ok := <-w.fswatcher.Errors:
			if !ok {
				return
			}
			log.Println(err)
		}
	}
}
