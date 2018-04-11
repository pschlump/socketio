package socketio

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/pschlump/godebug"
)

// PJS - could have it return more than just an error, if "rmsg" and "rbody" - then emit response?
// that makes it more like a RPC - call a func get back a response
type EventHandlerFunc func(so *Socket, message string, args [][]byte) error

type baseHandler struct {
	events      map[string]*caller
	allEvents   []*caller
	x_events    map[string]EventHandlerFunc
	x_allEvents []EventHandlerFunc
	name        string
	broadcast   BroadcastAdaptor
	lock        sync.RWMutex
}

func newBaseHandler(name string, broadcast BroadcastAdaptor) *baseHandler {
	// fmt.Printf("*********************************************************************** this one *********************************************************************\n")
	return &baseHandler{
		events:    make(map[string]*caller),
		allEvents: make([]*caller, 0, 5),
		name:      name,
		broadcast: broadcast,
	}
}

// On registers the function f to handle message.
func (h *baseHandler) On(message string, f interface{}) error {
	c, err := newCaller(f)
	if err != nil {
		return err
	}
	h.lock.Lock()
	h.events[message] = c
	h.lock.Unlock()
	return nil
}

func (h *baseHandler) Handle(message string, f EventHandlerFunc) error {
	h.lock.Lock()
	h.x_events[message] = f
	h.lock.Unlock()
	return nil
}

func (h *baseHandler) HandleAny(f EventHandlerFunc) error {
	h.lock.Lock()
	h.x_allEvents = append(h.x_allEvents, f)
	h.lock.Unlock()
	return nil
}

// On registers the function f to handle ANY message.
func (h *baseHandler) OnAny(f interface{}) error {
	c, err := newCaller(f)
	if err != nil {
		return err
	}
	h.lock.Lock()
	h.allEvents = append(h.allEvents, c)
	h.lock.Unlock()
	return nil
}

func (h *baseHandler) PrintEventsRespondedTo() {
	fmt.Printf("\tEvents:[")
	com := ""
	for i := range h.events {
		fmt.Printf("%s%s", com, i)
		com = ", "
	}
	fmt.Printf(" ] AllEvents = %d", len(h.allEvents))
	fmt.Printf("\n")
}

type socketHandler struct {
	*baseHandler
	acks   map[int]*caller
	socket *socket
	rooms  map[string]struct{}
}

func newSocketHandler(s *socket, base *baseHandler) *socketHandler {
	events := make(map[string]*caller)
	allEvents := make([]*caller, 0, 5)
	x_events := make(map[string]EventHandlerFunc)
	x_allEvents := make([]EventHandlerFunc, 0, 5)
	base.lock.Lock()
	for k, v := range base.events {
		events[k] = v
	}
	base.lock.Unlock()
	return &socketHandler{
		baseHandler: &baseHandler{
			events:      events,
			allEvents:   allEvents,
			x_events:    x_events,
			x_allEvents: x_allEvents,
			broadcast:   base.broadcast,
		},
		acks:   make(map[int]*caller),
		socket: s,
		rooms:  make(map[string]struct{}),
	}
}

func (h *socketHandler) Emit(message string, args ...interface{}) error {
	var c *caller
	if l := len(args); l > 0 {
		fv := reflect.ValueOf(args[l-1])
		if fv.Kind() == reflect.Func {
			var err error
			c, err = newCaller(args[l-1])
			if err != nil {
				return err
			}
			args = args[:l-1]
		}
	}
	args = append([]interface{}{message}, args...)
	h.lock.Lock()
	defer h.lock.Unlock()
	if c != nil {
		id, err := h.socket.sendId(args)
		if err != nil {
			return err
		}
		h.acks[id] = c
		return nil
	}
	return h.socket.send(args)
}

func (h *socketHandler) Rooms() []string {
	h.lock.RLock()
	defer h.lock.RUnlock()
	ret := make([]string, len(h.rooms))
	i := 0
	for room := range h.rooms {
		ret[i] = room
		i++
	}
	return ret
}

func (h *socketHandler) Join(room string) error {
	if err := h.baseHandler.broadcast.Join(h.broadcastName(room), h.socket); err != nil {
		return err
	}
	h.lock.Lock()
	h.rooms[room] = struct{}{}
	h.lock.Unlock()
	return nil
}

func (h *socketHandler) Leave(room string) error {
	if err := h.baseHandler.broadcast.Leave(h.broadcastName(room), h.socket); err != nil {
		return err
	}
	h.lock.Lock()
	delete(h.rooms, room)
	h.lock.Unlock()
	return nil
}

func (h *socketHandler) LeaveAll() error {
	h.lock.RLock()
	tmp := h.rooms
	h.lock.RUnlock()
	for room := range tmp {
		if err := h.baseHandler.broadcast.Leave(h.broadcastName(room), h.socket); err != nil {
			return err
		}
	}
	return nil
}

func (h *baseHandler) BroadcastTo(room, message string, args ...interface{}) error {
	return h.broadcast.Send(nil, h.broadcastName(room), message, args...)
}

