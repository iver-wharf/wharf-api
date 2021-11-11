package modelconv

import "github.com/iver-wharf/wharf-core/pkg/logger"

var log = logger.NewScoped("WHARF")

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
