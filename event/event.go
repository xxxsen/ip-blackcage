package event

import "context"

type IEventData interface {
	Timestamp() int64
	Data() interface{}
}

type defaultEventData struct {
	ts   int64
	data interface{}
}

func (d defaultEventData) Timestamp() int64 {
	return d.ts
}

func (d defaultEventData) Data() interface{} {
	return d.data
}

func NewEventData(ts int64, data interface{}) IEventData {
	return defaultEventData{ts: ts, data: data}
}

type IEventReader interface {
	Open(ctx context.Context) (<-chan IEventData, error)
}
