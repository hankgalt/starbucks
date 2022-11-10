package server

import (
	"context"
	"net"
	"testing"

	"github.com/hankgalt/starbucks/internal/auth"
	"github.com/hankgalt/starbucks/internal/config"
	"github.com/hankgalt/starbucks/pkg/services/store"

	api "github.com/hankgalt/starbucks/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func defaulAddStoreRequest(id uint32, name, city string) *api.AddStoreRequest {
	s := &api.AddStoreRequest{
		City:      city,
		Name:      name,
		Country:   "CN",
		Longitude: 114.20169067382812,
		Latitude:  22.340700149536133,
		StoreId:   id,
	}
	return s
}

func testGetStore(t *testing.T, client api.StoresClient, nbClient api.StoresClient, config *Config) {
	t.Helper()
	id, name, city := 1, "Plaza Hollywood", "Hong Kong"
	addStoreReq := defaulAddStoreRequest(uint32(id), name, city)

	ctx := context.Background()
	addStoreRes, err := client.AddStore(ctx, addStoreReq)
	require.NoError(t, err)
	assert.Equal(t, addStoreRes.Ok, true, "adding new store should be success")

	_, err = nbClient.AddStore(ctx, addStoreReq)
	require.Error(t, err)
}

func testUnauthorizedGetStore(t *testing.T, client api.StoresClient, nbClient api.StoresClient, config *Config) {
	t.Helper()
	id, name, city := 1, "Plaza Hollywood", "Hong Kong"
	addStoreReq := defaulAddStoreRequest(uint32(id), name, city)

	ctx := context.Background()
	addStoreRes, err := nbClient.AddStore(ctx, addStoreReq)
	require.Error(t, err)
	assert.Equal(t, addStoreRes, (*api.AddStoreResponse)(nil), "adding new store by an unauthorized client should fail")
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.StoresClient,
		nbclient api.StoresClient,
		config *Config,
	){
		"get store by id suceeds":  testGetStore,
		"test unauthorized client": testUnauthorizedGetStore,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, nbClient, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, nbClient, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (
	client api.StoresClient,
	nbClient api.StoresClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (*grpc.ClientConn, api.StoresClient, []grpc.DialOption) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})
		require.NoError(t, err)

		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client = api.NewStoresClient(conn)
		return conn, client, opts
	}

	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ := newClient(config.RootClientCertFile, config.RootClientKeyFile)

	var nbConn *grpc.ClientConn
	nbConn, nbClient, _ = newClient(config.NobodyClientCertFile, config.NobodyClientKeyFile)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	// cc, err := grpc.Dial(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	// require.NoError(t, err)

	logger := zaptest.NewLogger(t)
	css := store.New(logger)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)
	cfg = &Config{
		StoreService: css,
		Authorizer:   authorizer,
	}

	if fn != nil {
		fn(cfg)
	}
	server, err := NewGrpcServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return rootClient, nbClient, cfg, func() {
		server.Stop()
		rootConn.Close()
		nbConn.Close()
		l.Close()
		css.Clear()
	}
}
