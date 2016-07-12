package sync

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Sync struct
type Sync struct {
	config *ssh.ClientConfig
	client *ssh.Client
}

func check(err error, f string, v ...interface{}) {
	if err != nil {
		panic(fmt.Errorf(f, v))
	}
}

func publicKeyFile() ssh.AuthMethod {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	buffer, err := ioutil.ReadFile(usr.HomeDir + "/.ssh/id_rsa")
	check(err, "Failed to open id_rsa %v", err)

	key, err := ssh.ParsePrivateKey(buffer)
	check(err, "Failed to parse id_rsa %v", err)

	return ssh.PublicKeys(key)
}

// Open func
func (sync *Sync) Open(host string) (err error) {
	re, err := regexp.Compile("^(.*)\\@([^:]*)(\\:(\\d+))?$")
	check(err, "Bad regex %v", err)

	px := re.FindStringSubmatch(host)
	if len(px) != 5 {
		panic(fmt.Errorf("Invalid hosts '%s'", host))
	}
	if px[4] == "" {
		px[4] = "22"
	}

	sync.config = &ssh.ClientConfig{
		User: px[1],
		Auth: []ssh.AuthMethod{
			publicKeyFile(),
		},
	}

	sync.client, err = ssh.Dial("tcp", px[2]+":"+px[4], sync.config)
	return err
}

// List func
func (sync *Sync) List(maxSize int, paths []string) (res *FileDataList, err error) {
	session, err := sync.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b

	cmd := fmt.Sprintf("gosync list %v", maxSize)
	for _, v := range paths {
		cmd += " " + v
	}

	err = session.Run(cmd)
	if err != nil {
		return nil, err
	}

	res = &FileDataList{}
	err = json.Unmarshal(b.Bytes(), res)
	return res, err
}

// Sync func
func (sync *Sync) Sync(maxSize int, paths []string) error {
	re, err := regexp.Compile("^([^:]*)(\\:(.*))?$")
	if err != nil {
		panic(err)
	}

	var rpaths, lpaths []string
	for _, v := range paths {
		px := re.FindStringSubmatch(v)
		px[1], err = filepath.Abs(px[1])
		if err != nil {
			panic(err)
		}
		if !strings.HasSuffix(px[1], "/") {
			px[1] += "/"
		}
		lpaths = append(lpaths, px[1])
		if px[3] == "" {
			rpaths = append(rpaths, px[1])
		} else {
			if !strings.HasSuffix(px[3], "/") {
				px[3] += "/"
			}
			rpaths = append(rpaths, px[3])
		}
	}

	res, err := sync.List(maxSize, rpaths)
	if err != nil {
		return err
	}

	for _, v := range res.Files {
		for k, kv := range rpaths {
			if strings.HasPrefix(v.Name, kv) {
				v.Local = lpaths[k] + v.Name[len(kv):]
				break
			}
		}
	}

	for _, v := range res.Files {
		if v.Local != "" {
			err = sync.copy(v)
			if err != nil {
				return err
			}
		}
	}

	err = sync.clean(lpaths, res)

	return err
}

func (sync *Sync) copy(fd *FileData) error {
	if stat, err := os.Stat(fd.Local); os.IsNotExist(err) {
		fmt.Printf("Copy %s\n", fd.Local)
		hash, err := sync.hash(fd.Name)
		if err != nil {
			return err
		}
		err = sync.get(hash, fd.Local)
		if err != nil {
			return err
		}
	} else {
		if stat.Size() == fd.Size {
			fmt.Printf("Skip %s\n", fd.Name)
		} else {
			tempName := filepath.Dir(fd.Local) + "/." + filepath.Base(fd.Local)
			os.Rename(fd.Local, tempName)

			fmt.Printf("Copy %s\n", fd.Local)
			hash, err := sync.hash(fd.Name)
			if err != nil {
				return err
			}
			err = sync.get(hash, fd.Local)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sync *Sync) get(remote *FileData, local string) error {
	var b bytes.Buffer
	hasher := sha256.New()

	os.MkdirAll(filepath.Dir(local), 0777)

	tempName := filepath.Dir(local) + "/." + filepath.Base(local)
	fw, err := os.OpenFile(tempName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	// fw, err := os.Create()
	if err != nil {
		return err
	}
	defer fw.Close()

	_, err = fw.Seek(0, 0)
	if err != nil {
		return err
	}

	buf := make([]byte, 65536)

	pi := 0
	pil := len(remote.Hash)

	for ; pi < pil; pi++ {
		var n int
		n, err = fw.Read(buf)
		hasher.Reset()
		hasher.Write(buf[:n])
		hv := hasher.Sum(nil)
		if !bytes.Equal(remote.Hash[pi], hv) {
			// fmt.Printf("Invalid hash [%v]\n  %v\n  %v\n", n, remote.Hash[pi], hv)
			break
		} else {
			fmt.Printf("  Skip %v                      \r", pi+1)
		}
	}

	_, err = fw.Seek(int64(pi)*65536, 0)
	if err != nil {
		return err
	}
	err = fw.Truncate(int64(pi) * 65536)
	if err != nil {
		return err
	}

	pl := len(remote.Hash)
	for ; pi < pil; pi++ {
		session, err := sync.client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		session.Stdout = &b

		cmd := fmt.Sprintf("gosync get %v \"%s\"", pi, remote.Name)

		fmt.Printf("  Part %v/%v                      \r", pi+1, pl)
		err = session.Run(cmd)
		if err != nil {
			return err
		}

		bb := b.Bytes()
		b.Reset()

		hasher.Reset()
		hasher.Write(bb)

		hv := hasher.Sum(nil)
		if !bytes.Equal(remote.Hash[pi], hv) {
			panic(fmt.Errorf("Invalid hash [%v]\n  %v\n  %v\n", len(bb), remote.Hash[pi], hv))
		}
		fw.Write(bb)
	}
	err = fw.Sync()
	fw.Close()

	os.Rename(tempName, local)
	fmt.Printf("                                               \r")

	return err
}

func (sync *Sync) hash(path string) (res *FileData, err error) {
	session, err := sync.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b

	cmd := fmt.Sprintf("gosync hash \"%s\"", path)

	fmt.Printf("  Get hash                      \r")
	err = session.Run(cmd)
	if err != nil {
		return nil, err
	}

	res = &FileData{}
	err = json.Unmarshal(b.Bytes(), res)
	return res, err
}

func (sync *Sync) clean(lpaths []string, list *FileDataList) error {
	locals := make(map[string]*FileData)
	for _, v := range list.Files {
		if v.Local != "" {
			locals[v.Local] = v
		}
	}
	var dirty []string
	for _, fp := range lpaths {
		filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			} else if info.IsDir() {
				used := false
				for _, v := range list.Files {
					if strings.HasPrefix(v.Local, path) {
						used = true
					}
				}
				if !used {
					dirty = append(dirty, path)
				}
			} else {
				if locals[path] == nil {
					dirty = append(dirty, path)
				}
			}
			return err
		})
	}

	for i := len(dirty) - 1; i >= 0; i-- {
		fp := dirty[i]
		os.Remove(fp)
		fmt.Printf("Remove %s\n", fp)
	}

	return nil
}
