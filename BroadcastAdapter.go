package socketio

import "sync"

// BroadcastAdaptor is the adaptor to handle broadcast.
type BroadcastAdaptor interface {

	// Join lets socket join the t room.
	Join(room string, socket Socket) error

	// Leave let socket leave the room.
	Leave(room string, socket Socket) error

	// Send will send the message with args to room. If ignore is not nil, it won't send to the socket ignore.
	Send(ignore Socket, room, message string, args ...interface{}) error
}

var newBroadcast = newBroadcastDefault

// Broadcast is a set of "room" each with a set of Socket
type broadcast struct {
	roomSet       map[string]map[string]Socket
	broadcastLock sync.RWMutex
}

func newBroadcastDefault() BroadcastAdaptor {
	return &broadcast{
		roomSet: make(map[string]map[string]Socket),
	}
}

// Join into a room
func (b broadcast) Join(room string, socket Socket) error {
	b.broadcastLock.Lock()
	sockets, ok := b.roomSet[room]
	if !ok {
		sockets = make(map[string]Socket)
	}
	sockets[socket.Id()] = socket
	b.roomSet[room] = sockets
	b.broadcastLock.Unlock()
	return nil
}

// Disconnect from a room
func (b broadcast) Leave(room string, socket Socket) error {
	b.broadcastLock.Lock()
	sockets, ok := b.roomSet[room]
	if !ok {
		return nil
	}
	delete(sockets, socket.Id())
	if len(sockets) == 0 {
		delete(b.roomSet, room)
	} else {
		b.roomSet[room] = sockets
	}
	b.broadcastLock.Unlock()
	return nil
}

// Perform a brodcast send to all the sockets in a "room" except the ignored socket.
// Brodcast send to all with ignore == nil.
func (b broadcast) Send(ignore Socket, room, message string, args ...interface{}) error {
	b.broadcastLock.RLock()
	defer b.broadcastLock.RUnlock()
	sockets := b.roomSet[room]
	for id, s := range sockets {
		if ignore != nil && ignore.Id() == id {
			continue
		}
		s.Emit(message, args...)
	}
	return nil
}

// return the number of connections in a specified room
func (b broadcast) NumberInRoom(room string) (rv int, err error) {
	b.broadcastLock.RLock()
	defer b.broadcastLock.RUnlock()
	sockets := b.roomSet[room]
	rv = 0
	for _, _ = range sockets {
		rv++
	}
	return
}

// return the number of rooms
func (b broadcast) NumberOfRooms(room string) (rv int, err error) {
	b.broadcastLock.RLock()
	defer b.broadcastLock.RUnlock()
	rv = 0
	for _, _ = range b.roomSet {
		rv++
	}
	return
}

// return the names of the rooms as a slice of strings
func (b broadcast) ListOfRooms(room string) (rv []string, err error) {
	b.broadcastLock.RLock()
	defer b.broadcastLock.RUnlock()
	for room := range b.roomSet {
		rv = append(rv, room)
	}
	return
}
