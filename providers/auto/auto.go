package auto

import (
	"github.com/Rukopet/traefik-warp/providers/cloudflare"
	"github.com/Rukopet/traefik-warp/providers/cloudfront"
)

// TrustedIPS returns the union of Cloudflare and CloudFront trusted ranges.
func TrustedIPS() []string {
	var merged []string
	merged = append(merged, cloudflare.TrustedIPS()...)
	merged = append(merged, cloudfront.TrustedIPS()...)
	return merged
}
