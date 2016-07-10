GOPATH=${HOME}/.go:${CURDIR}

build:
	go build src/gosync.go
	git add .
	git commit
	git push