func (h *socketHandler) BroadcastTo(room, message string, args ...interface{}) error {
	return h.baseHandler.broadcast.Send(h.socket, h.broadcastName(room), message, args...)
}

func (h *baseHandler) broadcastName(room string) string {
	return fmt.Sprintf("%s:%s", h.name, room)
}

func (h *socketHandler) onPacket(decoder *decoder, packet *packet) ([]interface{}, error) {
	if db1 {
		fmt.Printf("At:%s\n", godebug.LF())
	}
	var message string
	switch packet.Type {
	case _CONNECT:
		message = "connection"
	case _DISCONNECT:
		message = "disconnect"
	case _ERROR:
		message = "error"
	case _ACK:
	case _BINARY_ACK:
		return nil, h.onAck(packet.Id, decoder, packet)
	default:
		message = decoder.Message()
	}
	if db1 {
		fmt.Printf("At:%s\n", godebug.LF())
	}
	h.PrintEventsRespondedTo()
	if DbLogMessage {
		fmt.Printf("Message [%s] ", message)
		if db1 {
			fmt.Printf("%s\n", godebug.LF())
		}
	}

	/*
		// xyzzy - allEvents
		for _, c2 := range h.allEvents {
			args := c2.GetArgs() // returns Array of interface{}
			olen := len(args)
		}
	*/

	h.lock.RLock()
	c, ok := h.events[message]
	xc, ok1 := h.x_events[message]
	h.lock.RUnlock()

	if !ok && !ok1 {
		if db1 {
			fmt.Printf("Did not have a handler for %s At:%s\n", message, godebug.LF())
		}
		// If the message is not recognized by the server, the decoder.currentCloser
		// needs to be closed otherwise the server will be stuck until the e xyzzy
		fmt.Printf("Error: %s was not found in h.events\n", message)
		decoder.Close()
		return nil, nil
	}

	_ = xc
	/* New -----------------------------------------------------------------------------------------------------------------
	if ok1 {
		// type EventHandlerFunc func(so *Socket, message string, args [][]byte) error
		err, xargs, nargs := decoder.DecodeDataX(packet)
		if err != nil {
			fmt.Printf("Unable to decode packet, %s, %s\n", err, godebug.LF())
			return nil, err
		}
		err := xc(xyzzy, message, xargs, nargs)
		if err != nil {
			fmt.Printf("Handler reported an error: %s, message=%s\n", err, message, godebug.LF())
			return nil, err
		}
		return nil, nil

	}
	*/

	args := c.GetArgs() // returns Array of interface{}
	if db1 {
		fmt.Printf("len(args) = %d At:%s\n", len(args), godebug.LF())
	}
	olen := len(args)
	if db1 {
		fmt.Printf("args = %v, %s\n", args, godebug.LF())
	}
	if olen > 0 {
		packet.Data = &args
		if err := decoder.DecodeData(packet); err != nil {
			if db1 {
				fmt.Printf("At:%s, err=%s, an error at this point means that your handler did not get called\n", godebug.LF(), err)
			}
			fmt.Printf("Try a `map[string]interface{}` for a parameter type, %s\n", godebug.LF())
			return nil, err
		}
	}

	// Padd out args to olen
	for i := len(args); i < olen; i++ {
		args = append(args, nil)
	}

	if DbLogMessage {
		if db1 {
			fmt.Printf("\tArgs = %s, %s\n", godebug.SVar(args), godebug.LF())
		} else {
			fmt.Printf("Args = %s\n", godebug.SVar(args))
		}
	}
	if LogMessage {
		logrus.Infof("Message [%s] Auruments %s", message, godebug.SVar(args))
	}

	// ------------------------------------------------------ call ---------------------------------------------------------------------------------------
	retV := c.Call(h.socket, args)
	if len(retV) == 0 {
		if db1 {
			fmt.Printf("At:%s\n", godebug.LF())
		}
		return nil, nil
	}

	var err error
	if last, ok := retV[len(retV)-1].Interface().(error); ok {
		err = last
		retV = retV[0 : len(retV)-1]
	}
	ret := make([]interface{}, len(retV))
	for i, v := range retV {
		ret[i] = v.Interface()
	}
	if db1 {
		fmt.Printf("At:%s\n", godebug.LF())
	}
	if DbLogMessage {
		if err != nil {
			fmt.Printf("Response/Error %s", err)
		} else {
			fmt.Printf("Response %s", godebug.SVar(ret))
		}
	}
	if LogMessage {
		if err != nil {
			logrus.Infof("Response/Error %s", err)
		} else {
			logrus.Infof("Response %s", godebug.SVar(ret))
		}
	}
	return ret, err
}

func (h *socketHandler) onAck(id int, decoder *decoder, packet *packet) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	c, ok := h.acks[id]
	if !ok {
		return nil
	}
	delete(h.acks, id)

	args := c.GetArgs()
	packet.Data = &args
	if err := decoder.DecodeData(packet); err != nil {
		return err
	}
	c.Call(h.socket, args)
	return nil
}

const db1 = true

var DbLogMessage = true
var LogMessage = true

/* vim: set noai ts=4 sw=4: */
