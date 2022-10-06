package cmd

import (
	"time"
	"tsm/config"
)

type TSMSerializer interface {
	Format(time.Time, string, string, *map[string]string, *config.TSMConfig) string
}
