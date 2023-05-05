package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	// BotToken Add your bot token and Channel ID here
	BotToken  = flag.String("token", "SECRET.TOKEN_HERE", "Bot token")
	ChannelID = "YOUR_Screenshots_ChannelID" // You should create a New Channel! The bot will spam a lot of images!
)

func deleteLocalFiles(files []string) error {
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			// Close the file handle if it's still open
			if e, ok := err.(*os.PathError); ok && e.Err == syscall.Errno(0x20) {
				// Error code 0x20 is "The process cannot access the file because it is being used by another process"
				f, err := os.Open(f)
				if err == nil {
					f.Close()
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func DisgordMain() {
	DcSession, _ := discordgo.New("Bot " + *BotToken)

	DcSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot job is done, waiting for new files!")
		// Add a timer to check for new files every 5 seconds

		dir, err := os.Open(filepath.Join(UserHomeDir, DownloadFolder))
		if err != nil {
			log.Println("[MAIN:Delete] Error while opening local directory:", err)
		}
		defer dir.Close()

		files, err := dir.Readdir(-1)
		if err != nil {
			log.Printf("[BOT:Delete] Error while reading directory: %v", err)
		}
		var fileNames []string
		for _, file := range files {
			fileNames = append(fileNames, filepath.Join(UserHomeDir, DownloadFolder, file.Name()))
		}
		time.Sleep(2 * time.Second)
		if err := deleteLocalFiles(fileNames); err != nil {
			log.Printf("[BOT:Delete] Error while deleting local files: %v", err)
		}

		err = DcSession.Close()
		if err != nil {
			log.Println("[BOT] Error while closing the session:", err)
		}
		time.Sleep(2 * time.Second)
		main()
	})

	for x := 0; x < len(fileVerify()); x++ {
		filePath := UserHomeDir + "/" + DownloadFolder + "/" + fileVerify()[x]
		log.Println("Sending file: ", fileVerify()[x])

		myFile, err := os.Open(filePath)
		if err != nil {
			log.Panicf("[BOT] Cannot open the file: %v", err)
		}
		j := 0
		var pbGuid string
		data, _ := io.ReadAll(myFile)

		lines := strings.Split(string(data), "\n")
	PbLoop:
		for _, line := range lines {
			if j == 4 {
				pbGuid = line
				break PbLoop
			}
			j++
		}
		err = myFile.Close() // close the file before sending it to Discord
		if err != nil {
			log.Panicf("[BOT - myFile.Close ] Cannot close the file: %v", err)
		}

		newFile, err := os.Open(filePath)
		if err != nil {
			log.Panicf("[BOT] Cannot open the file: %v", err)
		}

		_, err = DcSession.ChannelMessageSendComplex(ChannelID, &discordgo.MessageSend{
			Content: "File: " + fileVerify()[x] + "\n" + "PB GUID: " + pbGuid,
			Files: []*discordgo.File{
				{
					Name:   fileVerify()[x],
					Reader: newFile,
				},
			},
		})
		if err != nil {
			log.Panicf("[BOT] Cannot send the file to the Channel: %v", err)
		}
		err = newFile.Close()
		if err != nil {
			log.Panicf("[BOT-After Send] Cannot close the file: %v", err)
		}
		// If loop is complete, break the loop
		if x == len(fileVerify())-1 {
			break
		}
	}

	err := DcSession.Open()
	if err != nil {
		log.Panicln("[BOT] Cannot open the session:", err)
	}
	defer DcSession.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, os.Kill)
	<-stop
	//log.Println("Graceful shutdown")
	stop <- os.Interrupt

}

func fileVerify() []string { // Check if the file exists and return as io.Reader
	// List all files on local path downloads/
	userHomeDir, _ := os.UserHomeDir()
	filePath := userHomeDir + "/" + DownloadFolder + "/"
	files, err := os.ReadDir(filePath)
	if err != nil {
		log.Panicln("[BOT] Error on verifying Local Files:", err)
	}

	if len(files) == 0 {
		log.Println("[BOT] No files found")
		return nil
	}

	var LocalFiles []string
	for _, f := range files {
		//fmt.Println(f.Name())
		LocalFiles = append(LocalFiles, f.Name())
	}

	return LocalFiles
}
