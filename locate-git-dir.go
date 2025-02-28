package gitoo

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

func LocateGitDir(path string) (string, error) {
	for {
		_, err := os.Stat(filepath.Join(path, git.GitDirName))
		if err == nil {
			return filepath.Join(path, git.GitDirName), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		dir := filepath.Dir(path)
		if dir == path {
			return "", errors.New("not in a git directory")
		}
		path = dir
	}
}
