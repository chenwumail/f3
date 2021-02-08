package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const UPLOAD_SIZE_1GB = 1024 * 1024 * 1024
const BUFFER_SIZE_1MB = 1048576

var (
	help           bool
	maxUploadSize  int64
	uploadPath     string
	enableSubDir   bool
	maxExpireHours int64
	host           string
	port           int
)

func init() {
	flag.StringVar(&uploadPath, "d", "./files", "diretory for upload file storage")
	flag.Int64Var(&maxUploadSize, "l", UPLOAD_SIZE_1GB, "upload file bytes limit")
	flag.Int64Var(&maxExpireHours, "e", 0, "hours to keep fo upload files, 0 means keep it for ever")
	flag.StringVar(&host, "b", "0.0.0.0", "bind ip address")
	flag.IntVar(&port, "p", 80, "bind port")
	flag.BoolVar(&enableSubDir, "s", false, "enable sub directory")
	flag.BoolVar(&help, "h", false, "this help message")
}

func main() {
	flag.Parse()
	if help {
		log.Println("f3 [-b bind_host] [-p port] [-e expire_hours] [-l limit_bytes] [-d dir_upload] [-s] [-h]")
		flag.Usage()
		os.Exit(-1)
	}
	hostPort := fmt.Sprintf("%s:%d", host, port)

	MakeDir(uploadPath)

	fs := http.FileServer(http.Dir(uploadPath))
	http.HandleFunc("/", uploadDownloadFileHandler(fs))
	http.HandleFunc("/upload", renderForm())
	http.HandleFunc("/upload.html", renderForm())

	if maxExpireHours > 0 {
		log.Printf("Upload files while be remove automaticaly after %d hours.", maxExpireHours)
		go cleanThread()
	}

	log.Print("Server started on " + hostPort + ", use / for uploading files and /{fileName} for downloading")
	features := ""
	if maxExpireHours <= 0 {
		features += " -expire-clean "
	}
	if enableSubDir {
		features += " +enable-sub-dir "
	}
	log.Print("features: " + features)
	log.Fatal(http.ListenAndServe(hostPort, nil))
}

func cleanThread() {
	for {
		log.Printf("scan expires in [%s] every %d hours ...\n", uploadPath, maxExpireHours)
		removeExpireFiles(uploadPath)
		time.Sleep(1 * time.Hour)
	}
}

func removeExpireFiles(dirName string) []string {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Println(err)
	}
	var fileList []string
	for _, file := range files {
		modTime := file.ModTime()
		now := time.Now()
		subTime := now.Sub(modTime)
		if subTime.Minutes() > float64(maxExpireHours)*60 {
			filename := dirName + string(os.PathSeparator) + file.Name()
			log.Println(filename + " expires, deleted.")
			if err := os.Remove(filename); err != nil {
				log.Println(err)
			}
		}
		fileList = append(fileList, dirName+string(os.PathSeparator)+file.Name())
		/*
			recursive scan directory
			if file.IsDir() {
				fileList = append(fileList, scanDir(dirName+string(os.PathSeparator)+file.Name())...)
			}
		*/
	}
	return fileList
}

func htmlForm() string {
	html := `
	<!DOCTYPE html>
	<html>
	
	<head>
		<title></title>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link href="//static.chimeroi.com/finance/bootstrap/3.3.6/css/bootstrap.min.css" rel="stylesheet">
		<style>
			html,
			body{
				height:100%;
			}
			body{
				background-color:#4998e5;
			}
			h1{
				color: #fff;
				margin-bottom: 80px;
			}
			.wrap{
				position: relative;
				transform: translateY(-50%);
				top: 50%;
				text-align: center;
			}
		</style>
	</head>
	
	<body>
		<div class="wrap" style="text-align: center;">
			<h1 style="display: none;"></h1>
			<form action="/" method="post" enctype="multipart/form-data" style="position: relative;display:inline-block;margin-bottom:40px;">
				<button class="btn btn-default btn-lg" type="submit" style="color:#4998e5;"><span class="glyphicon glyphicon-open" style="margin-right:5px;position: relative;"></span>Upload</button>
				<input id="btn-upload" type="file" name="f" style="position:absolute;top:0;left:0;right:0;bottom:0;width:100%;height: 100%;opacity:0;cursor: pointer;" />
			</form>
			<p style="color:#fff;">&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Upload: <code style="color:#666;">curl <hostname:port|domain-name> -T test.tgz </code> </p>
			<p style="color:#fff;">Download: <code style="color:#666;">curl <hostname:port|domain-name>/test.tgz -O </code> </p>
		</div>
		<script src="//static.chimeroi.com/finance/jquery/1.12.1/jquery.min.js"></script>
		<script>
			var m = location.href.match(/result=([^&]+)/i);
			if (m){
				var msg = decodeURIComponent(m[1]);
				$('h1').text(msg).show();
			}
			$('#btn-upload').on('change', function(e){
				document.forms[0].submit();
			});
		</script>
	</body>
	
	</html>
	`
	return html
}
func renderForm() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlForm()))
	})
}

func redenIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlForm()))
}

func rederSuccess(w http.ResponseWriter, r *http.Request, message string) {
	url := "/?result=" + message
	http.Redirect(w, r, url, http.StatusFound)
}

func renderError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	url := "/?result=" + message
	http.Redirect(w, r, url, http.StatusFound)
}

func rederSuccessText(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message + "\n"))
}

func renderErrorText(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message + "\n"))
}

func writeBytesToFile(filenameRaw string, uploadFileStream io.Reader) (message string, status bool) {
	filename := filenameRaw
	log.Println(filename)
	if len(filename) < 1 {
		filename = "tmp.dat"
	}
	newPath := filepath.Join(uploadPath, filename)
	log.Printf("upload file: %s\n", newPath)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		return "CANT_WRITE_FILE", false
	}
	defer newFile.Close()

	reader := bufio.NewReader(uploadFileStream)
	writer := bufio.NewWriter(newFile)

	buf := make([]byte, BUFFER_SIZE_1MB)
	sum := 0
	for {
		nBytesRead, readErr := reader.Read(buf)
		if readErr != nil && readErr != io.EOF {
			fmt.Printf("read error, last %d bytes read, total %d bytes read,  err: %v\n", nBytesRead, sum, readErr)
			return "CANT_READ_FILE", false
		}
		mBytesWrite, writeErr := writer.Write(buf[0:nBytesRead])
		if writeErr != nil {
			fmt.Printf("write error, last %d bytes write, total write %d bytes, err: %v\n", mBytesWrite, sum+mBytesWrite, writeErr)
			return "CANT_WRITE_FILE", false
		}
		if nBytesRead < BUFFER_SIZE_1MB || readErr == io.EOF {
			fmt.Printf("finished last %d bytes read, buffer size: %d, total %d bytes read,  err: %v\n", nBytesRead, BUFFER_SIZE_1MB, sum+nBytesRead, readErr)
			writer.Flush()
			break
		}
		sum += nBytesRead
	}

	return "SUCCESS", true
}

func uploadDownloadFileHandler(fs http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.URL.Path == "/" {
				redenIndex(w, r)
				return
			}
			fs.ServeHTTP(w, r)
			return
		}

		if r.Method == "PUT" {
			filename := r.URL.Path
			paths, _ := filepath.Split(r.URL.Path)
			subUploadPath := uploadPath + paths

			if enableSubDir && paths != "/" {
				MakeDir(subUploadPath)
				log.Print("create sub directory: " + subUploadPath)
			}

			msg, result := writeBytesToFile(filename, r.Body)
			if result {
				rederSuccessText(w, r, "Success. Download by: curl "+r.Host+filename+" -O")
			} else {
				renderErrorText(w, r, msg, http.StatusBadRequest)
			}
			return
		}

		if r.Method == "POST" {
			// validate file size
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				renderError(w, r, "FILE_TOO_BIG", http.StatusBadRequest)
				return
			}

			uploadFileStream, handler, err := r.FormFile("f")
			if err != nil {
				renderError(w, r, "INVALID_FILE", http.StatusBadRequest)
				return
			}
			defer uploadFileStream.Close()

			filename := handler.Filename
			msg, result := writeBytesToFile(filename, uploadFileStream)
			if result {
				rederSuccess(w, r, "Success. Download by: curl "+r.Host+"/"+filename+" -O")
			} else {
				renderError(w, r, msg, http.StatusBadRequest)
			}
			return
		}
	})
}

func MakeDir(path string) (result bool) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.IsDir() {
			return true
		} else {
			log.Fatal(path + " is reguler file, You need remove manully.")
			return false
		}
	}
	if os.IsNotExist(err) {
		err2 := os.Mkdir(path, os.ModePerm)
		if err2 != nil {
			log.Println("mkdir "+path+" failed, ", err2)
			return false
		}
		return true
	}
	log.Println("mkdir "+path+" failed, ", err)
	return false
}

// OK 1. support curl --upload-file
// TODO 2. support return a link instead filename
// TODO 3. support delete when download (one times link, ?expire=0)
// TODO 4. support delete link
// OK 5. support delete after 24 hours (max)
// TODO 6. support parameters(?e[xpire]=<n>[minutes], [1-60*24(max-expire-hours)])
// OK 7. support default page instead file list
// OK 8. support command line argument (upload-directory and host-port [ip]:port, max-upload-size, max-expire-hours)
//         OK host-port
// OK 9. support big as 1G file upload, only 1 - 2 MB memory usage.
