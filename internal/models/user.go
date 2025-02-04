package models

// User はユーザー情報を表す構造体です
type User struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Password string `json:"password,omitempty"`
}

// Credentials はログイン情報を表す構造体です
type Credentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
}
