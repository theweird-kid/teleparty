package services

import (
	"log"

	"github.com/theweird-kid/teleparty/types"
)

func handleLeave(room *types.Room, cmd types.RoomCommand) bool {
	delete(room.Members, cmd.User.ID)
	// check for host and promotion or close room if empty
	if cmd.User.ID == room.Host.ID {
		if len(room.Members) == 0 {
			return false
		}
		for _, member := range room.Members {
			room.Host = member
			break
		}

		msg := &types.Message{
			Type: types.MessageTypeHostUpdate,
			Data: map[string]interface{}{
				"room_id":   room.RoomID,
				"host_id":   room.Host.ID,
				"host_name": room.Host.Name,
			},
		}

		// Broadcast host Update
		for _, member := range room.Members {
			select {
			case member.MessageChannel <- msg:
			default:
				log.Printf("MessageChannel full for %s", member.Name)
			}
		}
	}
	log.Printf("%s left room %s", cmd.User.Name, room.RoomID)
	log.Printf("Host update to %s", room.Host.Name)
	return true
}

func RunRoom(room *types.Room, commands <-chan types.RoomCommand, cleanupCh chan<- RoomCleanup) {
	for cmd := range commands {
		switch cmd.Type {
		case types.CmdJoin:
			room.Members[cmd.User.ID] = cmd.User
			log.Printf("%s joined room %s", cmd.User.Name, room.RoomID)
		case types.CmdLeave:
			if resp := handleLeave(room, cmd); resp == false {
				cleanupCh <- RoomCleanup{RoomID: room.RoomID}
				return
			}
		case types.CmdSyncReq:
			// If host request sync
			if room.Host.ID == cmd.User.ID {
				room.Host.MessageChannel <- &types.Message{
					Type: types.MessageTypeSyncResponse,
					Data: map[string]interface{}{
						"room_id":      room.RoomID,
						"host_id":      room.Host.ID,
						"host_name":    room.Host.Name,
						"video_url":    room.VideoURL,
						"current_time": room.VideoRuntime,
						"is_playing":   room.IsPlaying,
					},
				}
			} else {
				// If user request sync
				room.Host.MessageChannel <- cmd.Message
			}
		case types.CmdBroadcast:
			for _, member := range room.Members {
				if member.ID != cmd.User.ID || cmd.Message.Type == types.MessageTypeChat {
					select {
					case member.MessageChannel <- cmd.Message:
					default:
						log.Printf("MessageChannel full for %s", member.Name)
					}
				}
			}
		default:
		}
	}
}
