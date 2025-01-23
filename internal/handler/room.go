package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"net/http"
)

// LiveKitWebhook POST /webhook
func (h *Handler) LiveKitWebhook(c echo.Context) error {
	var event livekit.WebhookEvent
	if err := json.NewDecoder(c.Request().Body).Decode(&event); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Webhook payload",
		})
	}

	// ルーム状態を更新
	switch event.Event {
	case webhook.EventParticipantJoined:
		h.addParticipant(event.Room.Name, event.Participant.Identity)
	case webhook.EventParticipantLeft:
		h.removeParticipant(event.Room.Name, event.Participant.Identity)
	}

	// 全ルームの状態をWebSocketでブロードキャスト
	h.broadcastRoomState()

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// 参加者をルームに追加
func (h *Handler) addParticipant(room, identity string) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	h.RoomState[room] = append(h.RoomState[room], identity)
}

// 参加者をルームから削除
func (h *Handler) removeParticipant(room, identity string) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	participants := h.RoomState[room]
	for i, id := range participants {
		if id == identity {
			h.RoomState[room] = append(participants[:i], participants[i+1:]...)
			break
		}
	}
	if len(h.RoomState[room]) == 0 {
		delete(h.RoomState, room)
	}
}

// 現在のルーム状態をWebSocketでブロードキャスト
func (h *Handler) broadcastRoomState() {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	// 全ルームの状態をJSONにシリアライズ
	roomStateJSON, err := json.Marshal(h.RoomState)
	if err != nil {
		fmt.Printf("Failed to marshal room state: %v", err)
		return
	}

	// 全クライアントに送信
	for client := range h.Clients {
		if err := client.WriteMessage(websocket.TextMessage, roomStateJSON); err != nil {
			fmt.Printf("Failed to send message to WebSocket client: %v", err)
			client.Close()
			delete(h.Clients, client)
		}
	}
}

// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
func (h *Handler) InitializeRoomState() error {
	rsClient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.ApiKey, h.ApiSecret)

	// ルーム一覧を取得
	roomResp, err := rsClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list rooms: %v", err)
	}

	// 各ルームの参加者を取得し、状態を更新
	for _, room := range roomResp.Rooms {
		partResp, err := rsClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
			Room: room.Name,
		})
		if err != nil {
			fmt.Printf("Failed to list participants for room %s: %v", room.Name, err)
			continue
		}

		var participants []string
		for _, p := range partResp.Participants {
			participants = append(participants, p.Identity)
		}
		h.RoomState[room.Name] = participants
	}

	return nil
}

// WebSocket用のアップグレーダ
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 全リクエスト許可 (必要に応じて制限)
	},
}

// GetWs WebSocketエンドポイント: GET /ws
func (h *Handler) GetWs(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		fmt.Printf("Failed to upgrade to WebSocket: %v", err)
		return err
	}
	defer conn.Close()

	// クライアントを登録
	h.Mutex.Lock()
	h.Clients[conn] = true
	h.Mutex.Unlock()

	// WebSocket切断時にクライアントを削除
	defer func() {
		h.Mutex.Lock()
		delete(h.Clients, conn)
		h.Mutex.Unlock()
	}()

	// WebSocket接続維持
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("WebSocket connection error: %v", err)
			break
		}
	}
	return nil
}

// GetRooms GET /rooms
// 全ルームを取得し、それぞれの参加者一覧をまとめて返す。
func (h *Handler) GetRooms(ctx echo.Context) error {
	// 1) LiveKit RoomService クライアントを生成
	rsClient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.ApiKey, h.ApiSecret)

	// 2) ルーム一覧を取得
	roomResp, err := rsClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to list rooms: %v", err),
		})
	}

	// 3) 各ルームの参加者を取得し、まとめる
	type RoomInfo struct {
		RoomName     string   `json:"roomName"`
		Participants []string `json:"participants"`
	}
	var result []RoomInfo

	for _, rm := range roomResp.Rooms {
		// ルーム毎に参加者一覧を取得
		partResp, err := rsClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
			Room: rm.Name,
		})
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("failed to list participants in room %s: %v", rm.Name, err),
			})
		}

		// identityのリストだけ抽出
		var identities []string
		for _, p := range partResp.Participants {
			identities = append(identities, p.Identity)
		}

		result = append(result, RoomInfo{
			RoomName:     rm.Name,
			Participants: identities,
		})
	}

	// 4) JSONで返却
	return ctx.JSON(http.StatusOK, result)
}
