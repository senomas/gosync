package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/user"
	"regexp"

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
