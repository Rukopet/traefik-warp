package traefik_warp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rukopet/traefik-warp/providers"
	"github.com/Rukopet/traefik-warp/providers/cloudflare"
	"github.com/Rukopet/traefik-warp/providers/cloudfront"
)

// Traefik builds one instance per router and entry point on every configuration change;
// psp-billing-prod has 19 routes with this middleware on two entry points.
func TestManyInstancesFetchTheListsOnce(t *testing.T) {
	var cfCalls, cfnCalls atomic.Int32
	cf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfCalls.Add(1)
		w.Write([]byte("173.245.48.0/20\n"))
	}))
	defer cf.Close()
	cfn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfnCalls.Add(1)
		w.Write([]byte(`{"CLOUDFRONT_GLOBAL_IP_LIST":["120.52.22.96/27"],"CLOUDFRONT_REGIONAL_EDGE_IP_LIST":[]}`))
	}))
	defer cfn.Close()

	cloudflare.URLs = []string{cf.URL + "/ips-v4", cf.URL + "/ips-v6"}
	cloudfront.URL = cfn.URL
	cloudflare.Cache = &providers.ListCache{Fetch: cloudflare.Cache.Fetch, TTL: time.Hour, RetryAfter: time.Minute}
	cloudfront.Cache = &providers.ListCache{Fetch: cloudfront.Cache.Fetch, TTL: time.Hour, RetryAfter: time.Minute}

	cfg := CreateConfig()
	cfg.Provider = "auto"
	cfg.AutoRefresh = false
	for i := 0; i < 38; i++ {
		if _, err := New(context.Background(), http.NotFoundHandler(), cfg, "real-ip"); err != nil {
			t.Fatalf("instance %d: %v", i, err)
		}
	}

	if cfCalls.Load() != 2 || cfnCalls.Load() != 1 {
		t.Fatalf("cloudflare %d requests, cloudfront %d; want 2 and 1 for all 38 instances", cfCalls.Load(), cfnCalls.Load())
	}
}
