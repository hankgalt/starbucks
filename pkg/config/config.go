package config

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Configuration struct {
	GEOCODER_API_KEY string `json:"geocoder_api_key"`
}

func GetConfig() *Configuration {
	var config *Configuration

	rPath, err := os.Getwd()
	if err != nil {
		log.Println("\033[31m Error accessing config path \033[0m")
	}

	var filePath string
	if path.Base(rPath) == "config" {
		bIdx := strings.Index(rPath, path.Base(rPath))
		filePath = filepath.Join(string(rPath[:bIdx]), "config.json")
	} else {
		filePath = filepath.Join(rPath, "config.json")
	}

	// log.Printf("config file path: %s", filePath)

	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("\033[31m %s doesn't exist, error: %v \033[0m", filePath, err)
		} else {
			log.Printf("\033[31m Error accessing file: %s: %v \033[0m", filePath, err)
		}
	} else {
		config = getFromConfigJson(filePath)
	}
	return config
}

func getFromConfigJson(filePath string) *Configuration {
	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("\033[31m Error opening config file: %s: %v \033[0m", filePath, err)
		return nil
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Printf("\033[31m Error closing config file: %s: %v \033[0m", filePath, err)
		}
	}()

	r := bufio.NewReader(f)
	dec := json.NewDecoder(r)

	var config = new(Configuration)

	for {
		var conf Configuration
		if err := dec.Decode(&conf); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("\033[31m Error decoding config json: %v \033[0m", err)
			return nil
		}
		if conf.GEOCODER_API_KEY == "" {
			log.Println("\033[31m Error geocoder api key missing \033[0m")
			return nil
		}
		config.GEOCODER_API_KEY = conf.GEOCODER_API_KEY
	}
	return config
}
