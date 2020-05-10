
all:
	go build

test:
	( cd ./engineio ; make test )
	go test

clean:
	rm -f o45/o45 o52/o52 o67/o67 o82/o82 o83/o83 examples/chat/chat

