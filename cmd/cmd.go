// Package cmd for CLI commandline
/*
Copyright © 2022 Regents of the Univsersity of California

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"tsm/config"
	rlog "tsm/log"
)

const (
	// MinSampleInterval is the smallest sample interval in seconds
	MinSampleInterval time.Duration = 1 * time.Second
	// MaxSampleInterval is the largest sample interval in seconds
	MaxSampleInterval time.Duration = 60 * time.Second
)

// config holds parameters for the STATUS command
type cmdService struct {
	Host        string
	Port        string
	Community   string
	args        []string
	TSMCfg      *config.TSMConfig
	snmpService SNMPService
	serializer  TSMSerializer
}

type SNMPService interface {
	InitAndConnect(string, string, string) error
	QueryOids(*[]string) (time.Time, map[string]string, error)
	PollStart(context.Context, *sync.WaitGroup, *[]string, time.Duration) error
	GetScan() (time.Time, *map[string]string, error)
	Close()
}

type TSMCmdService interface {
	Status() error
	Poll() error
	// MBQuery() error
}

// dataOids are the OID endpoints that we will poll the device for
var dataOidInfo []config.OidInfo
var dataOids []string

// staticOids are the OID endpoints that do not change for a given device and FW version
var staticOidInfo []config.OidInfo
var staticOids []string

// allOids are the data+static OIDs
var allOidInfo []config.OidInfo
var allOids []string

var allRegisters []config.OidInfo

// var cfg cmdService
var sigdone chan bool

// model group list is a model groups containing the model Oid,
// model group name and list of models in the group
// var modelGroupOids []string
// var modelGroupMap map[string]string

func NewTSMCmdService(
	host, port, community string,
	args []string,
	snmpSvc SNMPService,
	tsmCfg *config.TSMConfig,
	serial TSMSerializer) TSMCmdService {

	return &cmdService{
		Host:        host,
		Port:        port,
		Community:   community,
		snmpService: snmpSvc,
		args:        args,
		TSMCfg:      tsmCfg,
		serializer:  serial,
	}

}

func init() {

	sigdone = setupSignals(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

}

// func initRegisters(c *cmdService) error {

// 	var err error

// 	return err
// }

// initialize OID vars
func initOids(c *cmdService) error {

	var err error

	// from the config collect dataoids to be polled
	staticOids, staticOidInfo, err = c.TSMCfg.StaticOidsInfo()
	if err != nil {
		return err
	}
	dataOids, dataOidInfo, err = c.TSMCfg.DataOidsInfo()
	if err != nil {
		return err
	}
	// relayOids, relayOidInfo = c.RelayOidsInfo()
	allOids = append(staticOids, dataOids...)
	// allOidInfo = append(staticOidInfo, dataOidInfo...)

	return nil
}

// SetupSignals to trap for external kill signals
func setupSignals(sigs ...os.Signal) chan bool {

	sigchan := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigchan, sigs...)

	go func() {
		sig := <-sigchan
		fmt.Println(sig)
		done <- true
	}()

	return done
}

// queryForModel queies the device to see which model OID is provides a response to
// If OID reporesenting the correct model will trigger a string response containing
// the specific Model name string
func (c *cmdService) queryForModel() (string, string, error) {

	var (
		model      string
		modelGroup string
	)

	err := c.snmpService.InitAndConnect(c.Host, c.Port, c.Community)
	if err != nil {
		return "", "", err
	}
	defer c.snmpService.Close()

	modelGroupOids, modelMap := c.TSMCfg.ModelInfo()

	_, results, err := c.snmpService.QueryOids(modelGroupOids)
	if err != nil {
		return "", "", err
	}

	for _, modinfo := range results {
		if modinfo != "0" {
			model = modinfo
			modelGroup = (*modelMap)[model]
		}
	}
	if model == "" {
		return "", "", errors.New(fmt.Sprintf("Model not found in Model Group OID list [%v]\n", modelGroupOids))
	}

	c.TSMCfg.SetModel(modelGroup)
	rlog.NoticeMsg(fmt.Sprintf("Controller identified as model: %s", modelGroup))

	return model, modelGroup, nil
}
