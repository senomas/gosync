GOPATH=${HOME}/.go:${CURDIR}


build:
	export GOPATH
	go build src/gosync.go

push: build
	git add .
	git commit ; git push

remote: push
	ssh root@joker "bash --login -c 'cd ~/workspaces/gosync ; export GOPATH=~/.go:~/workspaces/gosync ; make pull-build'"

pull-build:
	git pull
	go build src/gosync.go
