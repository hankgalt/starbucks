package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func DefaulTestStoreJSON(id uint32, name, city string) map[string]interface{} {
	s := map[string]interface{}{
		"city":      city,
		"name":      name,
		"country":   "CN",
		"longitude": 114.20169067382812,
		"latitude":  22.340700149536133,
		"store_id":  id,
	}
	return s
}

func TestMapResultToStore(t *testing.T) {
	id, name, city := uint32(1), "Plaza Hollywood", "Hong Kong"
	storeJSON := DefaulTestStoreJSON(id, name, city)

	store, err := MapResultToStore(storeJSON)
	require.NoError(t, err)

	assert.Equal(t, store.Id, id, "ID should be mapped")
	assert.Equal(t, store.Name, name, "name should be mapped")
	assert.Equal(t, store.City, city, "city should be mapped")
}

func TestAddStoreGetStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	css := New(logger)
	testAddStore(t, css)
	testGetStoreStats(t, css, 1)
}

func testAddStore(t *testing.T, ss *StoreService) {
	t.Helper()
	id, name, city := 1, "Plaza Hollywood", "Hong Kong"
	storeJSON := DefaulTestStoreJSON(uint32(id), name, city)

	store, err := MapResultToStore(storeJSON)
	require.NoError(t, err)

	ctx := context.Background()
	ok, err := ss.AddStore(ctx, store)
	require.NoError(t, err)
	assert.Equal(t, ok, true, "adding new store should be success")
}

func testGetStoreStats(t *testing.T, ss *StoreService, count int) {
	t.Helper()
	storeStats := ss.GetStoreStats()
	assert.Equal(t, storeStats.Count, count, "store count should be equal to added store count")
	assert.Equal(t, storeStats.HashCount, count, "geo hash count should be equal to added store count")
	assert.Equal(t, storeStats.Ready, false, "store ready status should be false")
}
