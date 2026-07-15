package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/network/apigateway"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/apigateway/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestAPIGatewayMemoryCreateGetList(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	api, err := mgr.CreateAPI(ctx, apigateway.CreateAPIOptions{
		Name:        "orders",
		Description: "Orders API",
		Type:        apigateway.APITypeHTTP,
	})
	require.NoError(t, err)
	require.NotEmpty(t, api.ID)
	require.Equal(t, "orders", api.Name)
	require.Equal(t, apigateway.APITypeHTTP, api.Type)
	require.NotEmpty(t, api.Endpoint)

	got, err := mgr.GetAPI(ctx, api.ID)
	require.NoError(t, err)
	require.Equal(t, api.ID, got.ID)

	list, err := mgr.ListAPIs(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestAPIGatewayMemoryRoutesAndDeploy(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	api, err := mgr.CreateAPI(ctx, apigateway.CreateAPIOptions{Name: "catalog"})
	require.NoError(t, err)

	route, err := mgr.AddRoute(ctx, api.ID, apigateway.Route{
		Method: "GET",
		Path:   "/items",
		Integration: apigateway.Integration{
			Type: "HTTP",
			URI:  "https://backend.example.com/items",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, route.ID)

	got, err := mgr.GetAPI(ctx, api.ID)
	require.NoError(t, err)
	require.Len(t, got.Routes, 1)

	stage, err := mgr.Deploy(ctx, api.ID, "prod")
	require.NoError(t, err)
	require.Equal(t, "prod", stage.Name)

	gotStage, err := mgr.GetStage(ctx, api.ID, "prod")
	require.NoError(t, err)
	require.Equal(t, "prod", gotStage.Name)

	require.NoError(t, mgr.RemoveRoute(ctx, api.ID, route.ID))
	got, err = mgr.GetAPI(ctx, api.ID)
	require.NoError(t, err)
	require.Empty(t, got.Routes)
}

func TestAPIGatewayMemoryNotFound(t *testing.T) {
	mgr := memory.New()
	ctx := context.Background()

	_, err := mgr.GetAPI(ctx, "missing")
	require.ErrorIs(t, err, apigateway.ErrAPINotFound)

	err = mgr.DeleteAPI(ctx, "missing")
	require.ErrorIs(t, err, apigateway.ErrAPINotFound)

	err = mgr.RemoveRoute(ctx, "missing", "r1")
	require.ErrorIs(t, err, apigateway.ErrAPINotFound)

	api, err := mgr.CreateAPI(ctx, apigateway.CreateAPIOptions{Name: "x"})
	require.NoError(t, err)

	err = mgr.RemoveRoute(ctx, api.ID, "missing-route")
	require.ErrorIs(t, err, apigateway.ErrRouteNotFound)

	_, err = mgr.GetStage(ctx, api.ID, "missing-stage")
	require.ErrorIs(t, err, apigateway.ErrStageNotFound)

	require.NoError(t, mgr.DeleteAPI(ctx, api.ID))
}
