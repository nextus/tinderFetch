package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"tinderFetch/api"
)

const (
	exeName      = "tinderGet"
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
	flag.DurationVar(&httpTimeout, "timeout", time.Second*5, "http timeout")
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

func printProfileInfo(tinderAPI *api.TinderAPI, args []string) {
	flags := flag.NewFlagSet("profile options", flag.ExitOnError)
	coordRaw := flags.String("coord", "", "Change profile coordinates. Format: latitude,longitude")
	flags.Parse(args)
	if len(*coordRaw) > 0 {
		coordString := strings.Split(*coordRaw, ",")
		if len(coordString) != 2 {
			log.Fatalln("Bad user input")
		}
		lat, errLat := strconv.ParseFloat(coordString[0], 64)
		lon, errLon := strconv.ParseFloat(coordString[1], 64)
		if errLon != nil || errLat != nil {
			log.Fatalln("Bad user input")
		}
		coord := api.Coordinates{Longitude: lon, Latitude: lat}
		if err := tinderAPI.ChangeProfilePosition(coord); err != nil {
			log.Fatalln(err)
		}
		return
	}
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

func printUserInfo(tinderAPI *api.TinderAPI, args []string) {
	flags := flag.NewFlagSet("user info options", flag.ExitOnError)
	userID := flags.String("id", "", "User id")
	isGeo := flags.Bool("geo", false, "Show user coordinates")
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
	if *isGeo {
		link, err := GetUserPosition(tinderAPI, user.ID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(link)
	} else {
		fmt.Println(string(rawData))
	}

}

func main() {
	args := flag.Args()
	if len(args) < 1 {
		helper(coreCommands)
		os.Exit(2)
	}
	tinderAPI := api.New(getTinderToken(), httpTimeout)
	switch args[0] {
	case "profile":
		printProfileInfo(tinderAPI, args[1:])
	case "download":
		downloadPhotos(tinderAPI, args[1:])
	case "info":
		printUserInfo(tinderAPI, args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		helper(coreCommands)
	}
}
