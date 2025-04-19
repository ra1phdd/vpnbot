package api

import (
	pbServer "nsvpn/pkg/server/v1"
)

func (a *API) GetLoadRequest() (float64, error) {
	if err := a.EnsureConnection(); err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return 0, err
	}

	load, err := a.server.GetLoad(a.ctx, &pbServer.ServerRequest{})
	if err != nil {
		return 0, err
	}
	return load.LoadScore, nil
}
