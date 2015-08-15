package main

//
// Command line arguments can be used to set the IP address that is liseneed to and the port.
//
// $ ./chat --port=8080 --host=127.0.0.1
//
// Bring up a pair of browsers and chat betwen them.
//

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	flags "github.com/jessevdk/go-flags"

	"../../../socketio" // "github.com/pschlump/scoketio/socketio"
)

var port string = "9000"
var host_ip string = ""

var opts struct {
	Port   int    `short:"P" long:"port"     description:"Port to listen to"                     default:"9000"`
	HostIP string `short:"H" long:"host"     description:"Host or IP address to listen on"       default:"localhost"`
}

// Simulate a user login -
//		bob/123
//		jane/abc
// are valid logins
func ValidateUser(un, pw string) (ok bool) {
	ok = false
	if un == "bob" && pw == "123" || un == "jane" && pw == "abc" {
		ok = true
	}
	return
}

type UserInfo struct {
	User       string
	IsLoggedIn bool
	So         socketio.Socket
}

var UserMap map[string]*UserInfo
var UserMapLock sync.RWMutex

func init() {
	UserMap = make(map[string]*UserInfo)
}

func main() {

	junk, err := flags.ParseArgs(&opts, os.Args)

	if len(junk) != 1 {
		fmt.Printf("Usage: Invalid arguments supplied, %s\n", junk)
		os.Exit(1)
	}
	if err != nil {
		os.Exit(1)
	}

	port = fmt.Sprintf("%d", opts.Port)
	if opts.HostIP != "localhost" {
		host_ip = opts.HostIP
	}

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")
		so.Join("chat")
		so.On("chat message", func(msg string) {
			m := make(map[string]interface{})
			m["a"] = "你好" // hello there
			e := so.Emit("cn1111", m)
			//这个没有问题			// this is no problem
			fmt.Println("\n\n")

			b := make(map[string]string)
			b["u-a"] = "中文内容" //这个不能是中文		// this is chineese // this can not be chineese
			m["b-c"] = b
			e = so.Emit("cn2222", m)
			log.Println(e)

			log.Println("emit:", so.Emit("chat message", msg))
			so.BroadcastTo("chat", "chat message", msg)
		})
		so.On("t45", func(msg string) {
			err := so.Emit("r45", "Yep")
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
		})
		so.On("registerForName", func(name, pass string) {
			b := ValidateUser(name, pass)
			// store the soketio.Socket in a global map
			UserMapLock.Lock()
			UserMap[name] = &UserInfo{
				User:       name,
				IsLoggedIn: b,
				So:         so,
			}
			UserMapLock.Unlock()
			if b {
				so.Emit("chat message", "logged in")
			} else {
				so.Emit("chat message", "invalid username/password (try bob/123 or jane/abc)")
			}
		})
		so.On("logout", func(name string) {
			UserMapLock.Lock()
			delete(UserMap, name)
			UserMapLock.Unlock()
			so.Emit("chat message", "bye now")
		})
		so.On("sendMessageTo", func(iAm, name string, message string) {
			fmt.Printf("sendMessage func, name = %v, message = %v", name, message)
			UserMapLock.RLock()
			defer UserMapLock.RUnlock()
			from, ok1 := UserMap[iAm]
			if !ok1 || !from.IsLoggedIn {
				so.Emit("error", "Error: you are not logged in.")
				return
			}
			dest, ok2 := UserMap[name]
			if !ok2 || !dest.IsLoggedIn {
				so.Emit("error", "Error: not logged in:"+name)
				return
			}
			dest.So.Emit("chat message", message)
		})
		so.On("disconnection", func() {
			// xyzzy - get "name" from so.Id() - use for auto/logout on disconnect - cleanup of global hash
			log.Println("on disconnect")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	fmt.Printf("Serving on port %s, brows to http://localhost:%s/\n", port, port)
	listen := fmt.Sprintf("%s:%s", host_ip, port)
	log.Fatal(http.ListenAndServe(listen, nil))
}
