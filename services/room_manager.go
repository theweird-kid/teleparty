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

type RoomManager struct {
	Rooms     map[string]*types.Room
	CmdChans  map[string]chan types.RoomCommand
	CleanupCh chan RoomCleanup
	Mu        sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:     make(map[string]*types.Room),
		CmdChans:  make(map[string]chan types.RoomCommand),
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
	go RunRoom(room, cmdChan, rm.CleanupCh)

	rm.Mu.Lock()
	rm.Rooms[roomID] = room
	rm.CmdChans[roomID] = cmdChan
	rm.Mu.Unlock()

	return room
}

func (rm *RoomManager) StartCleanupListener() {
	go func() {
		for cleanup := range rm.CleanupCh {
			rm.Mu.Lock()
			if cmdChan, ok := rm.CmdChans[cleanup.RoomID]; ok {
				close(cmdChan) // Close the command channel to stop the goroutine if not already done
				delete(rm.CmdChans, cleanup.RoomID)
			}
			delete(rm.Rooms, cleanup.RoomID)
			rm.Mu.Unlock()
			log.Printf("Room %s cleaned up", cleanup.RoomID)
		}
	}()
}
