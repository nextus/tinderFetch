package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
	"tinderFetch/api"
)

func getPhoto(rawURL string, photoChan chan io.ReadCloser) {
	const method = "GET"
	log.Printf("Get photo from %s", rawURL)
	defer close(photoChan)
	photoURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Unable to parse url: %v", err)
		return
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}
	request, err := http.NewRequest(method, photoURL.String(), nil)
	if err != nil {
		log.Printf("Unable to get photo by http: %v", err)
		return
	}
	(*request).Header.Set("User-agent", api.DefaultTinderAPIUserAgent)
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Unable to get photo by http: %v", err)
		return
	}
	photoChan <- response.Body
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

func extractPhotos(users []api.User, workDir string) {
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

func downloadPhotos(tinderAPI *api.TinderAPI, args []string) {
	flags := flag.NewFlagSet("download photo options", flag.ExitOnError)
	workingDir := flags.String("dir", "", "Working directory")
	timeSleep := flags.Duration("time", time.Second*1, "throttling time")
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
	usersChan := make(chan []api.User)
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
