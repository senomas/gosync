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

type info struct {
	Path  string
	Files []fileData
}

func (inf info) Len() int {
	return len(inf.Files)
}

func (inf info) Less(i, j int) bool {
	return inf.Files[j].Time.Before(inf.Files[i].Time)
}

func (inf info) Swap(i, j int) {
	inf.Files[i], inf.Files[j] = inf.Files[j], inf.Files[i]
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "list" {
		fp, _ := filepath.Abs(".")
		res := info{Path: fp}
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
					res.Files = append(res.Files, fileData{Name: fname, Time: info.ModTime(), Size: info.Size()})
				}
			}
			return err
		})
		sort.Sort(res)
		if len(os.Args) > 2 {
			var maxLen, pLen int64
			maxLen, _ = strconv.ParseInt(os.Args[2], 10, 64)
			maxLen *= 1073741824
			var nfs []fileData
			for _, v := range res.Files {
				pLen += v.Size
				if maxLen == -1 || pLen < maxLen {
					nfs = append(nfs, v)
				}
			}
			res.Files = nfs
		}
		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else {
		fmt.Printf("FORMAT\n  gosync list [max size in GB]\n  gosync hash <file name>\n  gosync get <file name> <part>\n")
	}
}
