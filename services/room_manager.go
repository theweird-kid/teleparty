package services

import (
	"sync"

	"github.com/theweird-kid/teleparty/types"
	"github.com/theweird-kid/teleparty/utils"
)

type RoomManager struct {
	Rooms map[string]*types.Room
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*types.Room),
	}
}

func (rm *RoomManager) CreateNewRoom(host *types.User) *types.Room {
	roomID := utils.GenerateRandomID()
	room := &types.Room{
		RoomID:       roomID,
		Host:         host,
		Members:      map[string]*types.User{host.ID: host},
		VideoURL:     "",
		VideoState:   "paused",
		VideoRuntime: 0,
		Messages:     []*types.Message{},
	}

	rm.Rooms[roomID] = room
	return room
}

func (rm *RoomManager) JoinRoom(user *types.User, roomID string) {
	room := rm.Rooms[roomID]
	room.Mu.Lock()
	defer room.Mu.Unlock()
	room.Members[user.ID] = user
}

func (rm *RoomManager) ExitRoom(user *types.User, roomID string) {
	rm.mu.RLock()
	room, exists := rm.Rooms[roomID]
	rm.mu.RUnlock()
	if !exists || room == nil {
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	delete(room.Members, user.ID)
	if room.Host == user {
		room.Host = nil
		// If the host leaves then promote someone else to host
		for _, u := range room.Members {
			room.Host = u
			break
		}
	}

	// If the room is empty remove it
	if len(room.Members) == 0 && room.Host == nil {
		rm.mu.Lock()
		delete(rm.Rooms, roomID)
		rm.mu.Unlock()
	}
}

func (rm *RoomManager) BroadcastToRoom(msg *types.Message) {
	room := rm.Rooms[msg.RoomID]
	if room == nil {
		return
	}

	room.Mu.RLock()
	defer room.Mu.RUnlock()

	// Check Message Type and User Type : Only Host can Manipulate the video
	if msg.Type != types.MessageTypeChat && msg.FromUser != room.Host {
		return
	}

	// Unicast Sync Request to Host
	if msg.Type == types.MessageTypeSyncRequest {
		// Get the Host of the room
		room, exists := rm.Rooms[msg.RoomID]
		if !exists || room == nil || room.Host == nil {
			return
		}
		select {
		case room.Host.MessageChannel <- msg:
			// Successfully sent to host
		default:
			// Host's channel is full; optionally handle this case (e.g., log, disconnect, etc.)
		}
		return
	}

	// Broadcast to other users
	for _, member := range room.Members {
		if member.ID == msg.FromUser.ID {
			continue
		}
		select {
		case member.MessageChannel <- msg:
		default:
		}
	}
}
