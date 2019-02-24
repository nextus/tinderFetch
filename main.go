package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	exeName      = "tinderFetch"
	envTokenName = "TINDER_TOKEN"
	idsFileName  = "ids.txt"
	metaFileName = "meta.json"
)

type commandExe struct {
	name, description string
}

type photoEntity struct {
	pURL, Path string
}

var httpTimeout time.Duration
var coreCommands []commandExe

func init() {
	flag.Parse()
	coreCommands = append(coreCommands, commandExe{"profile", "show profile"},
		commandExe{"download", "download user information"},
		commandExe{"info", "get information about specific user"},
	)
}

func getTinderToken() string {
	apiTokenString, ok := os.LookupEnv(envTokenName)
	if !ok {
		log.Fatalf("Environment variable %s is empty", envTokenName)
	}
	return apiTokenString
}

func getPhoto(rawURL string, photoChan chan io.ReadCloser) {
	const method = "GET"
	log.Printf("Get photo from %s", rawURL)
	defer close(photoChan)
	photoURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Unable to parse url: %v", err)
		return
	}
	headers := httpHeaders{
		"User-agent": []string{
			tinderAPIUserAgent,
		},
	}
	httpResponse, err := doHTTPRequest(photoURL, method, &headers, nil)
	if err != nil {
		log.Printf("Unable to get photo by http: %v", err)
		return
	}
	photoChan <- httpResponse.Body
}

func savePhoto(photoPath string, photoChan chan io.ReadCloser) {
	photoBody := <-photoChan
	if photoBody == nil {
		return
	}
	log.Printf("Saving photo %s", photoPath)
	defer photoBody.Close()
	if _, err := os.Stat(photoPath); err == nil {
		log.Printf("File has already been exist: %s. Do nothing", photoPath)
		return
	}
	fd, err := os.Create(photoPath)
	if err != nil {
		log.Printf("Unable to create file %v", err)
		return
	}
	defer fd.Close()
	io.Copy(fd, photoBody)
}

func processPhotos(url, photoPath string) {
	var wg sync.WaitGroup
	photoChan := make(chan io.ReadCloser)
	wg.Add(2)
	go func() {
		defer wg.Done()
		getPhoto(url, photoChan)
	}()
	go func() {
		defer wg.Done()
		savePhoto(photoPath, photoChan)
	}()
	wg.Wait()
}

func saveUserID(id, workDir string) error {
	idsFilePath := filepath.Join(workDir, idsFileName)
	f, err := os.OpenFile(idsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write([]byte(id + "\n")); err != nil {
		return err
	}
	return nil
}

func extractPhotos(users []User, workDir string) {
	var wg sync.WaitGroup
	sema := make(chan photoEntity)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entity := range sema {
				processPhotos(entity.pURL, entity.Path)
			}
		}()
	}
	for _, user := range users {
		if err := saveUserID(user.ID, workDir); err != nil {
			log.Fatalf("Unable to save user id to log file: %v", err)
		}
		userDir := filepath.Join(workDir, user.ID)
		// save metainformation about user
		if err := os.Mkdir(userDir, os.ModePerm); os.IsNotExist(err) {
			log.Printf("Unable to create directory %s for user data", userDir)
			continue
		}
		rawData, err := json.MarshalIndent(user, "", "\t")
		if err != nil {
			log.Printf("Unable to encode metainformation about user %s: %v", user.ID, err)
			continue
		}
		if err := ioutil.WriteFile(filepath.Join(userDir, metaFileName), rawData, os.ModePerm); err != nil {
			log.Printf("Unable to save metainformation about user %s: %v", user.ID, err)
			continue
		}
		for _, photo := range user.Photos {
			photoPath := filepath.Join(userDir, photo.FileName)
			sema <- photoEntity{photo.URL, photoPath}
		}
	}
	close(sema)
	wg.Wait()
}

func signalHandler(done chan bool) {
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan,
		syscall.SIGTERM, // the normal way to ask a program to terminate
		syscall.SIGINT,  // Ctrl + C
	)
	<-sigchan
	done <- true
}

func helper(comms []commandExe) {
	fmt.Printf("usage: %s {OPTIONS | help } OBJECT { OBJECT_OPTIONS | help }\n", exeName)
	fmt.Printf("Commands: \n")
	for _, comm := range comms {
		fmt.Printf(" %s:\t%s\n", comm.name, comm.description)
	}
}

func printProfileInfo(tinderAPI *TinderAPI) {
	profile, err := tinderAPI.GetProfile()
	if err != nil {
		log.Fatalf("Error while getting profile info: %v", err)
	}
	rawData, err := json.MarshalIndent(profile, "", "\t")
	if err != nil {
		log.Fatalf("Unable to print profile info: %v", err)
	}
	fmt.Println(string(rawData))
}

func printUserInfo(tinderAPI *TinderAPI, args []string) {
	flags := flag.NewFlagSet("user info options", flag.ExitOnError)
	userID := flags.String("id", "", "User id")
	flags.Parse(args)
	if len(*userID) < 1 {
		flags.PrintDefaults()
		return
	}
	user, err := tinderAPI.GetUser(*userID)
	if err != nil {
		log.Fatalf("Error while getting user info: %v", err)
	}
	rawData, err := json.MarshalIndent(user, "", "\t")
	if err != nil {
		log.Fatalf("Unable to print user info: %v", err)
	}
	fmt.Println(string(rawData))

}

func downloadPhotos(tinderAPI *TinderAPI, args []string) {
	flags := flag.NewFlagSet("download photo options", flag.ExitOnError)
	workingDir := flags.String("dir", "", "Working directory")
	timeSleep := flags.Duration("time", time.Second*1, "throttling time")
	flags.DurationVar(&httpTimeout, "timeout", time.Second*5, "http timeout")
	flags.Parse(args)
	if len(*workingDir) < 1 {
		flags.PrintDefaults()
		return
	}
	if _, err := os.Stat(*workingDir); err != nil {
		log.Fatalf("Unable to access to working directory: %v", err)
	}

	done := make(chan bool)
	go signalHandler(done)
	var wg sync.WaitGroup
	// get new users from tinder API
	usersChan := make(chan []User)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(usersChan)
		for {
			select {
			case <-done:
				return
			default:
				users, err := tinderAPI.GetUsers()
				if err != nil {
					log.Println(err)
					time.Sleep(*timeSleep)
					continue
				}
				usersChan <- users
				time.Sleep(*timeSleep)
			}
		}
	}()
	// dislike users
	dislikeUserChan := make(chan string, 32)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for user := range dislikeUserChan {
			tinderAPI.Dislike(user)
			time.Sleep(*timeSleep)
		}
	}()
	// download info about each user
	for users := range usersChan {
		extractPhotos(users, *workingDir)
		for _, user := range users {
			dislikeUserChan <- user.ID
		}
	}
	close(dislikeUserChan)
	wg.Wait()
}

func main() {
	args := flag.Args()
	if len(args) < 1 {
		helper(coreCommands)
		os.Exit(2)
	}
	tinderAPI := TinderAPI{
		token:       getTinderToken(),
		host:        tinderAPIHost,
		contentType: tinderAPIContentType,
		userAgent:   tinderAPIUserAgent,
	}
	switch args[0] {
	case "profile":
		printProfileInfo(&tinderAPI)
	case "download":
		downloadPhotos(&tinderAPI, args[1:])
	case "info":
		printUserInfo(&tinderAPI, args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		helper(coreCommands)
	}
}
