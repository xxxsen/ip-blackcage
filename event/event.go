package event

import "context"

type IEventData interface {
	EventType() string
	Timestamp() int64
	Data() interface{}
}

type defaultEventData struct {
	typ  string
	ts   int64
	data interface{}
}

func (d defaultEventData) Timestamp() int64 {
	return d.ts
}

func (d defaultEventData) Data() interface{} {
	return d.data
}

func (d defaultEventData) EventType() string {
	return d.typ
}

func NewEventData(typ string, ts int64, data interface{}) IEventData {
	return defaultEventData{typ: typ, ts: ts, data: data}
}

type IEventReader interface {
	Open(ctx context.Context) (<-chan IEventData, error)
}
