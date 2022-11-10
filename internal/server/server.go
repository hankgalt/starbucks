package server

import (
	"context"

	api "github.com/hankgalt/starbucks/api/v1"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/hankgalt/starbucks/pkg/services/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var _ api.StoresServer = (*grpcServer)(nil)

type Config struct {
	StoreService *store.StoreService
	Authorizer   Authorizer
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

const (
	objectWildcard = "*"
	addStoreAction = "addStore"
)

type grpcServer struct {
	api.UnimplementedStoresServer
	*Config
}

func NewGrpcServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_auth.UnaryServerInterceptor(authenticate),
	)))

	gsrv := grpc.NewServer(opts...)
	srv, err := newGrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterStoresServer(gsrv, srv)
	return gsrv, nil
}

// func defaultServerOptions() (*tls.Config, error) {
// 	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
// 		CertFile:      config.ServerCertFile,
// 		KeyFile:       config.ServerKeyFile,
// 		CAFile:        config.CAFile,
// 		ServerAddress: l.Addr().String(),
// 		Server:        true,
// 	})
// }

func newGrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

func (s *grpcServer) GetStore(ctx context.Context, req *api.GetStoreRequest) (*api.GetStoreResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		addStoreAction,
	); err != nil {
		return nil, err
	}

	store, err := s.StoreService.GetStore(ctx, req.StoreId)
	if err != nil {
		st := status.New(codes.NotFound, "store not found")
		return nil, st.Err()
	}
	return &api.GetStoreResponse{
		Store: &api.Store{
			Id:        store.Id,
			Name:      store.Name,
			Longitude: float32(store.Longitude),
			Latitude:  float32(store.Latitude),
			City:      store.City,
			Country:   store.Country,
		},
	}, nil
}

func (s *grpcServer) AddStore(ctx context.Context, req *api.AddStoreRequest) (*api.AddStoreResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		addStoreAction,
	); err != nil {
		return nil, err
	}

	ok, err := s.StoreService.AddStore(ctx, &store.Store{
		Id:        req.StoreId,
		Name:      req.Name,
		Longitude: float64(req.Longitude),
		Latitude:  float64(req.Latitude),
		City:      req.City,
		Country:   req.Country,
	})
	if !ok || err != nil {
		st := status.New(codes.AlreadyExists, "store already exists")
		return nil, st.Err()
	}
	return &api.AddStoreResponse{
		Ok: ok,
	}, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
