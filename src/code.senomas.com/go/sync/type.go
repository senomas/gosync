package sync

import "time"

// FileData struct
type FileData struct {
	Name  string    `json:"name,omitempty"`
	Local string    `json:"name,omitempty"`
	Time  time.Time `json:"time,omitempty"`
	Size  int64     `json:"size,omitempty"`
	Hash  [][]byte  `json:"hash,omitempty"`
}

// FileDataList struct
type FileDataList struct {
	Files []FileData
}

func (fl FileDataList) Len() int {
	return len(fl.Files)
}

func (fl FileDataList) Less(i, j int) bool {
	return fl.Files[j].Time.Before(fl.Files[i].Time)
}

func (fl FileDataList) Swap(i, j int) {
	fl.Files[i], fl.Files[j] = fl.Files[j], fl.Files[i]
}
