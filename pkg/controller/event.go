package controller

type eventType string

const (
	addRds eventType = "addRds"
	delRds eventType = "delRds"
)

type event struct {
	eventType      eventType
	oldObj, newObj interface{}
}
