package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"log/slog"
	"sync"

	"nsvpn/internal/app/models"
	pbClient "nsvpn/pkg/client/v1"
	"nsvpn/pkg/logger"
	pbServer "nsvpn/pkg/server/v1"
)

type ServerConnection struct {
	conn *grpc.ClientConn
	mu   sync.Mutex
	ctx  context.Context
	serv *models.Server
}

type API struct {
	log     *logger.Logger
	servers map[string]*ServerConnection
	server  pbServer.ServerServiceClient
	client  pbClient.ClientServiceClient
}

func NewAPI(log *logger.Logger) *API {
	return &API{
		log:     log,
		servers: make(map[string]*ServerConnection),
	}
}

func (a *API) EnsureConnection(serv *models.Server) (context.Context, error) {
	data, exists := a.servers[fmt.Sprintf("%s:%d", serv.IP, serv.Port)]
	if !exists {
		authKey := sha256.Sum256([]byte(fmt.Sprintf("%s%s", serv.Country.PublicKey, serv.Country.PrivateKey)))

		data = &ServerConnection{
			ctx: metadata.AppendToOutgoingContext(
				context.Background(),
				"X-AUTH-KEY", hex.EncodeToString(authKey[:]),
			),
			serv: serv,
		}
		a.servers[fmt.Sprintf("%s:%d", serv.IP, serv.Port)] = data
	}

	data.mu.Lock()
	defer data.mu.Unlock()

	if data.conn != nil {
		a.log.Debug("Checking existing gRPC connection state")
		if data.conn.GetState() != connectivity.Shutdown {
			a.log.Debug("gRPC connection is already active")
			return data.ctx, nil
		}

		a.log.Warn("gRPC connection is shutdown, attempting to close the connection")
		if err := data.conn.Close(); err != nil {
			a.log.Error("Failed to close existing gRPC connection", err)
			return nil, err
		}
	}

	a.log.Debug("Creating new gRPC connection")
	var err error
	data.conn, err = grpc.NewClient(fmt.Sprintf("%s:%d", serv.IP, serv.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		a.log.Error("Failed to connect API server", err)
		return nil, err
	}
	a.server = pbServer.NewServerServiceClient(data.conn)
	a.client = pbClient.NewClientServiceClient(data.conn)

	a.log.Debug("Successfully established gRPC connection to API server", slog.String("address", fmt.Sprintf("%s:%d", serv.IP, serv.Port)))
	return data.ctx, nil
}
