package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server        string
	User          string
	Password      string
	SFTPFolder    string
	BotToken      string
	ChannelID     string
	WaitingTime   int
	SelectFTPMode string
}

var NDownload int
var debugMode bool
var OneTime = false

func readConfig() (*Config, error) {
	var cfg Config

	cfg.Server = os.Getenv("SERVER")
	cfg.User = os.Getenv("USER")
	cfg.Password = os.Getenv("PASS")
	cfg.SFTPFolder = os.Getenv("SFTP_FOLDER")
	cfg.BotToken = os.Getenv("BOT_TOKEN")
	cfg.ChannelID = os.Getenv("CHANNEL_ID")
	cfg.SelectFTPMode = os.Getenv("SELECT_FTP_MODE")

	waitingTimeStr := os.Getenv("WAITING_TIME")
	waitingTime, err := strconv.Atoi(waitingTimeStr)
	if err != nil {
		// Handle parsing error if needed
		return nil, fmt.Errorf("failed to parse WAITING_TIME: %v", err)
	}
	cfg.WaitingTime = waitingTime
	if cfg.WaitingTime < 2 || cfg.WaitingTime > 120 {
		cfg.WaitingTime = 30
	}

	// Debug mode
	debugModeStr := os.Getenv("DEBUG_MODE")
	debugMode, err = strconv.ParseBool(debugModeStr)
	if err != nil {
		debugMode = false // Set a default value if DEBUG_MODE is not provided or not a valid boolean
	}
	if !OneTime { // Print the config only once, not every time the Config is read
		log.Println("Debug mode:", debugMode)
		OneTime = true
	}

	// Set default values if not provided
	if cfg.Server == "" {
		log.Fatalf("SERVER environment variable not set")
	}
	if cfg.User == "" {
		log.Fatalf("USER environment variable not set")
	}
	if cfg.Password == "" {
		log.Fatalf("PASS environment variable not set")
	}
	if cfg.SFTPFolder == "" {
		log.Fatalf("SFTP_FOLDER environment variable not set")
	}
	if cfg.BotToken == "" {
		log.Fatalf("BOT_TOKEN environment variable not set")
	}
	if cfg.ChannelID == "" {
		log.Fatalf("CHANNEL_ID environment variable not set")
	}
	if cfg.SelectFTPMode == "" || (cfg.SelectFTPMode != "sftp" && cfg.SelectFTPMode != "ftp") {
		cfg.SelectFTPMode = "ftp"
	}

	return &cfg, nil
}

func serverSelect(config Config, sshConfig *ssh.ClientConfig) (*ftp.ServerConn, *sftp.Client, int) {
	var (
		selectSftp *sftp.Client
		selectFtp  *ftp.ServerConn
		err        error
	)
	NDownload = 0
	if config.SelectFTPMode == "sftp" {
		selectSftp, NDownload, err = sftpServer(config, sshConfig)
		if err != nil {
			log.Fatalf("Error running sFTP server: %v", err)
		}
		return selectFtp, selectSftp, NDownload
	} else {
		selectFtp, NDownload, err = ftpServer(config)
		if err != nil {
			log.Fatalf("Error running FTP server: %v", err)
		}
		return selectFtp, selectSftp, NDownload
	}

	return selectFtp, selectSftp, NDownload
}

func ftpServer(config Config) (*ftp.ServerConn, int, error) {
	server := config.Server
	user := config.User
	pass := config.Password
	serverPath := config.SFTPFolder

	client, err := ftp.Dial(server)
	if err != nil {
		return nil, 0, fmt.Errorf("error while connecting to FTP server: %v", err)
	}
	if debugMode {
		log.Println("Connected to FTP server")
	}
	err = client.Login(user, pass)
	if err != nil {
		return nil, 0, fmt.Errorf("error while logging in: %v", err)
	}
	//log.Println("Logged in to FTP server")
	ftpFileList, err := client.List(serverPath)
	if err != nil {
		return nil, 0, fmt.Errorf("error on List: %v", err)
	}
	//log.Println("Read FTP folder, found", len(ftpFileList), "files")
	for _, file := range ftpFileList {
		if file.Size > 1000 && strings.Contains(file.Name, ".png") {
			NDownload++
		}
	}
	return client, NDownload, nil
}

