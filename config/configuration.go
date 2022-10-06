package config

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

var curModelGroup string
var curModelNdx int

// Config interface for RPM
type Config interface {
	Validate() error
}

// NewConfig constructor
func NewConfig() *TSMConfig {
	return &TSMConfig{}
	// return &TSMConfig{CfgFile: cfgFn}
}

// TSMConfig hold the RPM configuration structure
type TSMConfig struct {
	General generalConfig
	Oids    oids
}

// GeneralConfig top lebel config settings
type generalConfig struct {
	Sta string
	Net string
	Loc string
}

// Oids wraps the info for different categrories of
// Oids for a given TS model device
type oids struct {
	EMCOids      []OidInfo
	DeviceGroups []DeviceInfo
}

type DeviceInfo struct {
	GroupOid   string
	ModelGroup string
	Modellist  []string

	Static       []OidInfo
	Status       []OidInfo
	Measurements []OidInfo
	Alarms       []OidInfo
	Faults       []OidInfo
}

// OidInfo holds detailed info for each Oid endpoint
type OidInfo struct {
	Oid          string
	Chancode     string
	Label        string
	Units        string
	Type         string
	Scaling      float64
	Values       []string
	Result       string
	Register     uint16
	RegisterType string
}

// ValueString generates a text string for human readable output of oid value
func (oidInfo *OidInfo) ValueString(resstr string) string {

	switch oidInfo.Type {
	case "string":
		return fmt.Sprintf("%s", resstr)
	case "number":
		val, _ := strconv.ParseFloat(resstr, 64)
		return fmt.Sprintf("%4.1f", val*oidInfo.Scaling)
	case "bitreverse":
		val, _ := strconv.ParseUint(resstr, 10, 64)
		fmtstr := fmt.Sprintf("%%0%db", (int64)(oidInfo.Scaling))
		return fmt.Sprintf(fmtstr, reverseBits(val, uint(oidInfo.Scaling)))
	case "map":
		val, _ := strconv.ParseUint(resstr, 10, 64)
		return fmt.Sprintf("%s", oidInfo.Values[val])
	case "bitmap":
		val, _ := strconv.ParseUint(resstr, 10, 64)
		resstr := bitmapString(val, oidInfo.Values)
		if resstr == "" {
			resstr = "None"
		}
		return fmt.Sprintf("%s", resstr)
	default:
		fmt.Println("ToString WOWZA: ", resstr)
	}
	return resstr
}

// return an array of Status OidInfo for curModel
func (cfg TSMConfig) StaticOids() *[]OidInfo {
	statics := append(cfg.Oids.EMCOids, cfg.Oids.DeviceGroups[curModelNdx].Static...)
	return &statics
}

func (cfg TSMConfig) StatusOids() *[]OidInfo {
	return &cfg.Oids.DeviceGroups[curModelNdx].Status
}

func (cfg TSMConfig) MeasurementOids() *[]OidInfo {
	return &cfg.Oids.DeviceGroups[curModelNdx].Measurements
}

func (cfg TSMConfig) AlarmOids() *[]OidInfo {
	return &cfg.Oids.DeviceGroups[curModelNdx].Alarms
}

func (cfg TSMConfig) FaultOids() *[]OidInfo {
	return &cfg.Oids.DeviceGroups[curModelNdx].Faults
}

// Validate the rpm TOML config file
func (cfg TSMConfig) Validate() (e error) {
	return nil
}

// DumpCfg writes config to string for printing/saving
func (cfg *TSMConfig) DumpCfg(writer io.Writer) {

	fmt.Fprintf(writer, "%v\n", *&cfg.General)
	// fmt.Fprintf(writer, "%v\n", *&cfg.WinMain)
	for _, deviceGroup := range cfg.Oids.DeviceGroups {
		fmt.Fprintf(writer, "%v\n", deviceGroup.GroupOid)
		fmt.Fprintf(writer, "%v\n", deviceGroup.ModelGroup)
		fmt.Fprintf(writer, "%v\n", deviceGroup.Modellist)
		listlist := [][]OidInfo{deviceGroup.Static, deviceGroup.Status, deviceGroup.Measurements, deviceGroup.Alarms, deviceGroup.Faults}
		for _, list := range listlist {
			for _, detail := range list {
				fmt.Fprintf(writer, "%v\n", detail)
			}
		}
	}
	// fmt.Fprintf(writer, "%v\n", *&cfg.CfgFile)

	return
}

