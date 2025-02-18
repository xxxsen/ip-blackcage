package model

type BlackCageTab struct {
	ID      uint64 `json:"id"`
	Remark  string `json:"remark"`
	CTime   uint64 `json:"ctime"`
	MTime   uint64 `json:"mtime"`
	IP      string `json:"ip"`
	Counter int64  `json:"counter"`
}
