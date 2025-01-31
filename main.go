package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Message はJSONレスポンスの構造体です
type Message struct {
	ID        int64     `json:"id,omitempty"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
}

// User はユーザー情報を表す構造体です
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

var (
	db     *sql.DB
	store  *sessions.CookieStore
	appKey = []byte("your-secret-key") // 本番環境では環境変数などから読み込む
)

// データベースの初期化
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./messages.db")
	if err != nil {
		return err
	}

	// usersテーブルの作成
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);
	`
	_, err = db.Exec(createUsersTable)
	if err != nil {
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
	);
	`
	_, err = db.Exec(createMessagesTable)
	return err
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	message := Message{
		Text:      "JSONレスポンスのテストです",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// メッセージを保存するハンドラー
func saveMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	// ユーザーIDとユーザー名を取得
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int64)
	username, _ := session.Values["username"].(string)
	if !ok {
		http.Error(w, "ログインが必要です", http.StatusUnauthorized)
		return
	}

	// メッセージのデコード
	var requestBody struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Message構造体に変換
	var message Message
	message.Text = requestBody.Text

	// メッセージを保存
	result, err := db.Exec("INSERT INTO messages (text, timestamp, user_id) VALUES (?, ?, ?)",
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

// 全てのメッセージを取得するハンドラー
func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	// 認証チェック
	_, err := getUserID(r)
	if err != nil {
		http.Error(w, "ログインが必要です", http.StatusUnauthorized)
		return
	}

	// メッセージを取得
	rows, err := db.Query(`
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

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Text, &msg.Timestamp, &msg.UserID, &msg.Username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}


// メッセージを削除するハンドラー
func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "DELETEメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	// ユーザーIDを取得
	userID, err := getUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// URLからメッセージIDを取得
	messageID := r.URL.Query().Get("id")
	if messageID == "" {
		http.Error(w, "メッセージIDが指定されていません", http.StatusBadRequest)
		return
	}

	// メッセージの所有者を確認
	var messageUserID int64
	err = db.QueryRow("SELECT user_id FROM messages WHERE id = ?", messageID).Scan(&messageUserID)
	if err != nil {
		http.Error(w, "メッセージが見つかりません", http.StatusNotFound)
		return
	}

	if messageUserID != userID {
		http.Error(w, "このメッセージを削除する権限がありません", http.StatusForbidden)
		return
	}

	// メッセージを削除
	result, err := db.Exec("DELETE FROM messages WHERE id = ? AND user_id = ?", messageID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 削除された行数を確認
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "指定されたメッセージが見つかりません", http.StatusNotFound)
		return
	}

	// 成功レスポンス
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "メッセージを削除しました"})
}

// メッセージを編集するハンドラー
func updateMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "PUTメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	// ユーザーIDを取得
	userID, err := getUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// URLからメッセージIDを取得
	messageID := r.URL.Query().Get("id")
	if messageID == "" {
		http.Error(w, "メッセージIDが指定されていません", http.StatusBadRequest)
		return
	}

	// メッセージの所有者を確認
	var messageUserID int64
	err = db.QueryRow("SELECT user_id FROM messages WHERE id = ?", messageID).Scan(&messageUserID)
	if err != nil {
		http.Error(w, "メッセージが見つかりません", http.StatusNotFound)
		return
	}

	if messageUserID != userID {
		http.Error(w, "このメッセージを編集する権限がありません", http.StatusForbidden)
		return
	}

	// リクエストボディをデコード
	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// メッセージを更新
	result, err := db.Exec("UPDATE messages SET text = ? WHERE id = ? AND user_id = ?", message.Text, messageID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 更新された行数を確認
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "指定されたメッセージが見つかりません", http.StatusNotFound)
		return
	}

	// 更新後のメッセージを取得
	var updatedMessage Message
	err = db.QueryRow("SELECT id, text, timestamp FROM messages WHERE id = ?", messageID).Scan(
		&updatedMessage.ID, &updatedMessage.Text, &updatedMessage.Timestamp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 成功レスポンス
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

// セッションからユーザーIDを取得するヘルパー関数
func getUserID(r *http.Request) (int64, error) {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"].(int64)
	if !ok {
		return 0, fmt.Errorf("ログインが必要です")
	}
	return userID, nil
}

// ユーザー登録ハンドラー
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// パスワードのハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ユーザーの保存
	result, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)",
		user.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, "ユーザー名が既に使用されています", http.StatusBadRequest)
		return
	}

	userID, _ := result.LastInsertId()
	user.ID = userID
	user.Password = "" // パスワードをレスポンスから除外

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ログインハンドラー
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POSTメソッドのみ許可されています", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ユーザーの取得
	var user User
	err := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?",
		credentials.Username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		http.Error(w, "ユーザー名またはパスワードが正しくありません", http.StatusUnauthorized)
		return
	}

	// パスワードの検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		http.Error(w, "ユーザー名またはパスワードが正しくありません", http.StatusUnauthorized)
		return
	}

	// セッションの作成
	session, _ := store.Get(r, "session-name")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Save(r, w)

	user.Password = "" // パスワードをレスポンスから除外
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ログアウトハンドラー
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "ログアウトしました"})
}

func main() {
	// データベースの初期化
	if err := initDB(); err != nil {
		log.Fatal("データベースの初期化に失敗しました:", err)
	}
	defer db.Close()

	// セッションストアの初期化
	store = sessions.NewCookieStore(appKey)

	http.HandleFunc("/", handler)
	http.HandleFunc("/register", registerHandler)             // POST: ユーザー登録
	http.HandleFunc("/login", loginHandler)                   // POST: ログイン
	http.HandleFunc("/logout", logoutHandler)                 // POST: ログアウト
	http.HandleFunc("/messages", getMessagesHandler)          // GET: メッセージ一覧の取得
	http.HandleFunc("/messages/save", saveMessageHandler)     // POST: メッセージの保存
	http.HandleFunc("/messages/delete", deleteMessageHandler) // DELETE: メッセージの削除
	http.HandleFunc("/messages/update", updateMessageHandler) // PUT: メッセージの編集

	// 静的ファイルの提供
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("サーバーを起動します。http://localhost:8080 にアクセスしてください。")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
