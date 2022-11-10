package config

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type Configuration struct {
	GEOCODER_API_KEY string `json:"geocoder_api_key"`
	PORT             int    `json:"port"`
}

func GetAppConfig(logger *zap.Logger) (*Configuration, error) {
	rPath, err := os.Getwd()
	if err != nil {
		logger.Error("unable to access file path", zap.Error(err))
		return nil, err
	}

	var filePath string
	if path.Base(rPath) == "config" {
		bIdx := strings.Index(rPath, path.Base(rPath))
		filePath = filepath.Join(string(rPath[:bIdx]), "config.json")
	} else {
		filePath = filepath.Join(rPath, "config.json")
	}

	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("file doesn't exist", zap.Error(err), zap.String("filePath", filePath))
		} else {
			logger.Error("error accessing file", zap.Error(err), zap.String("filePath", filePath))
		}
		return nil, err
	}

	return getFromConfigJson(logger, filePath)
}

func getFromConfigJson(logger *zap.Logger, filePath string) (*Configuration, error) {
	f, err := os.Open(filePath)
	if err != nil {
		logger.Error("error opening file", zap.Error(err), zap.String("filePath", filePath))
		return nil, err
	}
	defer func() {
		if err = f.Close(); err != nil {
			logger.Error("error closing file", zap.Error(err), zap.String("filePath", filePath))
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
			logger.Error("error decoding config json", zap.Error(err), zap.String("filePath", filePath))
			return nil, err
		}
		if conf.GEOCODER_API_KEY == "" {
			logger.Error("missing geocoder config", zap.Error(err), zap.String("filePath", filePath))
			return nil, err
		}
		config.GEOCODER_API_KEY = conf.GEOCODER_API_KEY
		if conf.PORT == 0 {
			config.PORT = 8080
		}
		config.PORT = conf.PORT
	}
	return config, nil
}
