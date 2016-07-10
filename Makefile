GOPATH=${HOME}/.go:${CURDIR}

build:
	go build src/gosync.go

push: build
	git add .
	git commit
	git push

rbuild:
	git pull
	go build src/gosync.go
