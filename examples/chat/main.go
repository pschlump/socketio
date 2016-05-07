package main

//
// Command line arguments can be used to set the IP address that is listened to and the port.
//
// $ ./chat --port=8080 --host=127.0.0.1
//
// Bring up a pair of browsers and chat between them.
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
var Dir = flag.String("dir", "./asset", "Direcotry where files are served from")      // 1
func init() {
	flag.StringVar(Port, "P", "9000", "Port to listen to")                           // 0
	flag.StringVar(HostIP, "H", "localhost", "Host name or IP address to listen on") // 1
	flag.StringVar(Dir, "d", "./asset", "Direcotry where files are served from")     // 1
}

func main() {

	flag.Parse()
	fns := flag.Args()

	if len(fns) != 0 {
		fmt.Printf("Usage: Invalid arguments supplied, %s\n", fns)
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
	fmt.Printf("Serving on port %s, brows to http://localhost:%s/\n", *Port, *Port)
	listen := fmt.Sprintf("%s:%s", host_ip, *Port)
	log.Fatal(http.ListenAndServe(listen, nil))
}
