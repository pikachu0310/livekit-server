package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net/http"
)

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

	// 現在のルーム状態を送信
	err = h.broadcastRoomStateToSingleClient(conn)
	if err != nil {
		fmt.Printf("Failed to send room state to WebSocket client: %v", err)
		delete(h.Clients, conn)
		return err
	}

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

// 現在のルーム状態をWebSocketでブロードキャスト
func (h *Handler) broadcastRoomState() {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	// RoomStateをRoomWithParticipantsの形式に変換
	rooms := h.repo.RoomState

	// 全ルームの状態をJSONにシリアライズ
	roomStateJSON, err := json.Marshal(rooms)
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

func (h *Handler) broadcastRoomStateToSingleClient(client *websocket.Conn) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	// RoomStateをRoomWithParticipantsの形式に変換
	rooms := h.repo.RoomState

	// 全ルームの状態をJSONにシリアライズ
	roomStateJSON, err := json.Marshal(rooms)
	if err != nil {
		fmt.Printf("Failed to marshal room state: %v", err)
		return err
	}

	// 全クライアントに送信
	if err := client.WriteMessage(websocket.TextMessage, roomStateJSON); err != nil {
		fmt.Printf("Failed to send message to WebSocket client: %v", err)
		return err
	}

	return nil
}

// WebSocket用のアップグレーダ
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 全リクエスト許可 (必要に応じて制限)
	},
}
