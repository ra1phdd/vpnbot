package api

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"

	"nsvpn/internal/app/models"
	pbClient "nsvpn/pkg/client/v1"
)

func (a *API) IsFoundRequest(serv *models.Server, uuid string) (bool, error) {
	ctx, err := a.EnsureConnection(serv)
	if err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return false, err
	}

	exists, err := a.client.ClientExists(ctx, &pbClient.ClientExistsRequest{Uuid: uuid})
	if err != nil {
		return false, err
	}
	return exists.GetExists(), nil
}

func (a *API) AddRequest(serv *models.Server, uuid, email string, expiresAt time.Time) error {
	ctx, err := a.EnsureConnection(serv)
	if err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return err
	}

	req := pbClient.CreateClientRequest{
		Uuid:      uuid,
		Email:     email,
		ExpiresAt: timestamppb.New(expiresAt),
	}

	_, err = a.client.CreateClient(ctx, &req)
	if err != nil {
		return err
	}
	return nil
}

func (a *API) UpdateRequest(serv *models.Server, uuid string, expiresAt *time.Time) error {
	ctx, err := a.EnsureConnection(serv)
	if err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return err
	}

	req := pbClient.UpdateClientRequest{
		Uuid: uuid,
	}
	req.ExpiresAt = timestamppb.New(time.Unix(0, 0))
	if expiresAt != nil {
		req.ExpiresAt = timestamppb.New(*expiresAt)
	}

	_, err = a.client.UpdateClient(ctx, &req)
	if err != nil {
		return err
	}
	return nil
}

func (a *API) DeleteRequest(serv *models.Server, uuid string) error {
	ctx, err := a.EnsureConnection(serv)
	if err != nil {
		a.log.Error("Failed to ensure connection before adding client", err)
		return err
	}

	_, err = a.client.DeleteClient(ctx, &pbClient.DeleteClientRequest{
		Uuid: uuid,
	})
	if err != nil {
		return err
	}
	return nil
}
