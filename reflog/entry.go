package reflog

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Reference string
	OldHash   string
	NewHash   string
	Author    Author
	Time      time.Time
	Action    string
	Modifier  string
	Message   string
}

type Author struct {
	Name  string
	Email string
}

func (e *Entry) Parse(s string) error {
	r := bufio.NewReader(strings.NewReader(s))

	s, err := r.ReadString(' ')
	if err != nil {
		return err
	}
	oldHash := s[:len(s)-1]

	s, err = r.ReadString(' ')
	if err != nil {
		return err
	}
	newHash := s[:len(s)-1]

	s, err = r.ReadString('<')
	if err != nil {
		return err
	}
	name := s[:len(s)-2]

	s, err = r.ReadString('>')
	if err != nil {
		return err
	}
	email := s[:len(s)-1]

	_, err = r.Discard(1)
	if err != nil {
		return err
	}

	s, err = r.ReadString(' ')
	if err != nil {
		return err
	}
	ts, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return err
	}
	when := time.Unix(ts, 0)

	s, err = r.ReadString('\t')

	if err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02 15:04:05 -0700", "0000-01-01 00:00:00 "+s[:len(s)-1])
	if err != nil {
		return err
	}
	when = when.In(t.Location())

	s, err = r.ReadString(':')
	if err != nil {
		return err
	}
	action := s[:len(s)-1]
	modifier := ""
	i := strings.IndexByte(action, ' ')
	if i != -1 {
		modifier = action[i+2 : len(action)-1]
		action = action[:i]
	}

	s, err = r.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	message := s[1:]

	e.OldHash, e.NewHash = oldHash, newHash
	e.Author = Author{
		Name:  name,
		Email: email,
	}
	e.Time = when
	e.Action, e.Modifier = action, modifier
	e.Message = message

	return nil
}
