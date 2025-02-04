package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"kodik_anime_dl/pkgs/config"
	"kodik_anime_dl/pkgs/controllers"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/public/*
var staticFiles embed.FS

//go:embed web/templates/*
var templateFiles embed.FS

func main() {
	var err error
	if err = config.LoadEnv(); err != nil {
		log.Fatalln(err)
	}

	gin.SetMode(config.GinMode)
	staticFilesSub, err := fs.Sub(staticFiles, "web/public")

	if err != nil {
		log.Fatalln(err)
	}

	r := gin.Default()
	r.SetHTMLTemplate(template.Must(template.New("").ParseFS(templateFiles, "web/templates/*")))
	r.StaticFS("/public", http.FS(staticFilesSub))

	r.GET("/", controllers.Home)
	r.POST("/search", controllers.Search)
	r.POST("/translation", controllers.Translation)
	r.GET("/translations", controllers.Translations)
	r.POST("/episode", controllers.Episode)
	r.GET("/episodes", controllers.Episodes)
	r.GET("/download", controllers.Download)
	r.GET("/download/status", controllers.DownloadStatus)

	fmt.Printf("Сервер запущен по адресу: http://%s\n", config.Addr)

	if err = r.Run(config.Addr); err != nil {
		log.Fatalln(err)
		return
	}
}
