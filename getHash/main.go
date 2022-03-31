package main

import (
	"flag"

	"github.com/gin-gonic/gin"

	"M1/Network/API/controller"
	_ "M1/Network/API/docs"
	"M1/Network/API/fileVerification"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title uploadFile
// @version 1.0
// @license.name Apache 2.0
// @BasePath /api
func api() {
	r := gin.Default()
	r.LoadHTMLFiles("./index.html")

	c := controller.NewController()

	website := r.Group("/")
	{
		api := website.Group("/api")
		{
			file := api.Group("/file")
			{
				file.POST("/upload", c.Upload)
			}
		}
	}
	authorized := website.Group("/", gin.BasicAuth(gin.Accounts{
		"user1": "love",
	}))
	authorized.Any("/index.html", c.Protected)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8080")
}

func checkData() {
	fileVerification.VerifyFile()
}

func main() {
	apiFlag := flag.Bool("api", false, "a bool")
	flag.Parse()

	if *apiFlag {
		api()
	} else {
		checkData()
	}
}
