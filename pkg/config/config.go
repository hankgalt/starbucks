package config

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hankgalt/starbucks/pkg/logging"
	"go.uber.org/zap"
)

type Configuration struct {
	GEOCODER_API_KEY string `json:"geocoder_api_key"`
}

func GetConfig() (*Configuration, error) {
	rPath, err := os.Getwd()
	if err != nil {
		logging.Logger.Error("unable to access file path", zap.Error(err))
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
			logging.Logger.Error("file doesn't exist", zap.Error(err), zap.String("filePath", filePath))
		} else {
			logging.Logger.Error("error accessing file", zap.Error(err), zap.String("filePath", filePath))
		}
		return nil, err
	}

	return getFromConfigJson(filePath)
}

func getFromConfigJson(filePath string) (*Configuration, error) {
	f, err := os.Open(filePath)
	if err != nil {
		logging.Logger.Error("error opening file", zap.Error(err), zap.String("filePath", filePath))
		return nil, err
	}
	defer func() {
		if err = f.Close(); err != nil {
			logging.Logger.Error("error closing file", zap.Error(err), zap.String("filePath", filePath))
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
			logging.Logger.Error("error decoding config json", zap.Error(err), zap.String("filePath", filePath))
			return nil, err
		}
		if conf.GEOCODER_API_KEY == "" {
			logging.Logger.Error("missing geocoder config", zap.Error(err), zap.String("filePath", filePath))
			return nil, err
		}
		config.GEOCODER_API_KEY = conf.GEOCODER_API_KEY
	}
	return config, nil
}
