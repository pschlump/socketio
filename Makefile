
all:
	go build

test:
	( cd ./engineio ; make test )
	go test

