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

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot-lib/caches"
	"github.com/sabafly/gobot-lib/logging"
	"github.com/sabafly/gobot-lib/requests"
)

//TODO: 構造体にする

// ----------------------------------------------------------------
// ギルド関連
// ----------------------------------------------------------------

var createdGuilds *caches.CacheManager[struct{ ID string }] = caches.NewCacheManager[struct{ ID string }](nil)

// ギルド作成イベントを処理する
// 受け取ったギルドをキャッシュに登録しクライアントにステータス更新イベントを送る
func (h *WebsocketHandler) HandlerGuildCreate(w http.ResponseWriter, r *http.Request) {
	// データを取り出す
	guildCreate := struct{ ID string }{}
	if err := requests.Unmarshal(r, &guildCreate); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	// キャッシュに保存
	createdGuilds.Set(guildCreate.ID, guildCreate)

	h.Broadcast(func(ws *websocket.Conn) {
		statusUpdate := struct{ Servers int }{Servers: createdGuilds.Len()}
		b, _ := json.Marshal(statusUpdate) //TODO: エラーハンドリング
		logging.Debug("[内部] ステータス更新イベント")
		err := ws.WriteJSON(Event{Operation: 8, Sequence: h.Seq + 1, Type: "STATUS_UPDATE", RawData: b})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
			w.WriteHeader(500)
			err := json.NewEncoder(w).Encode(map[string]any{"status": "500 Server Error"})
			if err != nil {
				logging.Error("応答に失敗 %s", err)
			}
		}
		h.Seq++
	})

	err := json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

// ギルド削除イベントを受け取る
// 受け取ったギルドをキャッシュから削除しクライアントにステータス更新イベントを送る
func (h *WebsocketHandler) HandlerGuildDelete(w http.ResponseWriter, r *http.Request) {
	// データを取り出す
	guildDelete := discordgo.GuildDelete{}
	if err := requests.Unmarshal(r, &guildDelete); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request", "error": err})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	// キャッシュを削除
	createdGuilds.Delete(guildDelete.ID)

	h.Broadcast(func(ws *websocket.Conn) {
		statusUpdate := struct{ Servers int }{Servers: createdGuilds.Len()}
		b, _ := json.Marshal(statusUpdate) //TODO: エラーハンドリング
		logging.Debug("[内部] ステータス更新イベント")
		err := ws.WriteJSON(Event{Operation: 8, Sequence: h.Seq + 1, Type: "STATUS_UPDATE", RawData: b})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		h.Seq++
	})

	err := json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

// ギルドフィーチャー関連

var featureCache = caches.NewCacheManager[[]GuildFeature](nil)

func HandlerGuildFeaturePost(w http.ResponseWriter, r *http.Request) {
	//データ取り出す
	guildFeature := GuildFeature{}
	if err := requests.Unmarshal(r, &guildFeature); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request", "error": err})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	err := db.Save(&guildFeature)
	if err != nil {
		logging.Error("[内部] [REST] データベースへの書き込みに失敗 %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	caches.Append(featureCache, guildFeature.GuildID, guildFeature)

	err = json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

func HandlerGuildFeatureDelete(w http.ResponseWriter, r *http.Request) {
	//データ取り出す
	guildFeature := GuildFeature{}
	if err := requests.Unmarshal(r, &guildFeature); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request", "error": err})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	guildFeatures := []GuildFeature{}
	db.Find(&guildFeatures, "guild_id = ?", guildFeature.GuildID)

	for _, gf := range guildFeatures {
		if gf.TargetID == guildFeature.TargetID && gf.FeatureID == guildFeature.FeatureID {
			err := db.Delete(&guildFeature, "id = ?", gf.ID)
			if err != nil {
				logging.Error("[内部] [REST] データベースへの書き込みに失敗 %s", err)
				w.WriteHeader(400)
				err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
				if err != nil {
					logging.Error("応答に失敗 %s", err)
				}
				return
			}
		}
	}

	features, err := featureCache.Get(guildFeature.GuildID)
	if err != nil {
		logging.Error("[内部] [REST] キャッシュへの書き込みに失敗 %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}
	for i, gf := range features {
		if gf.TargetID == guildFeature.TargetID {
			features = features[:i+copy(features[i:], features[i+1:])]
		}
	}
	featureCache.Set(guildFeature.GuildID, features)

	err = json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

func HandlerGuildFeatureGet(w http.ResponseWriter, r *http.Request) {
	//データ取り出す
	guildFeature := GuildFeature{}
	if err := requests.Unmarshal(r, &guildFeature); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request", "error": err})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	_, err := featureCache.Get(guildFeature.GuildID)
	if err != nil {
		err := db.First(&guildFeature)
		if err != nil {
			logging.Error("[内部] [REST] データベースへの書き込みに失敗 %s", err)
			w.WriteHeader(404)
			err := json.NewEncoder(w).Encode(map[string]any{"status": "404 Not Found"})
			if err != nil {
				logging.Error("応答に失敗 %s", err)
			}
			return
		}
		caches.Append(featureCache, guildFeature.GuildID, guildFeature)
	}

	err = json.NewEncoder(w).Encode(struct{ Enabled bool }{Enabled: true})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

// ----------------------------------------------------------------
// メッセージ関連
// ----------------------------------------------------------------

var messageLog = caches.NewCacheManager[[]MessageLog](nil)

func (h *WebsocketHandler) HandlerMessageCreate(w http.ResponseWriter, r *http.Request) {
	// データを取り出す
	messageCreate := &discordgo.MessageCreate{}
	if err := requests.Unmarshal(r, messageCreate); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	author := messageCreate.Author
	if author == nil {
		logging.Error("[内部] [REST] 送信者が不明")
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	if messageCreate.WebhookID != "" {
		err := json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	log := MessageLog{
		Model: Model{
			ID: messageCreate.ID,
		},
		GuildID:   messageCreate.GuildID,
		ChannelID: messageCreate.ChannelID,
		UserID:    messageCreate.Author.ID,
		Content:   messageCreate.Content,
		Bot:       messageCreate.Author.Bot,
	}

	err := db.Create(&log)
	if err != nil {
		logging.Error("[内部] [REST] データベースへの書き込みに失敗 %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	caches.Append(messageLog, messageCreate.Author.ID, log)

	err = json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

// ----------------------------------------------------------------
// 統計
// ----------------------------------------------------------------

func (h *WebsocketHandler) HandlerStaticsUserMessage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !query.Has("user") || !query.Has("guild") {
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		return
	}

	logs := []MessageLog{}

	err := db.Find(&logs, "user_id = ?", query.Get("user"))
	if err != nil {
		logging.Warning("見つからなかった %s", err)
	}

	res := []MessageLog{}

	for _, v := range logs {
		if v.GuildID == query.Get("guild") {
			res = append(res, v)
		}
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}
