package controller

type eventType string

const (
	addRds eventType = "addRds"
)

type event struct {
	eventType      eventType
	oldObj, newObj interface{}
}
