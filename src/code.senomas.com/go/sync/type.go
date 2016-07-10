package sync

import "time"

// FileData struct
type FileData struct {
	Name string
	Time time.Time
	Size int64
}

// FileDataList struct
type FileDataList struct {
	Files []FileData
}

// FileHash struct
type FileHash struct {
	Name string
	Size int64
	Hash []string
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
