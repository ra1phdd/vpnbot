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
	"nsvpn/internal/app/models"
	pbClient "nsvpn/pkg/client/v1"
	"nsvpn/pkg/logger"
	pbServer "nsvpn/pkg/server/v1"
	"sync"
)

type API struct {
	log  *logger.Logger
	conn *grpc.ClientConn
	mu   sync.Mutex
	ctx  context.Context
	serv models.Server

	server pbServer.ServerServiceClient
	client pbClient.ClientServiceClient
}

func NewAPI(log *logger.Logger, serv models.Server) *API {
	authKey := sha256.Sum256([]byte(fmt.Sprintf("%s%s", serv.PublicKey, serv.PrivateKey)))

	return &API{
		log:  log,
		serv: serv,
		ctx: metadata.AppendToOutgoingContext(
			context.Background(),
			"X-AUTH-KEY", hex.EncodeToString(authKey[:]),
		),
	}
}

func (a *API) EnsureConnection() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		a.log.Debug("Checking existing gRPC connection state")
		if a.conn.GetState() != connectivity.Shutdown {
			a.log.Debug("gRPC connection is already active")
			return nil
		}

		a.log.Warn("gRPC connection is shutdown, attempting to close the connection")
		if err := a.conn.Close(); err != nil {
			a.log.Error("Failed to close existing gRPC connection", err)
			return err
		}
	}

	a.log.Debug("Creating new gRPC connection")
	var err error
	a.conn, err = grpc.NewClient(fmt.Sprintf("%s:%d", a.serv.IP, a.serv.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		a.log.Error("Failed to connect API server", err)
		return err
	}
	a.server = pbServer.NewServerServiceClient(a.conn)
	a.client = pbClient.NewClientServiceClient(a.conn)

	a.log.Debug("Successfully established gRPC connection to API server", slog.String("address", fmt.Sprintf("%s:%d", a.serv.IP, a.serv.Port)))
	return nil
}
