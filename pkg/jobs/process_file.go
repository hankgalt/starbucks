package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/services/store"
	"github.com/hankgalt/starbucks/pkg/utils/loader"
	"go.uber.org/zap"
)

func ProcessFile(ss *store.StoreService, logger *zap.Logger) {
	defer func() {
		logger.Info("finished setting up store data")
	}()

	ctx := context.WithValue(context.Background(), constants.FileNameContextKey, "locations.json")
	ctx = context.WithValue(ctx, constants.ReadRateContextKey, 2)
	ctx, cancel := context.WithCancel(ctx)

	cout := make(chan *store.Store)
	var wgp sync.WaitGroup
	var wgs sync.WaitGroup

	wgp.Add(1)
	go readFile(ctx, cancel, &wgp, &wgs, cout, logger)
	wgp.Add(1)
	go processStore(ctx, cancel, &wgp, &wgs, cout, ss, logger)

	func() {
		wgp.Wait()
		ss.SetReady(ctx, true)
		stats := ss.GetStoreStats(ctx)
		logger.Info("gateway status", zap.Any("stats", stats))
	}()
}

func readFile(
	ctx context.Context,
	cancel func(),
	wgp *sync.WaitGroup,
	wgs *sync.WaitGroup,
	out chan *store.Store,
	logger *zap.Logger,
) {
	logger.Info("start reading store data file")
	fileName := ctx.Value(constants.FileNameContextKey).(string)
	resultStream, err := loader.ReadFileArray(ctx, cancel, fileName)
	if err != nil {
		logger.Error("error reading store data file", zap.Error(err))
		cancel()
	}
	count := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("store data file read context done")
			return
		case r, ok := <-resultStream:
			if !ok {
				logger.Info("store data file result stream closed")
				wgp.Done()
				close(out)
				return
			}
			wgs.Add(1)
			count++
			// if count%1000 == 0 {
			// 	logger.Debug("publishing store", zap.Any("storeJson", r), zap.Int("storeCount", count))
			// }
			publishStore(r, out, logger)
		}
	}
}

func publishStore(
	r map[string]interface{},
	out chan *store.Store,
	logger *zap.Logger,
) {
	store, err := store.MapResultToStore(r)
	if err != nil {
		logger.Error("error processing store data", zap.Error(err), zap.Any("storeJson", r))
	} else {
		logger.Debug("publishing store", zap.Any("store", store))
		out <- store
	}
}

func processStore(
	ctx context.Context,
	cancel func(),
	wgp *sync.WaitGroup,
	wgs *sync.WaitGroup,
	out chan *store.Store,
	ss *store.StoreService,
	logger *zap.Logger,
) {
	logger.Info("start updating store data")
	count := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("store data update context done")
			return
		case store, ok := <-out:
			if !ok {
				logger.Info("store notification channel closed")
				wgp.Done()
				return
			}
			// if count%1000 == 0 {
			// 	 jg.logger.Debug("processing store", zap.Any("store", store), zap.Int("storeCount", count))
			// }
			success := updateDataStores(ctx, ss, store)
			if !success {
				logger.Error("error processing store data", zap.Any("store", store), zap.Int("storeCount", count))
			}
			count++
			wgs.Done()
		}
	}
}

func updateDataStores(ctx context.Context, ss *store.StoreService, s *store.Store) bool {
	time.Sleep(10 * time.Millisecond)
	return ss.AddStore(ctx, s)
}
