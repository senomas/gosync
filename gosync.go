package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fileData struct {
	name string
	time time.Time
	size int64
}

type fileDatas []fileData

func (files fileDatas) Len() int {
	return len(files)
}

func (files fileDatas) Less(i, j int) bool {
	return files[j].time.Before(files[i].time)
}

func (files fileDatas) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

func main() {
	var files fileDatas
	fp, _ := filepath.Abs(".")
	fmt.Printf("PATH [%s]\n", fp)
	filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
		} else {
			if !strings.HasPrefix(info.Name(), ".") {
				fname := fmt.Sprintf("%s%c%s", path, os.PathSeparator, info.Name())
				files = append(files, fileData{name: fname, time: info.ModTime(), size: info.Size()})
			}
		}
		return err
	})
	sort.Sort(files)
	for _, v := range files {
		fmt.Printf("FILE [%s]  %s  %v\n", v.name, v.time.Format(time.RFC3339), v.size)
	}
}
