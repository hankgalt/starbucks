package loader

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hankgalt/starbucks/pkg/errors"
	"github.com/hankgalt/starbucks/pkg/logging"
	"go.uber.org/zap"
)

// ReadFileArray reads an array of json data from existing file, one by one,
// and returns individual result at defined rate through returned channel
func ReadFileArray(ctx context.Context, cancel func(), fileName string) (<-chan map[string]interface{}, error) {
	filePath := filepath.Join("sample-data", fileName)

	// check if file exists
	err := ifFileExists(filePath)
	if err != nil {
		logging.Logger.Error("error checking file existence", zap.Error(err), zap.String("filePath", filePath))
		cancel()
		return nil, errors.WrapError(err, "error checking %s existence", filePath)
	}

	// Open file and deferred close it
	f, err := os.Open(filePath)
	if err != nil {
		logging.Logger.Error("error opening file", zap.Error(err), zap.String("filePath", filePath))
		cancel()
		return nil, errors.WrapError(err, "error reading file: %s", filePath)
	}

	resultStream := make(chan map[string]interface{}, 2)
	go func(ct context.Context, can func(), fp string, fi *os.File, rs chan map[string]interface{}) {
		defer func(rst chan map[string]interface{}) {
			logging.Logger.Info("Closing result stream")
			close(rst)
		}(rs)

		defer func(fpa string, fil *os.File, canc func()) {
			logging.Logger.Info("Closing file")
			if err = fil.Close(); err != nil {
				logging.Logger.Error("error closing file", zap.Error(err), zap.String("filePath", filePath))
				canc()
			}
		}(fp, fi, can)

		r := bufio.NewReader(fi)
		dec := json.NewDecoder(r)

		// read open bracket
		t, err := dec.Token()
		if err != nil {
			logging.Logger.Error("error reading starting token", zap.Error(err), zap.Any("token", t), zap.String("filePath", filePath))
			cancel()
		}

		// while the array contains values
		for dec.More() {
			var result map[string]interface{}
			err := dec.Decode(&result)
			if err != nil {
				logging.Logger.Error("error decoding result json", zap.Error(err))
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
			logging.Logger.Error("error reading closing token", zap.Error(err), zap.Any("token", t), zap.String("filePath", filePath))
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
			logging.Logger.Error("file doesn't exist", zap.Error(err), zap.String("filePath", filePath))
			return errors.WrapError(err, "File: %s doesn't exist", filePath)
		} else {
			logging.Logger.Error("unable to access file", zap.Error(err), zap.String("filePath", filePath))
			return errors.WrapError(err, "Error accessing file: %s", filePath)
		}
	}
	return nil
}
