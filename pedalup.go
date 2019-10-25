package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/disk"
	strava "github.com/strava/go.strava"
	"gopkg.in/ini.v1"
)

func findFile(targetDir string, pattern []string) []string {

	for _, v := range pattern {
		matches, err := filepath.Glob(targetDir + v)

		if err != nil {
			fmt.Println(err)
		}

		if len(matches) != 0 {
			return matches
		}
	}

	return []string{}
}

func getFitFiles(targetDirectory string) []string {
	fitFilesGlob := []string{"Lezyne/Activities/*.fit"}
	return findFile(targetDirectory+"/", fitFilesGlob)
}

func findLezyneGPSVolume() (string, error) {
	var found string
	lezyneIniFilename := "autorun.inf"

	partitions, _ := disk.Partitions(false)
	for _, partition := range partitions {
		if strings.Contains(partition.Mountpoint, "/Volumes") {
			file, _ := os.Stat(partition.Mountpoint + "/" + lezyneIniFilename)
			if file != nil {
				found = partition.Mountpoint
			}
		}
	}

	if len(found) > 0 {
		cfg, err := ini.Load(found + "/" + lezyneIniFilename)
		if err != nil {
			fmt.Printf("Fail to read file: %v", err)
			os.Exit(1)
		}

		autorunLabel := cfg.Section("autorun").Key("label").String()

		if strings.Contains(autorunLabel, "Lezyne GPS") {
			return found, nil
		}
	}

	return "", errors.New("Lezyne GPS Volume not found")
}

func main() {
	var accessToken string

	// Provide an access token, with write permissions.
	// You'll need to complete the oauth flow to get one.
	flag.StringVar(&accessToken, "token", "", "Access Token")

	flag.Parse()

	if accessToken == "" {
		fmt.Println("\nPlease provide an access_token")

		flag.PrintDefaults()
		os.Exit(1)
	}

	lezyne, err := findLezyneGPSVolume()
	if err != nil {
		log.Fatal(err)
	}

	fitFiles := getFitFiles(lezyne)

	client := strava.NewClient(accessToken)
	service := strava.NewUploadsService(client)

	fmt.Printf("Uploading data...\n")

	for _, fitFile := range fitFiles {
		r, err := os.Open(fitFile)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(fitFile)

		upload, err := service.
			Create(strava.FileDataTypes.FIT, filepath.Base(fitFile), r).
			Private().
			Do()

		if err != nil {
			if e, ok := err.(strava.Error); ok && e.Message == "Authorization Error" {
				log.Printf("Make sure your token has 'write' permissions. You'll need implement the oauth process to get one")
			}

			log.Fatal(err)
		}
		log.Printf("Upload Complete...")
		jsonForDisplay, _ := json.Marshal(upload)
		log.Printf(string(jsonForDisplay))
	}

}
