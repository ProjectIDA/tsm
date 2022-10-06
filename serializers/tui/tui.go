package tui

import (
	"fmt"
	"time"
	"tsm/config"
)

type tuiCfg struct {
	Host string
	Port string
}

func NewTui(host, port string) *tuiCfg {
	return &tuiCfg{
		Host: host,
		Port: port,
	}
}

func (t *tuiCfg) Format(ts time.Time, host, port string, results *map[string]string, cfg *config.TSMConfig) string {

	result := "\n"
	result += fmt.Sprintf("%40s:  %s\n", "Time of Query", ts.Format("2006-01-02 15:04:05 MST"))
	result += fmt.Sprintf("%40s:  %s:%s\n\n", "Host", t.Host, t.Port)

	for _, oidInfo := range *cfg.StaticOids() {
		result += fmt.Sprintf("%40s:  %s %s\n", oidInfo.Label, (*results)[oidInfo.Oid], oidInfo.Units)
	}
	result += "\n"

	for _, oidInfo := range *cfg.StatusOids() {
		result += fmt.Sprintf("%40s:  %s %s\n", oidInfo.Label, oidInfo.ValueString((*results)[oidInfo.Oid]), oidInfo.Units)
	}
	result += "\n"

	for _, oidInfo := range *cfg.MeasurementOids() {
		result += fmt.Sprintf("%40s:  %s %s\n", oidInfo.Label, oidInfo.ValueString((*results)[oidInfo.Oid]), oidInfo.Units)
	}
	result += "\n"

	for _, oidInfo := range *cfg.AlarmOids() {
		result += fmt.Sprintf("%40s:  %s %s\n", oidInfo.Label, oidInfo.ValueString((*results)[oidInfo.Oid]), oidInfo.Units)
	}
	result += "\n"

	for _, oidInfo := range *cfg.FaultOids() {
		result += fmt.Sprintf("%40s:  %s %s\n", oidInfo.Label, oidInfo.ValueString((*results)[oidInfo.Oid]), oidInfo.Units)
	}

	return result
}
