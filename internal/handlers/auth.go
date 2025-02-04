package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
    "go-simple-app/internal/database"
    "go-simple-app/internal/models"
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
