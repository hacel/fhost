package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
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
		fmt.Fprintf(w, `<a href="/files/">files</a>`)
		fmt.Fprintf(w, "<pre>yourserver</pre>\n")
	case "POST":
		r.URL.Path += "files/"
		fhost(w, r)
	}
}

func fhost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if len(r.URL.Path[len("/"):]) > 0 {
			http.ServeFile(w, r, path.Join("files/", r.URL.Path[1:]))
			return
		}
		fmt.Fprintf(w, "<div><pre>yourserver\n\n</pre></div>")
		fmt.Fprintf(w, "<pre>%s</pre>", `HTTP POST:
	curl -F'file=@yourfile.ext' https://yourserver/`)
		fmt.Fprintf(w, "<div>%s</div>", `<form enctype="multipart/form-data" method="post"><input type="file" id="file" name="file"><input type="submit"></form>`)

	case "POST":
		r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024)
		file, handler, err := r.FormFile("file")
		if err != nil {
			if err.Error() == "http: request body too large" {
				http.Error(w, "file too large (max 50MB)", http.StatusBadRequest)
				logger.Println("ERROR: ", err)
				return
			}
			http.Error(w, "failed to retrieve file in request", http.StatusBadRequest)
			logger.Println("ERROR: failed to retrieve file in request: ", err)
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
			http.Error(w, "server reached max number of files", http.StatusInternalServerError)
			logger.Println("ERROR: server reached max number of files: ")
			return
		}
		logger.Printf("uploaded file: %s (%.1fK) -> %s", handler.Filename, float64(handler.Size)/1024, tempFile.Name())

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			logger.Println("ERROR: could not read file: ", err)
			return
		}
		if _, err := tempFile.Write(fileBytes); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			logger.Println("ERROR: could not write to file: ", err)
			return
		}

		fmt.Fprintf(w, "https://%s%s%s\n", r.Host, r.RequestURI, filepath.Base(tempFile.Name()))
	}
}

func logHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		},
	)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	logf, err := os.OpenFile("serv.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()
	logger = log.New(logf, "", log.LstdFlags)
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		if err := os.Mkdir("files", 0755); err != nil {
			log.Fatal(err)
		}
	}

	router := http.NewServeMux()
	router.HandleFunc("/", fhost)
	logger.Fatal(http.ListenAndServe(":9990", logHandler(router)))
}
