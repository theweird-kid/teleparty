package services

import (
	"log"
	"sync"

	"github.com/theweird-kid/teleparty/types"
	"github.com/theweird-kid/teleparty/utils"
)

type RoomCleanup struct {
	RoomID string
}

type roomEntry struct {
	Room        *types.Room
	CmdChan     chan types.RoomCommand
	CleanupOnce sync.Once
}

type RoomManager struct {
	Rooms     map[string]*roomEntry
	CleanupCh chan RoomCleanup
	Mu        sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:     make(map[string]*roomEntry), // use roomEntry now
		CleanupCh: make(chan RoomCleanup),
	}
}

func (rm *RoomManager) CreateNewRoom(host *types.User) *types.Room {
	roomID := utils.GenerateRandomID()
	room := &types.Room{
		RoomID:       roomID,
		Host:         host,
		Members:      map[string]*types.User{host.ID: host},
		VideoURL:     "",
		IsPlaying:    false,
		VideoRuntime: 0,
		Messages:     []*types.Message{},
	}
	cmdChan := make(chan types.RoomCommand, 32)

	// start the room goroutine
	go RunRoom(room, cmdChan, rm.CleanupCh)

	// store the entry
	rm.Mu.Lock()
	rm.Rooms[roomID] = &roomEntry{
		Room:    room,
		CmdChan: cmdChan,
	}
	rm.Mu.Unlock()

	return room
}

func (rm *RoomManager) GetRoomEntry(roomID string) (*roomEntry, bool) {
	rm.Mu.RLock()
	defer rm.Mu.RUnlock()
	entry, ok := rm.Rooms[roomID]
	return entry, ok
}

func (rm *RoomManager) StartCleanupListener() {
	go func() {
		for cleanup := range rm.CleanupCh {
			rm.CleanupRoom(cleanup.RoomID)
		}
	}()
}

func (rm *RoomManager) CleanupRoom(roomID string) {
	rm.Mu.Lock()
	entry, exists := rm.Rooms[roomID]
	rm.Mu.Unlock()
	if !exists {
		log.Printf("Room %s already cleaned or does not exist", roomID)
		return
	}

	entry.CleanupOnce.Do(func() {
		log.Printf("Cleaning up room %s", roomID)
		close(entry.CmdChan)

		rm.Mu.Lock()
		delete(rm.Rooms, roomID)
		rm.Mu.Unlock()
	})
}
