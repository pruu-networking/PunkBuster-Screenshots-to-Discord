package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	// BotToken Add your bot token and Channel ID here
	BotToken  = flag.String("token", "SECRET.TOKEN_HERE", "Bot token")
	ChannelID = "YOUR_Screenshots_ChannelID" // You should create a New Channel! The bot will spam a lot of images!
)

func DisgordMain() {
	DcSession, _ := discordgo.New("Bot " + *BotToken)

	DcSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot job is done, waiting for new files!")
		// Add a timer to check for new files every 5 seconds
		time.Sleep(20 * time.Second)
		err := DcSession.Close()
		if err != nil {
			log.Println("[BOT] Error while closing the session:", err)
		}
		// Delete files on local directory
		dir, _ := os.Open(DownloadFolder + "/")
		defer dir.Close()
		files, _ := dir.Readdir(-1)
		for _, file := range files {
			err := os.Remove(filepath.Join(DownloadFolder, file.Name()))
			if err != nil {
				log.Println("[BOT:Delete] Error while deleting local file:", err)
			} else {
				log.Println("Deleted local file:", file.Name())
			}
		}
		main()
	})

	for x := 0; x < len(fileVerify()); x++ {

		//fmt.Println("Files found")
		filePath := UserHomeDir + DownloadFolder + "/" + fileVerify()[x]

		log.Println("Sending file: ", fileVerify()[x])
		myFile, err := os.Open(filePath)
		if err != nil {
			log.Panicf("[BOT] Cannot open the file: %v", err)
		}
		defer myFile.Close()

		_, err = DcSession.ChannelFileSend(ChannelID, fileVerify()[x], myFile)
		if err != nil {
			log.Panicf("[BOT] Cannot send the file to the Channel: %v", err)
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
	filePath := userHomeDir + "/git/PunkBuster-Screenshots-to-Discord/" + DownloadFolder + "/"
	files, err := os.ReadDir(filePath)
	if err != nil {
		log.Println("[BOT] Error on verifying Local Files:", err)
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
