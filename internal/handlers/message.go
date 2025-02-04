package handlers

import (
    "encoding/json"
    "net/http"
    "time"
    "go-simple-app/internal/database"
    "go-simple-app/internal/models"
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
