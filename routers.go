package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"io"
	"os"
)

func initEngine() *gin.Engine {
	e := gin.New()
	if !C.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	// logger middleware
	if C.Log {
		gin.DisableConsoleColor()
		fp, err := os.OpenFile(C.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", C.LogPath)
			os.Exit(1)
		}
		gin.DefaultWriter = io.MultiWriter(fp)
	} // else gin.DefaultWriter = os.Stdout
	e.Use(gin.Logger())
	e.Use(gin.Recovery())
	e.Use(cors.Default())

	// static file
	e.Use(static.Serve("/", static.LocalFile(C.PublicPath, true)))
	e.Use(static.Serve("/data", static.LocalFile(C.DataPath, true)))

	// api router
	api := e.Group("/api")
	{
		api.GET("/ping", ping)

		storage := api.Group("/storage")
		{
			storage.GET("/definition", getEventDefinition)
			storage.GET("/file_system", getDataStorageFiles)
		}

		api.GET("/event", getEvent)
		api.POST("/event", postEvent)
		api.DELETE("/event", deleteEvent)

		api.GET("/rating", getRating)
		api.POST("/rating", postRating)
		api.DELETE("/rating", deleteRating)

		storage.GET("/video", getVideoList)
		storage.PUT("/video", putVideo)

		storage.GET("/trajectory", getTrajectoryList)
		storage.PUT("/trajectory", putTrajectory)
	}

	fmt.Println("Router initialized.")
	return e
}