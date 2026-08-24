package httpmw_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/internal/shared/httpmw"
	"github.com/stretchr/testify/require"
)

// TestIPKey — the keyFn must namespace buckets by client IP so distinct
// endpoints (auth vs public apply) never share a rate-limit window.
func TestIPKey(t *testing.T) {
	app := fiber.New()
	var got, rawIP string
	app.Get("/", func(c *fiber.Ctx) error {
		rawIP = c.IP()
		got = httpmw.IPKey("public-apply:")(c)
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	require.Equal(t, "public-apply:"+rawIP, got)
	require.NotEqual(t, "public-apply:", got, "IP part must be present")
}

func TestIPKey_DistinctPrefixes(t *testing.T) {
	app := fiber.New()
	var authKey, applyKey string
	app.Get("/", func(c *fiber.Ctx) error {
		authKey = httpmw.IPKey("auth:")(c)
		applyKey = httpmw.IPKey("public-apply:")(c)
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	require.NotEqual(t, authKey, applyKey)
	require.True(t, len(applyKey) > len(authKey))
}
