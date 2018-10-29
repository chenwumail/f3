package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const maxUploadSize = 1024 * 1024 * 1024 // 1 GB
const uploadPath = "./files"

func main() {
	MakeDir(uploadPath)

	fs := http.FileServer(http.Dir(uploadPath))
	http.HandleFunc("/", uploadDownloadFileHandler(fs))
	http.HandleFunc("/index.htm", renderForm())
	http.HandleFunc("/index.html", renderForm())

	// http.Handle("/files/", http.StripPrefix("/files", fs))

	log.Print("Server started on localhost:80, use / for uploading files and /{fileName} for downloading")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func renderForm() http.HandlerFunc {
	html := `
	<head>
	</head>
	<body>
	<form action="/" method="post" enctype="multipart/form-data">
	<input type="file" name="file" />
    <input type="submit" value="upload" />
	</form>
	</body>
	`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})
}

func uploadDownloadFileHandler(fs http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fs.ServeHTTP(w, r)
			return
		}
		// validate file size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			renderError(w, "FILE_TOO_BIG", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("f")
		if err != nil {
			renderError(w, "INVALID_FILE", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := handler.Filename
		fmt.Println(filename)
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			renderError(w, "INVALID_FILE", http.StatusBadRequest)
			return
		}

		if len(filename) < 1 {
			filename = "tmp.dat"
		}
		newPath := filepath.Join(uploadPath, filename)
		// fmt.Printf("FileType: %s, File: %s\n", fileType, newPath)
		fmt.Printf("upload file: %s\n", newPath)

		// write file
		newFile, err := os.Create(newPath)
		if err != nil {
			renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
			return
		}
		defer newFile.Close() // idempotent, okay to call twice
		if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
			renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("SUCCESS\n"))
	})
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message + "\n"))
}

func MakeDir(path string) (result bool) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.IsDir() {
			return true
		} else {
			panic(path + " is reguler file, You need remove manully.")
			return false
		}
	}
	if os.IsNotExist(err) {
		err2 := os.Mkdir(path, os.ModePerm)
		if err2 != nil {
			fmt.Println("mkdir "+path+" failed, ", err2)
			return false
		}
		return true
	}
	fmt.Println("mkdir "+path+" failed, ", err)
	return false
}
