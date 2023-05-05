package main

import (
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// Add the login to the FTP server here
	server     = "YOUR_SERVER:PORT"
	user       = "YOUR_USER"
	pass       = "YOUR_PASS"
	serverPath = "/YOUR_DIRECTORY_TO_PB_SS/svss/"
	// DownloadFolder Add the path to the folder where you want to save the PB SS
	DownloadFolder = "punkbustersstodiscord"
	timeout        = 35 * time.Second
	maxRetries     = 2
)

var (
	maxFileQueue   = 75
	wg             sync.WaitGroup
	anyNumber      = 0
	UserHomeDir, _ = os.UserHomeDir()
)

func main() {

	DeleteLocal() // Call it before downloading new files

	if maxFileQueue < 10 {
		maxFileQueue = 11
	}
	fileList := fileList()
	if len(fileList) == 0 {
		log.Println("No files to download!")
		time.Sleep(50 * time.Second)
		main()
	} else if len(fileList) < 10 {
		time.Sleep(50 * time.Second)
		main()
		return
	}
	fmt.Println("Found", len(fileList), "files to download!\n")

	for _, filename := range fileList {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()

			c, err := ftp.Dial(server, ftp.DialWithTimeout(timeout), ftp.DialWithShutTimeout(timeout))
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
			localPath := filepath.Join(UserHomeDir, DownloadFolder, filename)
			// Create LocalPath if it doesn't exist
			if _, err := os.Stat(filepath.Join(UserHomeDir, DownloadFolder)); os.IsNotExist(err) {
				os.Mkdir(filepath.Join(UserHomeDir, DownloadFolder), 0777)
			}

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
				c.Delete(filePath) // Always return an error (even if the file is deleted)
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
	anyNumber = 0
	verifyLocalFiles()
	DisgordMain()
	DeleteLocal()

}
func DeleteLocal() {
	// Delete all files in downloads folder after sent to Discord
	dir, err := os.Open(filepath.Join(UserHomeDir, DownloadFolder))
	if err != nil {
		log.Println("[MAIN:Delete] Error while opening local directory:", err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Println("[MAIN:Delete] Error while reading local directory:", err)
	}
	for _, file := range files {
		err := os.Remove(filepath.Join(UserHomeDir, DownloadFolder, file.Name()))
		if err != nil {
			log.Println("[MAIN:Delete] Error while deleting local file:", err)
		} else {
			log.Println("Deleted local file:", file.Name())
		}
	}
}
func verifyLocalFiles() { // Verify the integrity of the files
	dir, err := os.Open(filepath.Join(UserHomeDir, DownloadFolder))
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
			err := os.Remove(filepath.Join(UserHomeDir, DownloadFolder) + "/" + file.Name())
			if err != nil {
				log.Panic("[MAIN:verifyLocalFiles] Error while deleting Server Files:", err)
			}
		}
	}
}

// Func return the file list from the server
func fileList() []string {
	for i := 0; i < maxRetries; i++ {
		c, err := ftp.Dial(server, ftp.DialWithTimeout(timeout), ftp.DialWithShutTimeout(timeout))
		if err != nil {
			log.Println("[MAIN] Error while connecting to FTP server:", err)
			continue
		}
		err = c.Login(user, pass)
		if err != nil {
			log.Println("[MAIN] Error while logging in:", err)
			continue
		}
		defer c.Quit()

		ftpFileList, err := c.List(serverPath)
		if err != nil {
			log.Println("[MAIN] Error on List: ", err)
			continue
		}
		if len(ftpFileList) == 0 {
			log.Println("[MAIN] No files found")
			// Wait for new files
			time.Sleep(5 * time.Second)
			break
		}
		var fileList []string
		for _, file := range ftpFileList {
			// Ignore files smaller than 1000 bytes and just check .png files:
			if file.Size > 1000 && strings.Contains(file.Name, ".png") {
				if anyNumber == maxFileQueue {
					return fileList
				}
				fileList = append(fileList, file.Name)
				anyNumber++
			}
		}
		return fileList
	}
	log.Println("[MAIN] Failed to get file list after 3 retries")
	return nil
}
