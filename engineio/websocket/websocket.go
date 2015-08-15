package websocket

import (
	"../transport" // "github.com/pschlump/socketio/engineio/transport"
)

var Creater = transport.Creater{
	Name:      "websocket",
	Upgrading: true,
	Server:    NewServer,
	Client:    NewClient,
}
