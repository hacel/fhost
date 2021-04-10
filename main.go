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

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString() string {
	b := make([]byte, 3)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func fhost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if len(r.URL.Path) > 1 {
			http.ServeFile(w, r, filepath.Join("files", r.URL.Path[1:]))
			return
		}
		fmt.Fprintf(w, `HTTP POST:
	curl -F'file=@yourfile.ext' http://yourserver/`)

	case "POST":
		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Println("error: failed to retrieve file in request: ", err)
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
			log.Println("error: server reached max number of files")
			return
		}
		log.Printf("uploaded file: %s (%.1fK) -> %s\n", handler.Filename, float64(handler.Size)/1024, tempFile.Name())

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			log.Println("error: could not read file: ", err)
			return
		}
		if _, err := tempFile.Write(fileBytes); err != nil {
			log.Println("error: could not write to file: ", err)
			return
		}

		fmt.Fprintf(w, "%s%s\n", r.URL.Path, filepath.Base(tempFile.Name()))
	default:
		return
	}
}

func main() {
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		if err := os.Mkdir("files", 0755); err != nil {
			log.Fatal(err)
		}
	}
	http.HandleFunc("/", fhost)
	rand.Seed(time.Now().UnixNano())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
