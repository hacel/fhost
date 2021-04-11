package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	logger *log.Logger
)

func randString() string {
	b := make([]byte, 3)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func index(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "yourserver\n")
	case "POST":
		r.URL.Path += "files/"
		fhost(w, r)
	}
}

func fhost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if len(r.URL.Path[len("/files/"):]) > 0 {
			http.ServeFile(w, r, r.URL.Path[1:])
			return
		}
		fmt.Fprintf(w, `HTTP POST:
	curl -F'file=@yourfile.ext' http://yourserver/`)

	case "POST":
		file, handler, err := r.FormFile("file")
		if err != nil {
			logger.Println("error: failed to retrieve file in request: ", err)
			return
		}
		defer file.Close()

		var tempFile *os.File
		for i := 0; i < 1000; i++ {
			name := filepath.Join("files/", fmt.Sprintf("%s%s", randString(), filepath.Ext(handler.Filename)))
			tempFile, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
			if os.IsExist(err) {
				continue
			}
			break
		}
		defer tempFile.Close()
		if tempFile == nil {
			logger.Println("error: server reached max number of files")
			return
		}
		logger.Printf("uploaded file: %s (%.1fK) -> %s", handler.Filename, float64(handler.Size)/1024, tempFile.Name())

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			logger.Println("error: could not read file: ", err)
			return
		}
		if _, err := tempFile.Write(fileBytes); err != nil {
			logger.Println("error: could not write to file: ", err)
			return
		}

		fmt.Fprintf(w, "%s%s\n", r.URL.Path, filepath.Base(tempFile.Name()))
	}
}

func logHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			logger.Println(r.Method, r.RemoteAddr)
			next.ServeHTTP(w, r)
		},
	)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		if err := os.Mkdir("files", 0755); err != nil {
			log.Fatal(err)
		}
	}
	logf, err := os.OpenFile("serv.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()
	logger = log.New(logf, "", log.LstdFlags)

	router := http.NewServeMux()
	router.HandleFunc("/", index)
	router.HandleFunc("/files/", fhost)
	log.Fatal(http.ListenAndServe(":8080", logHandler(router)))
}
