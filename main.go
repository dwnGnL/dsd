package main

import(
	// "net/http"
	"chat/routs"
	"chat/db"
	"chat/logs"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	// "github.com/gorilla/sessions"
	// "encoding/gob"
	// "os"
	// "io"
	"chat/models"
	// "log"
	"chat/utils"
)


func main()  {
	config := utils.ReadConfig()
	logs.LogConfig()
	r:=gin.Default()
	logger := logrus.New()
	db.Open(config.DbURI, logger)
	// r.LoadHTMLGlob("templates/*")
	// session:=r.Group("/ses")
	// {
	// 	session.GET("/",routs.Index)
	// 	session.GET("/logout", routs.Logout)
	// 	session.POST("/checkLog", routs.CheckLog)
	// }
	routs.Routs(r) //Start routs
	r.Run(":"+config.Port)
	var history models.History
	db := db.GetDB()
	defer db.Delete(&history)
}