package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-simple-app/internal/database"
	"go-simple-app/internal/handlers"
	"go-simple-app/internal/middleware"
)

var appKey = []byte("your-secret-key") // 本番環境では環境変数などから読み込む

func getProjectRoot() (string, error) {
	// 現在の作業ディレクトリを取得
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// cmd/server から プロジェクトルートへ
	projectRoot := filepath.Join(workDir, "../..")
	return projectRoot, nil
}

func main() {
	// プロジェクトルートの取得
	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatal("プロジェクトルートの取得に失敗しました:", err)
	}

	// データベースの初期化
	dbPath := filepath.Join(projectRoot, "var", "messages.db")
	// データベースディレクトリの作成
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatal("データベースディレクトリの作成に失敗しました:", err)
	}

	if err := database.InitDB(dbPath); err != nil {
		log.Fatal("データベースの初期化に失敗しました:", err)
	}
	defer database.DB.Close()

	// セッションストアの初期化
	handlers.InitSession(appKey)

	// 静的ファイルの設定
	staticDir := filepath.Join(projectRoot, "static")
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// ルーティング
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/messages", middleware.AuthRequired(handlers.GetMessagesHandler))
	http.HandleFunc("/messages/save", middleware.AuthRequired(handlers.SaveMessageHandler))
	http.HandleFunc("/messages/delete", middleware.AuthRequired(handlers.DeleteMessageHandler))
	http.HandleFunc("/messages/update", middleware.AuthRequired(handlers.UpdateMessageHandler))

	// サーバー起動
	port := "8080"
	fmt.Printf("サーバーを起動します。http://localhost:%s にアクセスしてください。\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
