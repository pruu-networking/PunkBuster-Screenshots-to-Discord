package main

import (
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (	
	server     = "YOUR_SERVER:PORT"
	user       = "YOUR_USER"
	pass       = "YOUR_PASS"
	serverPath = "/YOUR_DIRECTORY_TO_PB_SS/svss/"
	timeout    = 30 * time.Second
	maxRetries = 3
)

var wg sync.WaitGroup

func main() {
	fileList := fileList()
	fmt.Println("Found", len(fileList), "files to download!\n")

	for _, filename := range fileList {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()

			c, err := ftp.Dial(server, ftp.DialWithTimeout(timeout))
			if err != nil {
				log.Println("[MAIN] Error while connecting to FTP server:", err)
				return
			}
			defer c.Quit()
			err = c.Login(user, pass)
			if err != nil {
				log.Panic(err)
			}

			filePath := filepath.Join(serverPath, filename)
			localPath := filepath.Join("downloads", filename)
			log.Println("Downloading:", filename)

			// Create local file
			var downloaded bool
			for i := 0; i < maxRetries; i++ {
				res, err := c.Retr(filePath)
				if err != nil {
					log.Printf("[MAIN] Error while retrieving file from server (retry %d/%d): %v", i+1, maxRetries, err)
					time.Sleep(time.Second)
					continue
				}
				defer res.Close()

				file, err := os.Create(localPath)
				if err != nil {
					log.Println("[MAIN] Error while creating local file:", err)
					return
				}
				defer file.Close()

				_, err = io.Copy(file, res)
				if err != nil {
					log.Println("[MAIN] Error while copying the file:", err)
					os.Remove(localPath)
					return
				}

				downloaded = true
				// Delete file from server
				err = c.Delete(filePath)
				if err != nil {
					log.Println("[MAIN] Error while deleting file from server:", err)
				}

				break
			}

			if downloaded {
				log.Println("Downloaded:", filename)
			} else {
				log.Printf("[MAIN] Failed to download file %s after %d retries", filename, maxRetries)
				os.Remove(localPath)
			}
		}(filename)
	}

	wg.Wait()
	verifyLocalFiles()
	DisgordMain()

	// Delete all files in downloads folder after sent to Discord
	dir, err := os.Open("downloads/")
	if err != nil {
		log.Println("[MAIN:Delete] Error while opening local directory:", err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Println("[MAIN:Delete] Error while reading local directory:", err)
	}
	for _, file := range files {
		err := os.Remove(filepath.Join("downloads", file.Name()))
		if err != nil {
			log.Println("[MAIN:Delete] Error while deleting local file:", err)
		} else {
			log.Println("Deleted local file:", file.Name())
		}
	}
}

func verifyLocalFiles() { // Verify the integrity of the files
	dir, err := os.Open("downloads/")
	if err != nil {
		log.Panic(err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Panic(err)
	}
	for _, file := range files { // Delete files with 0 bits (corrupted)
		if file.Size() == 0 {
			fmt.Println("Deleting file: ", file.Name())
			err := os.Remove("downloads/" + file.Name())
			if err != nil {
				log.Panic("[MAIN:verifyLocalFiles] Error while deleting Server Files:", err)
			}
		}
	}
}

// Func return the file list from the server
func fileList() []string {

	c, err := ftp.Dial(server, ftp.DialWithTimeout(timeout))
	if err != nil {
		log.Panic(err)
	}
	err = c.Login(user, pass)
	if err != nil {
		log.Panic(err)
	}
	defer c.Quit()

	ftpFileList, err := c.List(serverPath)
	if err != nil {
		log.Panic("[MAIN] Error on List: ", err)
	}
	if len(ftpFileList) == 0 {
		log.Panic("[MAIN] No files found")
	}
	var fileList []string
	for _, file := range ftpFileList {
		if file.Size > 1000 && file.Name != "pbsvss.htm" {
			fileList = append(fileList, file.Name)
		}
	}
	return fileList
}
