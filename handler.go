package socketio

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pschlump/godebug"
)

type baseHandler struct {
	events    map[string]*caller
	name      string
	broadcast BroadcastAdaptor
	lock      sync.RWMutex
}

func newBaseHandler(name string, broadcast BroadcastAdaptor) *baseHandler {
	// fmt.Printf("*********************************************************************** this one *********************************************************************\n")
	return &baseHandler{
		events:    make(map[string]*caller),
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

func (h *baseHandler) PrintEventsRespondedTo() {

	fmt.Printf("\tEvents:[ ")
	for i := range h.events {
		fmt.Printf("%s, ", i)
	}
	fmt.Printf("]\n")
}

type socketHandler struct {
	*baseHandler
	acks   map[int]*caller
	socket *socket
	rooms  map[string]struct{}
}

func newSocketHandler(s *socket, base *baseHandler) *socketHandler {
	events := make(map[string]*caller)
	base.lock.Lock()
	for k, v := range base.events {
		events[k] = v
	}
	base.lock.Unlock()
	return &socketHandler{
		baseHandler: &baseHandler{
			events:    events,
			broadcast: base.broadcast,
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
		message = "disconnection"
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
	fmt.Printf("handerl.go: 167: message >%s<\n", message)
	h.lock.RLock()
	c, ok := h.events[message]
	h.lock.RUnlock()
	if !ok {
		if db1 {
			fmt.Printf("At:%s\n", godebug.LF())
		}
		// If the message is not recognized by the server, the decoder.currentCloser
		// needs to be closed otherwise the server will be stuck until the e xyzzy
		fmt.Printf("Error: %s ws not found in h.events\n", message)
		decoder.Close()
		return nil, nil
	}
	args := c.GetArgs()
	if db1 {
		fmt.Printf("At:%s\n", godebug.LF())
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

	if db1 {
		fmt.Printf("Args = %v, h.socket >%s<, %s\n", args, h.socket, godebug.LF())
	}
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

const db1 = false
