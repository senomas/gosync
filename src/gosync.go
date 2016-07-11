package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"code.senomas.com/go/sync"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	if len(os.Args) > 3 && os.Args[1] == "list" {
		res := sync.FileDataList{}
		for i, l := 3, len(os.Args); i < l; i++ {
			fp, err := filepath.Abs(os.Args[i])
			if err != nil {
				panic(err)
			}
			filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				} else if info.IsDir() {
					if strings.HasPrefix(info.Name(), ".") {
						return filepath.SkipDir
					}
				} else {
					if !strings.HasPrefix(info.Name(), ".") {
						res.Files = append(res.Files, &sync.FileData{Name: path, Time: info.ModTime(), Size: info.Size()})
					}
				}
				return err
			})
		}
		sort.Sort(res)
		var maxLen, pLen int64
		maxLen, _ = strconv.ParseInt(os.Args[2], 10, 64)
		maxLen *= 1073741824
		var nfs []*sync.FileData
		for _, v := range res.Files {
			pLen += v.Size
			if maxLen == -1 || pLen < maxLen {
				nfs = append(nfs, v)
			}
		}
		res.Files = nfs
		sort.Reverse(res)

		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else if len(os.Args) == 3 && os.Args[1] == "hash" {
		fp, _ := filepath.Abs(os.Args[2])
		finfo, err := os.Stat(fp)
		check(err)

		hasher := sha256.New()
		res := sync.FileData{Name: fp, Size: finfo.Size(), Time: finfo.ModTime()}

		f, err := os.Open(fp)
		check(err)

		buf := make([]byte, 1024)

		for i := 1; ; i++ {
			n, err := f.Read(buf)
			if n > 0 {
				hasher.Write(buf[:n])
			}
			if err == io.EOF {
				res.Hash = append(res.Hash, hasher.Sum(nil))
				hasher.Reset()
				break
			} else if err != nil {
				panic(err)
			} else {
				if i == 64 {
					res.Hash = append(res.Hash, hasher.Sum(nil))
					hasher.Reset()
					i = 0
				}
			}
		}

		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else if len(os.Args) == 4 && os.Args[1] == "get" {
		part, err := strconv.ParseInt(os.Args[2], 10, 64)
		check(err)

		fp, _ := filepath.Abs(os.Args[3])

		f, err := os.Open(fp)
		check(err)

		_, err = f.Seek(part*65536, 0)
		check(err)

		buf := make([]byte, 1024)

		for i := 0; i < 64; i++ {
			n, err := f.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			} else {
				os.Stdout.Write(buf[:n])
				// fmt.Println(base64.StdEncoding.EncodeToString(buf[:n]))
			}
		}
	} else if len(os.Args) >= 5 && os.Args[1] == "sync" {
		sshSync := sync.Sync{}

		err := sshSync.Open(os.Args[2])
		check(err)

		maxSize, err := strconv.Atoi(os.Args[3])
		check(err)

		err = sshSync.Sync(maxSize, os.Args[4:])
		check(err)

	} else {
		fmt.Printf("FORMAT\n  gosync list <max size in GB> <path>\n  gosync hash <file name>\n  gosync get <file name> <part>\n  gosync sync <user> <host> <path> [max size in GB]\n")
	}
}
