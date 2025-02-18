package reflog

type EventType string

const (
	NewEntry EventType = "NewEntry"
	Remove   EventType = "Remove"
)

type Event struct {
	Reference string
	Type      EventType
	Entry     Entry
}
