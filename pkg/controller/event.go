package controller

type eventType string

const (
	addRds    eventType = "addRds"
	delRds    eventType = "delRds"
	updateRds eventType = "updateRds"
)

type event struct {
	eventType      eventType
	oldObj, newObj interface{}
}
