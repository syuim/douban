package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	robfigcron "github.com/robfig/cron/v3"

	appcron "stremio-addon-douban/internal/cron"
	"stremio-addon-douban/internal/db"
	"stremio-addon-douban/internal/handler"
	"stremio-addon-douban/internal/middleware"
)

func main() {
	godotenv.Load()

	if _, err := db.GetDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.CORS)

	// catalog 路由（/catalog/{type}/{catalogID}.json）— 含点路径段，手动解析
	r.NotFound(handler.CatalogResourceHandler)

	// cron：每日凌晨 2:00 全量拉取刷新缓存
	c := robfigcron.New()
	c.AddFunc("0 2 * * *", func() {
		appcron.RunScheduledTask()
	})
	c.Start()

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
