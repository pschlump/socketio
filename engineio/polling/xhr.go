package polling

import (
	"github.com/pschlump/socketio/engineio/transport"
)

var Creater = transport.Creater{
	Name:      "polling",
	Upgrading: false,
	Server:    NewServer,
	Client:    NewClient,
}