func sftpServer(config Config, sshConfig *ssh.ClientConfig) (*sftp.Client, int, error) {
	conn, err := ssh.Dial("tcp", config.Server, sshConfig)
	if err != nil {
		log.Fatalf("Failed to connect to sFTP server: %v", err)
	}
	if debugMode {
		log.Println("Connected to sFTP server")
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatalf("Failed to create sFTP client: %v", err)
	}
	if debugMode {
		log.Println("Created sFTP client")
	}
	files, err := client.ReadDir(config.SFTPFolder)
	if err != nil {
		log.Fatalf("Failed to read sFTP folder: %v", err)
	}
	if debugMode {
		log.Println("Read sFTP folder, found", len(files), "files")
	}

	NDownload := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".png") && file.Size() > 1000 {
			NDownload++
		}
	}

	return client, NDownload, nil

}
func main() {
	for {
		cfg, err := readConfig()
		if err != nil {
			log.Fatalf("Error reading config: %v", err)
		}

		config := &ssh.ClientConfig{
			User: cfg.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(cfg.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		// Select FTP or sFTP server
		ftpClientConfig, sftpClient, NDownload := serverSelect(*cfg, config)

		if ftpClientConfig != nil {
			defer ftpClientConfig.Quit()
		} else if sftpClient != nil {
			defer sftpClient.Close()
		}

		if NDownload == 0 {
			if debugMode {
				log.Println("No files to download, waiting", cfg.WaitingTime, "minutes")
			}
			time.Sleep(time.Minute * time.Duration(cfg.WaitingTime))
			// Restart the program
			continue
		}
		if debugMode {
			log.Println("Found", NDownload, "files to download")
		}

		// Create a Discord session
		DcSession, err := discordgo.New("Bot " + cfg.BotToken)
		if err != nil {
			log.Fatalf("Failed to create Discord session: %v", err)
		}
		err = DcSession.Open()
		if err != nil {
			log.Fatalf("failed to open connection: %v", err)
		}
		defer DcSession.Close()
		if debugMode {
			log.Println("Created Discord session")
		}

		// Download all the files and send to Discord
		if sftpClient != nil {
			files, _ := sftpClient.ReadDir(cfg.SFTPFolder)
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".png") && file.Size() > 1000 {
					downloadAndSendToDiscord(DcSession, cfg.SFTPFolder+"/"+file.Name(), sftpClient, nil)
				}

			}
		} else {
			ftpFileList, _ := ftpClientConfig.List(cfg.SFTPFolder)
			if debugMode {
				log.Println("Read FTP folder, found", len(ftpFileList), "files")
			}
			for _, file := range ftpFileList {
				if file.Size > 1000 && strings.Contains(file.Name, ".png") {
					downloadAndSendToDiscord(DcSession, cfg.SFTPFolder+"/"+file.Name, nil, ftpClientConfig)
				}
			}
		}
		NDownload = 0
	}
}