// DataOidsInfo is a convenience func to generate an ordered list of OIDS that have real data for polling/querying
func (cfg *TSMConfig) DataOidsInfo() ([]string, []OidInfo, error) {

	if curModelGroup == "" {
		return nil, nil, errors.New("Model moust be set before calling DataOidsInfo")
	}

	cnt := len(cfg.Oids.DeviceGroups[curModelNdx].Status) +
		len(cfg.Oids.DeviceGroups[curModelNdx].Measurements) +
		len(cfg.Oids.DeviceGroups[curModelNdx].Alarms) +
		len(cfg.Oids.DeviceGroups[curModelNdx].Faults)

	oids := make([]string, 0, cnt)
	oidInfo := make([]OidInfo, 0, cnt)
	oidInfo = append(oidInfo, cfg.Oids.DeviceGroups[curModelNdx].Status...)
	oidInfo = append(oidInfo, cfg.Oids.DeviceGroups[curModelNdx].Measurements...)
	oidInfo = append(oidInfo, cfg.Oids.DeviceGroups[curModelNdx].Alarms...)
	oidInfo = append(oidInfo, cfg.Oids.DeviceGroups[curModelNdx].Faults...)

	for _, oidinfo := range oidInfo {
		oids = append(oids, oidinfo.Oid)
	}

	return oids, oidInfo, nil

}

// StaticOidsInfo is a convenience func to generate an ordered list of OIDS that have device static values
func (cfg *TSMConfig) StaticOidsInfo() ([]string, []OidInfo, error) {

	if curModelGroup == "" {
		return nil, nil, errors.New("Model moust be set before calling StaticOidsInfo")
	}

	cnt := len(cfg.Oids.DeviceGroups[curModelNdx].Static)

	oidInfo := make([]OidInfo, 0, cnt)
	oidInfo = append(oidInfo, cfg.Oids.EMCOids...)
	oidInfo = append(oidInfo, cfg.Oids.DeviceGroups[curModelNdx].Static...)

	oids := make([]string, 0, cnt)
	for _, oidinfo := range oidInfo {
		oids = append(oids, oidinfo.Oid)
	}

	return oids, oidInfo, nil

}

// ModelOidsInfo is a convenience func to generate an ordered list of OIDS that have device static values
func (cfg *TSMConfig) ModelInfo() (*[]string, *map[string]string) {

	cnt := len(cfg.Oids.DeviceGroups)

	modelMap := make(map[string]string, cnt)
	modelOids := make([]string, 0, cnt)

	// oids := make([]string, 0, cnt)
	for _, devGroup := range cfg.Oids.DeviceGroups {
		modelOids = append(modelOids, devGroup.GroupOid)
		for _, model := range devGroup.Modellist {
			modelMap[model] = devGroup.ModelGroup
		}
	}

	return &modelOids, &modelMap

}

func (cfg *TSMConfig) SetModel(model string) {
	for ndx, devGroup := range cfg.Oids.DeviceGroups {
		if model == devGroup.ModelGroup {
			curModelGroup = model
			curModelNdx = ndx
		}
	}
}

func reverseBits(num uint64, len uint) uint64 {
	var ret = uint64(0)
	var power = uint64(len - 1)
	for num != 0 {
		ret += (num & 1) << power
		num = num >> 1
		power -= 1
	}
	return ret
}

func bitmapString(bits uint64, bitmap []string) string {
	res := ""
	bmlen := len(bitmap)
	for ndx := 0; ndx < 64; ndx++ {
		if ndx < bmlen {
			if (bits & uint64(math.Pow(2, float64(ndx)))) != 0 {
				res += fmt.Sprintf("%s, ", bitmap[ndx])
			}
		}
	}
	return strings.TrimRight(res, " ,")
}
