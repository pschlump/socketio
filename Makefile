
all:
	go build

test:
	( cd ./engineio ; make test )
	go test

clean:
	rm -f test/o45/o45 test/o52/o52 test/o67/o67 test/o82/o82 test/o83/o83 examples/chat/chat

