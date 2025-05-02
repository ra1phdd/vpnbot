package api

import (
	"nsvpn/internal/app/models"
	pbServer "nsvpn/pkg/server/v1"
)

func (a *API) GetLoadRequest(serv *models.Server) (float64, error) {
	ctx, err := a.EnsureConnection(serv)
	if err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return 0, err
	}

	load, err := a.server.GetLoad(ctx, &pbServer.ServerRequest{})
	if err != nil {
		return 0, err
	}
	return load.GetLoadScore(), nil
}
