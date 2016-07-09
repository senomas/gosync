package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type fileData struct {
	Name string
	Time time.Time
	Size int64
}

type fileDatas []fileData

func (files fileDatas) Len() int {
	return len(files)
}

func (files fileDatas) Less(i, j int) bool {
	return files[j].Time.Before(files[i].Time)
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
					files = append(files, fileData{Name: fname, Time: info.ModTime(), Size: info.Size()})
				}
			}
			return err
		})
		sort.Sort(files)
		if len(os.Args) > 2 {
			var maxLen, pLen int64
			maxLen, _ = strconv.ParseInt(os.Args[2], 10, 64)
			maxLen *= 1073741824
			var nfs fileDatas
			for _, v := range files {
				pLen += v.Size
				if maxLen == -1 || pLen < maxLen {
					nfs = append(nfs, v)
				}
			}
			files = nfs
		}
		b, _ := json.Marshal(files)
		fmt.Printf("%s", string(b))
	} else {
		fmt.Printf("FORMAT\n  gosync list\n  gosync hash\n")
	}
}
