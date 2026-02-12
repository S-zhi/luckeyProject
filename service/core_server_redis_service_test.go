package service

import (
	"context"
	"lucky_project/config"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initRedisForTest(t *testing.T) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")

	projectRoot := filepath.Dir(filepath.Dir(currentFile))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectRoot))
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	require.NoError(t, config.InitConfig())
	require.NoError(t, config.InitRedis())
	t.Cleanup(func() {
		_ = config.CloseRedis()
	})
}

func TestRedisConnection(t *testing.T) {
	initRedisForTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pong, err := config.RedisClient.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)
}

func TestListCoreServers(t *testing.T) {
	initRedisForTest(t)

	ctx := context.Background()
	err := config.RedisClient.HSet(
		ctx,
		coreServersHashKey,
		"rtx3090",
		`{"ip":"117.50.174.176","port":23}`,
	).Err()
	require.NoError(t, err)

	servers, err := ListCoreServers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, servers)

	found := false
	for _, server := range servers {
		if server.Key != "rtx3090" {
			continue
		}
		found = true
		assert.Equal(t, "117.50.174.176", server.IP)
		assert.Equal(t, 23, server.Port)
	}
	assert.True(t, found, "core-servers should contain key rtx3090")
}

func TestGetCoreServerByKey(t *testing.T) {
	initRedisForTest(t)

	ctx := context.Background()
	err := config.RedisClient.HSet(
		ctx,
		coreServersHashKey,
		"rtx4090",
		`{"ip":"10.10.10.10","port":2222}`,
	).Err()
	require.NoError(t, err)

	server, err := GetCoreServerByKey(ctx, "rtx4090")
	require.NoError(t, err)
	assert.Equal(t, "rtx4090", server.Key)
	assert.Equal(t, "10.10.10.10", server.IP)
	assert.Equal(t, 2222, server.Port)
}

func TestGetCoreServerByKeyNotFound(t *testing.T) {
	initRedisForTest(t)

	_, err := GetCoreServerByKey(context.Background(), "not-exists-core-server")
	assert.ErrorIs(t, err, ErrCoreServerNotFound)
}
