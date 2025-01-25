package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
	"github.com/pikachu0310/livekit-server/openapi/models"
)

// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
func (r *Repository) InitializeRoomState() error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(context.Background())
	r.RoomState = roomWithParticipants
	return err
}

func (r *Repository) AddParticipantToRoomState(room *livekit.Room, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == room.Name {
			t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
			r.RoomState[i].Participants = append(r.RoomState[i].Participants, models.Participant{
				Identity:   &participant.Identity,
				JoinedAt:   &t,
				Name:       &participant.Name,
				Attributes: &participant.Attributes,
				CanPublish: &participant.Permission.CanPublish,
			})
		}
	}
}

func (r *Repository) UpdateParticipantCanPublish(roomId string, participantId string, canPublish bool) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, participant := range roomState.Participants {
				if *participant.Identity == participantId {
					r.RoomState[i].Participants[j].CanPublish = &canPublish
				}
			}
		}
	}
}

func (r *Repository) UpdateParticipant(roomId string, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, p := range roomState.Participants {
				if *p.Identity == participant.Identity {
					t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
					r.RoomState[i].Participants[j] = models.Participant{
						Identity:   &participant.Identity,
						JoinedAt:   &t,
						Name:       &participant.Name,
						Attributes: &participant.Attributes,
					}
				}
			}
		}
	}
}

func (r *Repository) RemoveParticipant(roomId string, participantId string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, participant := range roomState.Participants {
				if *participant.Identity == participantId {
					r.RoomState[i].Participants = append(r.RoomState[i].Participants[:j], r.RoomState[i].Participants[j+1:]...)
				}
			}
		}
	}
}

func (r *Repository) GetRoomsWithParticipantsByLiveKitServerAndSave(ctx context.Context) error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(ctx)
	if err != nil {
		return err
	}
	r.RoomState = roomWithParticipants
	return nil
}

func (r *Repository) AddRoomState(room models.RoomWithParticipants) {
	r.RoomState = append(r.RoomState, room)
}

func (r *Repository) CreateRoomState(roomId string) error {
	roomUUID, err := uuid.Parse(roomId)
	if err != nil {
		return err
	}
	r.AddRoomState(models.RoomWithParticipants{
		RoomId:       roomUUID,
		Participants: make([]models.Participant, 0),
	})
	return nil
}

func (r *Repository) RemoveRoomState(roomId string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			r.RoomState = append(r.RoomState[:i], r.RoomState[i+1:]...)
		}
	}
}

func (r *Repository) NewLiveKitRoomServiceClient() *lksdk.RoomServiceClient {
	return lksdk.NewRoomServiceClient(r.LiveKitHost, r.ApiKey, r.ApiSecret)
}

func (r *Repository) GetRoomsByLiveKitServer(ctx context.Context) (*livekit.ListRoomsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListRooms(ctx, &livekit.ListRoomsRequest{})
}

func (r *Repository) GetParticipantsByLiveKitServer(ctx context.Context, roomId string) (*livekit.ListParticipantsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: roomId,
	})
}

func (r *Repository) GetRoomsWithParticipantsByLiveKitServer(ctx context.Context) ([]models.RoomWithParticipants, error) {
	roomResp, err := r.GetRoomsByLiveKitServer(ctx)
	if err != nil {
		return nil, err
	}

	var roomWithParticipants []models.RoomWithParticipants
	for _, rm := range roomResp.Rooms {
		partResp, err := r.GetParticipantsByLiveKitServer(ctx, rm.Name)
		if err != nil {
			return nil, err
		}

		var Participants []models.Participant
		for _, p := range partResp.Participants {
			t := time.Unix(p.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
			Participants = append(Participants, models.Participant{
				Identity:   &p.Identity,
				JoinedAt:   &t,
				Name:       &p.Name,
				Attributes: &p.Attributes,
			})
		}

		roomId, err := uuid.Parse(rm.Name)
		if err != nil {
			return nil, err
		}

		var metadata *util.Metadata
		// rm.MetadataをJSON文字列としてunmarshalする
		err = json.Unmarshal([]byte(rm.Metadata), &metadata)
		if err != nil {
			return nil, err
		}

		roomWithParticipants = append(roomWithParticipants, models.RoomWithParticipants{
			Metadata:     &metadata.Metadata,
			IsWebinar:    &metadata.IsWebinar,
			RoomId:       roomId,
			Participants: Participants,
		})
	}

	return roomWithParticipants, nil
}
