package loader

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/hankgalt/starbucks/pkg/errors"
)

// ReadFileArray reads an array of json data from existing file, one by one,
// and returns individual result at defined rate through returned channel
func ReadFileArray(ctx context.Context, cancel func(), fileName string) (<-chan map[string]interface{}, error) {
	filePath := filepath.Join("sample-data", fileName)

	// check if file exists
	err := ifFileExists(filePath)
	if err != nil {
		log.Printf("\033[31m Error checking %s existence, error: %v \033[0m", filePath, err)
		cancel()
		return nil, errors.WrapError(err, "error checking %s existence", filePath)
	}

	// Open file and deferred close it
	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("\033[31m Error reading file: %s, error: %v \033[0m", filePath, err)
		cancel()
		return nil, errors.WrapError(err, "error reading file: %s", filePath)
	}

	resultStream := make(chan map[string]interface{}, 2)
	go func(ct context.Context, can func(), fp string, fi *os.File, rs chan map[string]interface{}) {
		defer func(rst chan map[string]interface{}) {
			log.Printf("Closing result stream")
			close(rst)
		}(rs)

		defer func(fpa string, fil *os.File, canc func()) {
			log.Printf("Closing file: %s", fpa)
			if err = fil.Close(); err != nil {
				log.Printf("\033[31m Error closing file: %s, error: %v \033[0m", fpa, err)
				canc()
			}
		}(fp, fi, can)

		r := bufio.NewReader(fi)
		dec := json.NewDecoder(r)

		// read open bracket
		t, err := dec.Token()
		if err != nil {
			log.Printf("\033[31m Error reading starting token: %v, for file: %s, error: %v \033[0m", t, fp, err)
			cancel()
		}

		// while the array contains values
		for dec.More() {
			var result map[string]interface{}
			err := dec.Decode(&result)
			if err != nil {
				log.Printf("\033[31m Error decoding result json, error: %v \033[0m", err)
			}
			// log.Printf("Retrieved %#v\n", result)
			select {
			case <-ct.Done():
				return
			case rs <- result:
			}
		}

		// read closing bracket
		t, err = dec.Token()
		if err != nil {
			log.Printf("\033[31m Error reading closing token: %v, for file: %s, error: %v \033[0m", t, fp, err)
			can()
		}
	}(ctx, cancel, fileName, f, resultStream)

	return resultStream, nil
}

// checks if file exists
func ifFileExists(filePath string) error {
	// path, err := os.Getwd()
	// if err != nil {
	// 	log.Println(err)
	// }
	// log.Println("ifFileExists() - current path: ", path)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("ifFileExists() - \033[31m File %s doesn't exist, error: %v \033[0m", filePath, err)
			return errors.WrapError(err, "File: %s doesn't exist", filePath)
		} else {
			log.Printf("ifFileExists() - \033[31m Error accessing file: %s, error: %v \033[0m", filePath, err)
			return errors.WrapError(err, "Error accessing file: %s", filePath)
		}
	}
	return nil
}
