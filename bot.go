package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	BotToken  = flag.String("token", "SECRET.TOKEN_HERE", "Bot token")
	ChannelID = "YOUR_Screenshots_ChannelID" // You should create a New Channel! The bot will spam a lot of images!
)

func DisgordMain() {

	sc, _ := discordgo.New("Bot " + *BotToken)
	sc.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot job is done, closing the session!")
		err := sc.Close()
		if err != nil {
			log.Println("[BOT] Error while closing the session:", err)
		}
	})

	for x := 0; x < len(fileVerify()); x++ {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("[BOT] Error getting user home directory:", err)
			return
		}

		//fmt.Println("Files found")
		filePath := userHomeDir + "/git/PunkBuster-Screenshots-to-Discord/downloads/" + fileVerify()[x]
		// if file is NIL, skip
		if fileVerify()[x] == "nil" {
			log.Println("[BOT] File is NIL, skipping...")
			continue
		}

		log.Println("Sending file: ", fileVerify()[x])
		myFile, err := os.Open(filePath)
		if err != nil {
			log.Panicf("[BOT] Cannot open the file: %v", err)
		}
		defer myFile.Close()

		_, err = sc.ChannelFileSend(ChannelID, filePath, myFile)
		if err != nil {
			log.Panicf("[BOT] Cannot send the file to the Channel: %v", err)
		}
	}

	err := sc.Open()
	if err != nil {
		log.Println("[BOT] Cannot open the session:", err)
	}
	defer sc.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, os.Kill)
	<-stop
	log.Println("Graceful shutdown")
	stop <- os.Interrupt

}

func fileVerify() []string { // Check if the file exists and return as io.Reader
	// List all files on local path downloads/
	userHomeDir, _ := os.UserHomeDir()
	filePath := userHomeDir + "/git/PunkBuster-Screenshots-to-Discord/downloads/"
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
