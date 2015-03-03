all: build test

build:
		go build -v ./...

test:
		go test -covermode=count -coverprofile=api.cover.out -test.short -coverpkg=./... ./api
		go test -covermode=count -coverprofile=daemon.cover.out -test.short -coverpkg=./... ./daemon
		go test -covermode=count -coverprofile=main.cover.out -test.short

test-all:
		go test -covermode=count -coverprofile=api.cover.out -coverpkg=./... ./api
		go test -covermode=count -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
		go test -covermode=count -coverprofile=main.cover.out
