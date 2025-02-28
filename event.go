package gitoo

type EventType string

const (
	EvtCommit EventType = "commit"
	EvtSwitch EventType = "switch"
)

type Event struct {
	Type EventType
}
