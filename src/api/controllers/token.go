package controllers

import (
	"context"
	"server/src/schemas"
)

func (c *Controller) PostToken(ctx context.Context, username, password string) (*schemas.TokenResponse, error) {
	return c.ESCOClient.PostToken(ctx, username, password)
}
