package api

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	pbClient "nsvpn/pkg/client/v1"
	"time"
)

func (a *API) IsFoundRequest(uuid string) (bool, error) {
	if err := a.EnsureConnection(); err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return false, err
	}

	exists, err := a.client.ClientExists(a.ctx, &pbClient.ClientExistsRequest{Uuid: uuid})
	if err != nil {
		return false, err
	}
	return exists.Exists, nil
}

func (a *API) AddRequest(uuid, email string, expiresAt time.Time) error {
	if err := a.EnsureConnection(); err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return err
	}

	_, err := a.client.CreateClient(a.ctx, &pbClient.CreateClientRequest{
		Uuid:      uuid,
		Email:     email,
		ExpiresAt: timestamppb.New(expiresAt),
	})
	if err != nil {
		return err
	}
	return nil
}
