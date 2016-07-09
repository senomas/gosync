package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	if len(os.Args) > 1 && os.Args[1] == "list" {
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
		var maxLen, pLen int64
		if len(os.Args) > 2 {
			maxLen, _ = strconv.ParseInt(os.Args[2], 10, 64)
			maxLen *= 1024 * 1024
		} else {
			maxLen = -1
		}
		for _, v := range files {
			pLen += v.size
			if maxLen == -1 && pLen < maxLen {
				fmt.Printf("FILE [%s]  %s  %v\n", v.name, v.time.Format(time.RFC3339), v.size)
			}
		}
	} else {
		fmt.Printf("FORMAT\n  gosync list\n  gosync hash\n")
	}
}
