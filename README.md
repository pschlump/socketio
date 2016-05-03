# socketio

A copy of github.com/googollee/go-socket.io renamed with defects fixed and updated to 1.3.6 of socket.io

socketio is an implementation of [socket.io](http://socket.io) in Go (golang).
This provides the ability to perform real time communication from a browser
to a server and back.  Content can be pushed from the server out to
the browser.

It is compatible with the 1.3.6 version of socket.io in Node.js, and supports room and namespace.
This version will be updated on a regular basis with the latest version of socket.io.

## Install

Install the package with:

```bash
go get github.com/pschlump/socketio
```

Import it with:

```go
import "github.com/pschlump/socketio"
```

## Example

Please check the ./examples and ./test directory for more comprehensive examples.

```go
package main

import (
	"log"
	"net/http"

	"github.com/pschlump/socketio"
)

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")
		so.Join("chat")
		so.On("chat message", func(msg string) {
			log.Println("emit:", so.Emit("chat message", msg))
			so.BroadcastTo("chat", "chat message", msg)
		})
		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving on port 9000, brows to http://localhost:9000/")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
```

## License

The 3-clause BSD License  - see LICENSE for more details

## History

This code is from an original https://github.com/googollee/go-socket.io .  The following 
have been made:

1. Renamed the package so that the directory structure matches with the package name.
Some outside tools depend on this.
1. Included go-engine.io as a subdirectory.
1. Fixed defect #68 - "Not Thread Safe".  All accesses to maps are synchronized.
1. Documentation improvements.
1. Updated to use the current version of socket.io (1.3.6)
1. Provided a packed(uglified) version of the JavaScript scoket.io library. A non-uglified version is in the 
same directory also.
1. Original defect #95 - Crash occurs when too many arguments are passed - suggested fix used and tested.
1. Fixed a set of continuous connect/disconnect problems
1. #45 - incorrect usage - see correct usage in test/o45 - fixed.
1. #47 - crashing on Windows - unable to reproduce with go 1.3.1 on windows 8.  Appears to be fixed by changes for #68.
1. #83 - see example in test/o83 - fixed.
1. #82 - see example in test/o82 - fixed.
1. #52 - see example in test/o52 - fixed.
1. #67 - see example in test/o67 - fixed.
1. Identified the problem where a emit is sent from client to server and server seems to discard/ignore the emit.  This is caused by an invalid paramter and an ignored error message.  Code review for all discarded/ignored error messages in progress.

## FAQ

1. Why is this not a fork of the original?  A: I can't figure out how to make a fork and change the
name of the package on github.com.   Since a variety of outside tools hurl over the "-" and ".io" in
the directory name I just made a copy and started at the beginning.   My apologies to anybody
that feels offended by this approach.  


