package model

type BlackCageTab struct {
	ID        uint64 `json:"id"`
	EventType string `json:"event_type"`
	CTime     uint64 `json:"ctime"`
	MTime     uint64 `json:"mtime"`
	IP        string `json:"ip"`
}
