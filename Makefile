GOPATH=${HOME}/.go:${CURDIR}


build:
	go build src/gosync.go

push: build
	git add .
	git commit
	git push

remote: build
	if git diff-index --cached --quiet HEAD --ignore-submodules ; then git add . ; git commit ; git push ; fi
	ssh root@joker "bash --login -c 'cd ~/workspaces/gosync ; export GOPATH=~/.go:~/workspaces/gosync ; make rbuild'"

rbuild:
	git pull
	go build src/gosync.go