func downloadAndSendToDiscord(DcSession *discordgo.Session, filePath string, client *sftp.Client, ftpClent *ftp.ServerConn) {
	// Download the file
	var fileCreationDate string
	localPath := filepath.Base(filePath)
	if client != nil {
		srcFile, err := client.Open(filePath)
		if err != nil {
			log.Printf("Cannot open the file %s: %v", filePath, err)
			return
		}

		dstFile, err := os.Create(localPath)
		if err != nil {
			log.Printf("Cannot create local file %s: %v", localPath, err)
			return
		}

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			log.Printf("Failed to download file %s: %v", filePath, err)
			return
		}
		// Check the CreationTime of the orignal file
		fileInfo, err := srcFile.Stat()
		if err != nil {
			log.Printf("Failed to get file info %s: %v", filePath, err)
		} else {
			creationTime := fileInfo.ModTime().Add(-3 * time.Hour)
			fileCreationDate = creationTime.Format("2006-01-02 15:04:05")
			if debugMode {
				log.Printf("File %s created at %s", filePath, fileCreationDate)
			}
		}

		dstFile.Close()
		srcFile.Close()
		// Delete the original file
		err = client.Remove(filePath)
		if err != nil {
			log.Println("Failed to delete remote file:", err, filePath)
			return
		}
	} else {
		myFile, err := os.Create(localPath)
		if debugMode {
			log.Printf("Created local path for file %s", filePath)
		}
		if err != nil {
			log.Printf("Cannot create local file %s: %v", localPath, err)
			return
		}
		// Get file information for FTP downloaded file
		if debugMode {
			if ftpClent.IsGetTimeSupported() {
				log.Println("The server supports the MDTM command")
			} else {
				log.Println("The server does not support the MDTM command")
			}
		}
		fileInfo, err := ftpClent.GetTime(filePath)
		fileInfo = fileInfo.Add(-3 * time.Hour)
		fileCreationDate = fileInfo.Format("2006-01-02 15:04:05")
		if debugMode {
			if err != nil {
				log.Printf("fileInfo error: %v", err)
			}
			log.Printf("File %s sent with time: %s", filePath, fileCreationDate)
		}

		fileResp, err := ftpClent.Retr(filePath)
		if err != nil {
			log.Printf("Failed to download file %s: %v", filePath, err)
			return
		}
		if debugMode {
			log.Printf("Downloaded file %s", filePath)
		}

		_, err = io.Copy(myFile, fileResp)
		if err != nil {
			log.Println("Failed to copy file:", err, filePath)
			return
		}
		if debugMode {
			log.Printf("Copied file %s", filePath)
		}

		myFile.Close()
		fileResp.Close()
		err = ftpClent.Delete(filePath)
		if err != nil {
			log.Println("Failed to delete remote file:", err, filePath)
			return
		}
		if debugMode {
			log.Println("Deleted remote file:", filePath)
		}
	}

	// Extract pbGuid
	myFile, err := os.Open(localPath)
	if err != nil {
		log.Panicf("Cannot open the file %s: %v", localPath, err)
		return
	}
	defer myFile.Close()

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

	err = myFile.Close()
	if err != nil {
		log.Printf("Failed to close file %s: %v", localPath, err)
		return
	}

	// Send the file to Discord
	err = Sender(DcSession, pbGuid, localPath, fileCreationDate)
	if err != nil {
		log.Printf("Failed to send file %s to Discord: %v", filePath, err)
	}
	time.Sleep(time.Second / 2)
	// Delete the downloaded local file
	err = os.Remove(myFile.Name())
	if err != nil {
		log.Printf("Failed to delete local file %s: %v", myFile.Name, err)
	}
}

func Sender(DcSession *discordgo.Session, pbGuid, filePath, fileCreationDate string) error {
	// Open the file
	fileData, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	if debugMode {
		log.Println("Opened file:", fileData.Name())
	}
	defer fileData.Close()
	cfg, _ := readConfig()
	_, err = DcSession.ChannelMessageSendComplex(cfg.ChannelID, &discordgo.MessageSend{
		Content: "File: " + fileData.Name() + " | Created at: " + fileCreationDate + "\n" + "PBGUID: " + pbGuid,
		Files: []*discordgo.File{
			{
				Name:   fileData.Name(),
				Reader: fileData,
			},
		},
	})
	if debugMode {
		log.Printf("Sending: %s", fileData.Name())
	}
	if err != nil {
		fileData.Close()
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}
