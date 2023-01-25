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

func parseEntry(reference string, s string) (*Entry, error) {
	r := bufio.NewReader(strings.NewReader(s))

	s, err := r.ReadString(' ')
	if err != nil {
		return nil, err
	}
	oldHash := s[:len(s)-1]

	s, err = r.ReadString(' ')
	if err != nil {
		return nil, err
	}
	newHash := s[:len(s)-1]

	s, err = r.ReadString('<')
	if err != nil {
		return nil, err
	}
	name := s[:len(s)-2]

	s, err = r.ReadString('>')
	if err != nil {
		return nil, err
	}
	email := s[:len(s)-1]

	_, err = r.Discard(1)
	if err != nil {
		return nil, err
	}

	s, err = r.ReadString(' ')
	if err != nil {
		return nil, err
	}
	ts, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return nil, err
	}
	when := time.Unix(ts, 0)

	s, err = r.ReadString('\t')

	if err != nil {
		return nil, err
	}
	t, err := time.Parse("2006-01-02 15:04:05 -0700", "0000-01-01 00:00:00 "+s[:len(s)-1])
	if err != nil {
		return nil, err
	}
	when = when.In(t.Location())

	s, err = r.ReadString(':')
	if err != nil {
		return nil, err
	}
	action := s[:len(s)-1]
	modifier := ""
	i := strings.IndexByte(action, ' ')
	if i != -1 {
		modifier = action[i+1 : len(action)-1]
		action = action[:i-1]
	}

	s, err = r.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	message := s[1:]

	return &Entry{
		Reference: reference,
		OldHash:   oldHash,
		NewHash:   newHash,
		Author: Author{
			Name:  name,
			Email: email,
		},
		Time:     when,
		Action:   action,
		Modifier: modifier,
		Message:  message,
	}, nil
}
