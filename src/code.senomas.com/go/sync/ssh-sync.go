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
				fp, err := filepath.Abs(lpaths[k] + v.Name[len(kv):])
				if err == nil {
					v.Local = fp
					// err = sync.copy(v, fp)
					// if err != nil {
					// 	return err
					// }
					break
				}
			}
		}
	}

	for _, v := range res.Files {
		if v.Local != "" {
			fmt.Printf("FILE %s\n", v.Local)
		}
	}

	return nil
}

func (sync *Sync) copy(remote FileData, local string) error {
	if stat, err := os.Stat(local); os.IsNotExist(err) {
		fmt.Printf("Copy %s\n", local)
		hash, err := sync.hash(remote.Name)
		if err != nil {
			return err
		}
		err = sync.get(hash, local)
		if err != nil {
			return err
		}
	} else {
		if stat.Size() == remote.Size {
			fmt.Printf("Skip %s\n", remote.Name)
		} else {
			fmt.Printf("CONTINUE\n  LOCAL: [%s]\n  SIZE REMOTE %v - LOCAL %v\n\n", remote.Name, remote.Size, stat.Size())
		}
	}
	return nil
}

func (sync *Sync) get(remote *FileData, local string) error {
	var b bytes.Buffer
	hasher := sha256.New()

	os.MkdirAll(filepath.Dir(local), 0777)

	tempName := filepath.Dir(local) + "/." + filepath.Base(local)
	fw, err := os.Create(tempName)
	if err != nil {
		return err
	}
	defer fw.Close()

	pl := len(remote.Hash)
	for k, v := range remote.Hash {
		session, err := sync.client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		session.Stdout = &b

		cmd := fmt.Sprintf("gosync get %v \"%s\"", k, remote.Name)

		err = session.Run(cmd)
		if err != nil {
			return err
		}
		fmt.Printf("  Part %v/%v\n", k+1, pl)

		bb := b.Bytes()
		b.Reset()

		hasher.Reset()
		hasher.Write(bb)

		hv := hasher.Sum(nil)
		if !bytes.Equal(v, hv) {
			panic(fmt.Errorf("Invalid hash [%v]\n  %v\n  %v\n", len(bb), v, hv))
		}
		fw.Write(bb)
	}
	err = fw.Sync()
	fw.Close()

	os.Rename(tempName, local)

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

	err = session.Run(cmd)
	if err != nil {
		return nil, err
	}

	res = &FileData{}
	err = json.Unmarshal(b.Bytes(), res)
	return res, err
}

func (sync *Sync) clean(list *FileDataList) error {
	return nil
}
