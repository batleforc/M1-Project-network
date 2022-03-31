package controller

import (
	"M1/Network/API/app"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	redisIP = "10.8.3.1:6379"
)

// @Summary Upload file
// @Description Upload file
// @Accept  multipart/form-data
// @Param   myFile formData file true  "this is a test file"
// @Success 200 {string} string "ok"
// @Router /file/upload [post]
func (c *Controller) Upload(ctx *gin.Context) {
	w := ctx.Writer
	r := ctx.Request
	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)

	// Create a temporary file within our temp directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("temp", "*-"+handler.Filename)

	if err != nil {
		fmt.Println(err)
	}
	pathToFile := tempFile.Name()
	defer removeFile(pathToFile)
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a byte array
	fileBytes, err := ioutil.ReadAll(file)

	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	fmt.Fprintf(w, "Successfully Checked the File\n")
	if hashFileExist(tempFile) {
		fmt.Fprintf(w, "The file is genuine\n")
	} else {
		fmt.Fprintf(w, "This file isn't genuine \n")
	}
}

func hashFileExist(file *os.File) bool {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%x", hash.Sum(nil))

	return app.HashExist(string(hash.Sum(nil)), redisIP)
}

func removeFile(file string) {
	e := os.Remove(file)
	if e != nil {
		log.Fatal(e)
	}
}

func (c *Controller) Protected(ctx *gin.Context) {
	_, _, ok := ctx.Request.BasicAuth()
	if ok {
		ctx.Next()
		ctx.HTML(http.StatusOK, "./index.html", gin.H{})
		return
	}
	ctx.AbortWithStatus(401)
}
