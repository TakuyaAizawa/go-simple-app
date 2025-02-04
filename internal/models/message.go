package models

import "time"

// Message はJSONレスポンスの構造体です
type Message struct {
    ID        int64     `json:"id,omitempty"`
    Text      string    `json:"text"`
    Timestamp time.Time `json:"timestamp"`
    UserID    int64     `json:"user_id"`
    Username  string    `json:"username"`
}
