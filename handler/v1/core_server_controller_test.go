package v1_test

import (
	"context"
	"encoding/json"
	"lucky_project/config"
	"lucky_project/service"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCoreServersAPI(t *testing.T) {
	require.NoError(t, config.InitRedis())
	t.Cleanup(func() {
		_ = config.CloseRedis()
	})

	ctx := context.Background()
	err := config.RedisClient.HSet(
		ctx,
		"core-servers",
		"rtx3090",
		`{"ip":"117.50.174.176","port":23}`,
	).Err()
	require.NoError(t, err)

	w := performRequest(testRouter, http.MethodGet, "/v1/core-servers", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []service.CoreServer
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "response should be a JSON list")
	require.NotEmpty(t, resp)

	found := false
	for _, item := range resp {
		if item.Key != "rtx3090" {
			continue
		}
		found = true
		assert.Equal(t, "117.50.174.176", item.IP)
		assert.Equal(t, 23, item.Port)
	}
	assert.True(t, found, "response list should contain rtx3090")
}
