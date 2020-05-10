package main

//
// Command line arguments can be used to set the IP address that is listened to and the port.
//
// $ ./chat --port=8080 --host=127.0.0.1 --dir=./asset
//
// Bring up a pair of browsers and chat between them.
//

//
// Notes
//
// 1. Updated to use current jQuery 3.1.5 -- Sun May 10 06:45:52 MDT 2020
//

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pschlump/MiscLib"
	"github.com/pschlump/godebug"
	"github.com/pschlump/socketio"
)

var Port = flag.String("port", "9000", "Port to listen to")                           // 0
var HostIP = flag.String("host", "localhost", "Host name or IP address to listen on") // 1
var Dir = flag.String("dir", "./asset", "Direcotry where files are served from")      // 2
func init() {
	flag.StringVar(Port, "P", "9000", "Port to listen to")                           // 0
	flag.StringVar(HostIP, "H", "localhost", "Host name or IP address to listen on") // 1
	flag.StringVar(Dir, "d", "./asset", "Direcotry where files are served from")     // 2
}

func Usage() {
	fmt.Printf(`
Compile and run server with:

$ go run main.go [ -P | --port #### ] [ -H | --host IP-Host ] [ -d | --dir Path-To-Assets ]

-P | --port        Port number.  Default 9000
-H | --host        Host to listen on.  Default 'localhost' but can be an IP or 0.0.0.0 for all.
-d | --dir         Directory to serve with files.  Default ./asset.

`)
}

func main() {

	flag.Parse()
	fns := flag.Args()

	if len(fns) != 0 {
		fmt.Printf("Usage: Invalid arguments supplied, %s\n", fns)
		Usage()
		os.Exit(1)
	}

	var host_ip string = ""
	if *HostIP != "localhost" {
		host_ip = *HostIP
	}

	// Make certain that the command line parameters are handled correctly
	// fmt.Printf("host_ip >%s< HostIP >%s< Port >%s<\n", host_ip, *HostIP, *Port)

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.On("connection", func(so socketio.Socket) {
		fmt.Printf("%sa user connected%s, %s\n", MiscLib.ColorGreen, MiscLib.ColorReset, godebug.LF())
		so.Join("chat")
		so.On("chat message", func(msg string) {
			fmt.Printf("%schat message, %s%s, %s\n", MiscLib.ColorGreen, msg, MiscLib.ColorReset, godebug.LF())
			so.BroadcastTo("chat", "chat message", msg)
		})
		so.On("disconnect", func() {
			fmt.Printf("%suser disconnect%s, %s\n", MiscLib.ColorYellow, MiscLib.ColorReset, godebug.LF())
		})
	})

	server.On("error", func(so socketio.Socket, err error) {
		fmt.Printf("Error: %s, %s\n", err, godebug.LF())
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir(*Dir)))
	fmt.Printf("Serving on port %s, browse to http://localhost:%s/\n", *Port, *Port)
	listen := fmt.Sprintf("%s:%s", host_ip, *Port)
	log.Fatal(http.ListenAndServe(listen, nil))
}
