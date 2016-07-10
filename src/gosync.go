package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
			fp, _ := filepath.Abs(os.Args[i])
			filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				} else if info.IsDir() {
					if strings.HasPrefix(info.Name(), ".") {
						return filepath.SkipDir
					}
				} else {
					if !strings.HasPrefix(info.Name(), ".") {
						res.Files = append(res.Files, sync.FileData{Name: path, Time: info.ModTime(), Size: info.Size()})
					}
				}
				return err
			})
		}
		sort.Sort(res)
		var maxLen, pLen int64
		maxLen, _ = strconv.ParseInt(os.Args[2], 10, 64)
		maxLen *= 1073741824
		var nfs []sync.FileData
		for _, v := range res.Files {
			pLen += v.Size
			if maxLen == -1 || pLen < maxLen {
				nfs = append(nfs, v)
			}
		}
		res.Files = nfs

		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else if len(os.Args) == 3 && os.Args[1] == "hash" {
		fp, _ := filepath.Abs(os.Args[2])
		finfo, err := os.Stat(fp)
		check(err)

		hasher := sha256.New()
		res := sync.FileHash{Name: fp, Size: finfo.Size()}

		f, err := os.Open(fp)
		check(err)

		buf := make([]byte, 1024)

		i := 0
		for ; ; i++ {
			n, err := f.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			} else {
				hasher.Write(buf[:n])
				if i == 1024 {
					res.Hash = append(res.Hash, base64.StdEncoding.EncodeToString(hasher.Sum(nil)))
					i = 0
				}
			}
		}
		if i > 0 {
			res.Hash = append(res.Hash, base64.StdEncoding.EncodeToString(hasher.Sum(nil)))
		}

		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else if len(os.Args) == 4 && os.Args[1] == "get" {
		fp, _ := filepath.Abs(os.Args[2])

		f, err := os.Open(fp)
		check(err)

		part, err := strconv.ParseInt(os.Args[3], 10, 64)
		check(err)

		_, err = f.Seek(part*1048576, 0)
		check(err)

		buf := make([]byte, 1024)

		for i := 0; i < 1024; i++ {
			n, err := f.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			} else {
				os.Stdout.Write(buf[:n])
			}
		}
	} else if len(os.Args) >= 5 && os.Args[1] == "sync" {
		sshSync := sync.Sync{}

		err := sshSync.Open(os.Args[2])
		check(err)

		maxSize, err := strconv.Atoi(os.Args[3])
		check(err)

		re, err := regexp.Compile("^([^:]*)(\\:(.*))?$")
		check(err)

		var paths []string
		for _, v := range os.Args[4:] {
			px := re.FindStringSubmatch(v)
			if px[3] == "" {
				if !strings.HasSuffix(px[1], "/") {
					px[1] += "/"
				}
				paths = append(paths, px[1])
			} else {
				if !strings.HasSuffix(px[3], "/") {
					px[3] += "/"
				}
				paths = append(paths, px[3])
			}
		}

		res, err := sshSync.List(maxSize, paths)
		check(err)

		fmt.Printf("RESULT %+v\n", res)
	} else {
		fmt.Printf("FORMAT\n  gosync list <max size in GB> <path>\n  gosync hash <file name>\n  gosync get <file name> <part>\n  gosync sync <user> <host> <path> [max size in GB]\n")
	}
}