GOPATH=${HOME}/.go:${CURDIR}


build:
	go build src/gosync.go

push: build
	git add .
	git commit
	git push

remote: build
	git add . ; git commit ; git push
	ssh root@joker "bash --login -c 'cd ~/workspaces/gosync ; export GOPATH=~/.go:~/workspaces/gosync ; make rbuild'"

rbuild:
	git pull
	go build src/gosync.go
