package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type fileData struct {
	Name string
	Time time.Time
	Size int64
}

type info struct {
	Files []fileData
}

type fhash struct {
	Name string
	Size int64
	Hash []string
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	check(err)

	key, err := ssh.ParsePrivateKey(buffer)
	check(err)

	return ssh.PublicKeys(key)
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	usr, err := user.Current()
	check(err)

	if len(os.Args) > 3 && os.Args[1] == "list" {
		res := info{}
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
						res.Files = append(res.Files, fileData{Name: path, Time: info.ModTime(), Size: info.Size()})
					}
				}
				return err
			})
		}
		sort.Sort(res)
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

		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	} else if len(os.Args) == 3 && os.Args[1] == "hash" {
		fp, _ := filepath.Abs(os.Args[2])
		finfo, err := os.Stat(fp)
		check(err)

		hasher := sha256.New()
		res := fhash{Name: fp, Size: finfo.Size()}

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
		sshConfig := &ssh.ClientConfig{
			User: os.Args[2],
			Auth: []ssh.AuthMethod{
				publicKeyFile(usr.HomeDir + "/.ssh/id_rsa"),
			},
		}

		conn, err := ssh.Dial("tcp", os.Args[3], sshConfig)
		check(err)

		session, err := conn.NewSession()
		check(err)

		stdout, err := session.StdoutPipe()
		check(err)
		go io.Copy(os.Stdout, stdout)

		cmd := "gosync list"
		for i, len := 4, len(os.Args); i < len; i++ {
			cmd += " " + os.Args[i]
		}
		fmt.Printf("EXEC [%s]\n", cmd)
		err = session.Run(cmd)
		check(err)

		session.Close()
	} else {
		fmt.Printf("FORMAT\n  gosync list <max size in GB> <path>\n  gosync hash <file name>\n  gosync get <file name> <part>\n  gosync sync <user> <host> <path> [max size in GB]\n")
	}
}
