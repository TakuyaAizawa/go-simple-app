#!/bin/bash

# エラーが発生したら停止
set -e

# 現在のディレクトリ名を取得
CURRENT_DIR=$(basename $(pwd))

# バックアップディレクトリの作成
echo "既存のファイルをバックアップしています..."
BACKUP_DIR="../${CURRENT_DIR}_backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp -r * "$BACKUP_DIR/"

# 既存のファイルを削除（バックアップは残す）
echo "既存のファイルをクリーンアップしています..."
rm -f *.go
rm -f go.mod go.sum

# 新しいディレクトリ構造の作成
echo "新しいディレクトリ構造を作成しています..."
mkdir -p cmd/server
mkdir -p internal/{models,database,handlers,middleware}
mkdir -p static

# Go モジュールの初期化
echo "Go モジュールを初期化しています..."
go mod init ${CURRENT_DIR}

# 各ファイルの作成
echo "ソースファイルを作成しています..."

# models/message.go
cat > internal/models/message.go << 'EOF'
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
EOF

# models/user.go
cat > internal/models/user.go << 'EOF'
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
EOF

# database/db.go
cat > internal/database/db.go << 'EOF'
package database

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB はデータベースの初期化を行います
func InitDB(dbPath string) error {
    var err error
    DB, err = sql.Open("sqlite3", dbPath)
    if err != nil {
        return err
    }

    // usersテーブルの作成
    createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL
    );`
    if _, err := DB.Exec(createUsersTable); err != nil {
        return err
    }

    // messagesテーブルの作成
    createMessagesTable := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        text TEXT NOT NULL,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users(id)
    );`
    return DB.Exec(createMessagesTable)
}
EOF

# handlers/auth.go
cat > internal/handlers/auth.go << 'EOF'
package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
    "github.com/yourusername/chat-app/internal/database"
    "github.com/yourusername/chat-app/internal/models"
)

var store *sessions.CookieStore

func InitSession(key []byte) {
    store = sessions.NewCookieStore(key)
}

// GetUserID セッションからユーザーIDを取得
func GetUserID(r *http.Request) (int64, error) {
    session, _ := store.Get(r, "session-name")
    userID, ok := session.Values["user_id"].(int64)
    if !ok {
        return 0, fmt.Errorf("ログインが必要です")
    }
    return userID, nil
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
        return
    }

    var user models.User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    result, err := database.DB.Exec("INSERT INTO users (username, password) VALUES (?, ?)",
        user.Username, string(hashedPassword))
    if err != nil {
        http.Error(w, "ユーザー名が既に使用されています", http.StatusBadRequest)
        return
    }

    user.ID, _ = result.LastInsertId()
    user.Password = ""

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
        return
    }

    var credentials models.Credentials
    if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var user models.User
    err := database.DB.QueryRow("SELECT id, username, password FROM users WHERE username = ?",
        credentials.Username).Scan(&user.ID, &user.Username, &user.Password)
    if err != nil {
        http.Error(w, "ユーザー名またはパスワードが正しくありません", http.StatusUnauthorized)
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
        http.Error(w, "ユーザー名またはパスワードが正しくありません", http.StatusUnauthorized)
        return
    }

    session, _ := store.Get(r, "session-name")
    session.Values["user_id"] = user.ID
    session.Values["username"] = user.Username
    session.Save(r, w)

    user.Password = ""
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "session-name")
    session.Values = make(map[interface{}]interface{})
    session.Save(r, w)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "ログアウトしました"})
}
EOF

# handlers/message.go
cat > internal/handlers/message.go << 'EOF'
package handlers

import (
    "encoding/json"
    "net/http"
    "time"
    "github.com/yourusername/chat-app/internal/database"
    "github.com/yourusername/chat-app/internal/models"
)

func SaveMessageHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
        return
    }

    session, _ := store.Get(r, "session-name")
    userID, ok := session.Values["user_id"].(int64)
    username, _ := session.Values["username"].(string)
    if !ok {
        http.Error(w, "ログインが必要です", http.StatusUnauthorized)
        return
    }

    var requestBody struct {
        Text string `json:"text"`
    }
    if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var message models.Message
    message.Text = requestBody.Text

    result, err := database.DB.Exec("INSERT INTO messages (text, timestamp, user_id) VALUES (?, ?, ?)",
        message.Text, time.Now(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    id, _ := result.LastInsertId()
    message.ID = id
    message.UserID = userID
    message.Username = username
    message.Timestamp = time.Now()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(message)
}

func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := database.DB.Query(`
        SELECT m.id, m.text, m.timestamp, m.user_id, u.username
        FROM messages m
        JOIN users u ON m.user_id = u.id
        ORDER BY m.timestamp DESC
    `)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var messages []models.Message
    for rows.Next() {
        var msg models.Message
        if err := rows.Scan(&msg.ID, &msg.Text, &msg.Timestamp, &msg.UserID, &msg.Username); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        messages = append(messages, msg)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(messages)
}

func DeleteMessageHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "DELETEメソッドのみ許可されています", http.StatusMethodNotAllowed)
        return
    }

    userID, err := GetUserID(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    messageID := r.URL.Query().Get("id")
    if messageID == "" {
        http.Error(w, "メッセージIDが指定されていません", http.StatusBadRequest)
        return
    }

    var messageUserID int64
    err = database.DB.QueryRow("SELECT user_id FROM messages WHERE id = ?", messageID).Scan(&messageUserID)
    if err != nil {
        http.Error(w, "メッセージが見つかりません", http.StatusNotFound)
        return
    }

    if messageUserID != userID {
        http.Error(w, "このメッセージを削除する権限がありません", http.StatusForbidden)
        return
    }

    result, err := database.DB.Exec("DELETE FROM messages WHERE id = ? AND user_id = ?", messageID, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "指定されたメッセージが見つかりません", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "メッセージを削除しました"})
}

func UpdateMessageHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut {
        http.Error(w, "PUTメソッドのみ許可されています", http.StatusMethodNotAllowed)
        return
    }

    userID, err := GetUserID(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    messageID := r.URL.Query().Get("id")
    if messageID == "" {
        http.Error(w, "メッセージIDが指定されていません", http.StatusBadRequest)
        return
    }

    var messageUserID int64
    err = database.DB.QueryRow("SELECT user_id FROM messages WHERE id = ?", messageID).Scan(&messageUserID)
    if err != nil {
        http.Error(w, "メッセージが見つかりません", http.StatusNotFound)
        return
    }

    if messageUserID != userID {
        http.Error(w, "このメッセージを編集する権限がありません", http.StatusForbidden)
        return
    }

    var message models.Message
    if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    result, err := database.DB.Exec("UPDATE messages SET text = ? WHERE id = ? AND user_id = ?", 
        message.Text, messageID, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "指定されたメッセージが見つかりません", http.StatusNotFound)
        return
    }

    err = database.DB.QueryRow("SELECT id, text, timestamp FROM messages WHERE id = ?", messageID).Scan(
        &message.ID, &message.Text, &message.Timestamp)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(message)
}
EOF

# middleware/auth.go
cat > internal/middleware/auth.go << 'EOF'
package middleware

import (
    "net/http"
    "github.com/yourusername/chat-app/internal/handlers"
)

func AuthRequired(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
        _, err := handlers.GetUserID(r)
        if err != nil {
            http.Error(w, "ログインが必要です", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
EOF

# cmd/server/main.go
cat > cmd/server/main.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "net/http"
    "github.com/yourusername/chat-app/internal/database"
    "github.com/yourusername/chat-app/internal/handlers"
    "github.com/yourusername/chat-app/internal/middleware"
)

var appKey = []byte("your-secret-key") // 本番環境では環境変数などから読み込む

func main() {
    // データベースの初期化
    if err := database.InitDB("./messages.db"); err != nil {
        log.Fatal("データベースの初期化に失敗しました:", err)
    }
    defer database.DB.Close()

    // セッションストアの初期化
    handlers.InitSession(appKey)

    // ルーティング
    http.HandleFunc("/", serveIndex)
    http.HandleFunc("/register", handlers.RegisterHandler)
    http.HandleFunc("/login", handlers.LoginHandler)
    http.HandleFunc("/logout", handlers.LogoutHandler)
    http.HandleFunc("/messages", middleware.AuthRequired(handlers.GetMessagesHandler))
    http.HandleFunc("/messages/save", middleware.AuthRequired(handlers.SaveMessageHandler))
    http.HandleFunc("/messages/delete", middleware.AuthRequired(handlers.DeleteMessageHandler))
    http.HandleFunc("/messages/update", middleware.AuthRequired(handlers.UpdateMessageHandler))

    // 静的ファイルの提供
    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    fmt.Println("サーバーを起動します。http://localhost:8080 にアクセスしてください。")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    http.ServeFile(w, r, "static/index.html")
}
EOF

# 既存のindex.htmlファイルを移動（存在する場合）
if [ -f "static/index.html" ]; then
    mv static/index.html "$BACKUP_DIR/static/"
fi

# 依存関係の追加
echo "依存関係を追加しています..."
go get github.com/mattn/go-sqlite3
go get github.com/gorilla/sessions
go get golang.org/x/crypto/bcrypt

# モジュール名の置換
echo "モジュール名を置換しています..."
find . -type f -name "*.go" -exec sed -i "s/github.com\/yourusername\/chat-app/${CURRENT_DIR}/g" {} \;

echo "プロジェクトの再構築が完了しました!"
echo "バックアップは ${BACKUP_DIR} に保存されています"
echo ""
echo "次のコマンドを実行してプロジェクトをビルドできます:"
echo "go mod tidy"
echo "go build -o chat-app cmd/server/main.go"

# 実行権限を付与
chmod +x cmd/server/main.go

EOF