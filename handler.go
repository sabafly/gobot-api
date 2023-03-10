/*
	Copyright (C) 2022-2023  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot-lib/logging"
)

// XXX: ライブラリにしてまとめるほうがいいか

// イベントを格納する構造体
type Event struct {
	Operation int             `json:"op"`
	Sequence  int64           `json:"s"`
	Type      string          `json:"t"`
	RawData   json.RawMessage `json:"d"`
	// Structはこのファイルのその他の型の一つを含む
	// TODO:よくわからん
	Struct any `json:"-"`
}

// ウェブソケットをハンドルする
type WebsocketHandler struct {
	WSMutex sync.Mutex
	Conn    map[*websocket.Conn]bool
	Seq     int64
}

// 新たなウェブソケットハンダラを生成する
func NewWebSocketHandler() *WebsocketHandler {
	return &WebsocketHandler{
		Conn: make(map[*websocket.Conn]bool),
	}
}

// ウェブソケット接続をハンドルする
func (h *WebsocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	logging.Debug("called")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if r.URL.Host != "api.sabafly.net" && r.URL.Host == "localhost:8686" {
				return false
			}
			// TODO: tokenか何かで認証したい
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Fatal("[内部] アップグレードできませんでした")
	}
	message := Event{
		Operation: 1,
		RawData:   json.RawMessage{},
	}

	b, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{"Hello World"})
	message.RawData = b

	b, _ = json.Marshal(message)

	if err := ws.WriteMessage(1, b); err != nil {
		logging.Error("[内部] WebSocket呼び出しに失敗 %s", err)
	}

	data := Event{}
	if err := ws.ReadJSON(&data); err != nil {
		logging.Error("[内部] JSON読み込みに失敗 %s", err)
	}
	logging.Info("[内部] 受信 %v", data)

	h.Conn[ws] = true

	go h.Listen(ws)
}

func (h *WebsocketHandler) Listen(c *websocket.Conn) {
	for {
		var event Event
		err := c.ReadJSON(&event)
		if err != nil {
			logging.Info("切断されました %s:%s", c.LocalAddr().Network(), c.LocalAddr().String())
			delete(h.Conn, c)
			return
		}

		// TODO: 受け取ったイベントを処理する
	}
}

// 渡された関数をゴルーチンですべてのウェブソケット接続で実行する
//
// TODO: もっといい方法があるはず
func (h *WebsocketHandler) Broadcast(f func(*websocket.Conn)) {
	for c, ok := range h.Conn {
		if !ok {
			continue
		}
		go func(c *websocket.Conn) {
			h.WSMutex.Lock()
			f(c)
			h.WSMutex.Unlock()
		}(c)
	}
}
