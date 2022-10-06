package main

/*
Copyright © 2020 Regents of the University of California

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

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"tsm/cmd"
	"tsm/config"
	l "tsm/log"
	"tsm/serializers/tui"
	"tsm/snmp"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type appConfig struct {
	debug     bool
	cfgFile   string
	cmd       string
	host      string
	port      string
	runAsUser string
	community string
	tsmCfg    *config.TSMConfig
}

func initLogging(debug bool) error {

	severityLevel := syslog.LOG_NOTICE
	if debug {
		severityLevel = syslog.LOG_DEBUG
	}
	err := l.InitLogging("tsm", syslog.LOG_LOCAL0|severityLevel)
	if err != nil {
		return err
	}
	// if appCfg.debug {
	l.SetLogLevel(severityLevel)
	// }
	return nil
}

func logStartup() error {

	abspath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}
	l.NoticeMsg(fmt.Sprintf("%s starting up...", abspath))
	return nil
}

func setUser(username string) error {

	var uid, gid int
	nrtsuser, err := user.Lookup(username)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.WarningMsg(err.Error())
	} else {
		if uid, err = strconv.Atoi(nrtsuser.Uid); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			l.ErrMsg("could not convert uid %v to int", uid)
			l.ErrMsg(err.Error())
		}
		if gid, err = strconv.Atoi(nrtsuser.Gid); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			l.ErrMsg("could not convert gid %v to int", gid)
			l.ErrMsg(err.Error())
		}
		syscall.Setuid(uid)
		syscall.Setgid(gid)
		os.Chdir(nrtsuser.HomeDir)
		l.NoticeMsg(fmt.Sprintf("nrtsuser.HomeDir: %s", nrtsuser.HomeDir))

		var wd string
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			l.ErrMsg("could not determine working dir", gid)
			l.ErrMsg(err.Error())
		} else {
			l.NoticeMsg(fmt.Sprintf("working dir: %s", wd))
		}
	}
	return err
}

// read CLI flags adjust app config appropriately
func (c *appConfig) readCLI(params []string) error {

	var err error

	// sanity check on params; needs at least host[:port] and cmd
	if len(params) < 2 {
		err := errors.New("command line error, not enough parameters")
		return err
	}

	// parse host[]:port]
	hostport := flag.Args()[0]
	c.host, c.port, err = formatHostPort(hostport)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.ErrMsg(err.Error())
		os.Exit(1)
	}

	// get command
	cmd := params[1]
	if !validCmd(cmd) {
		err = fmt.Errorf("\ninvalid command: %s", cmd)
		return err
	}
	c.cmd = cmd

	return err
}

// initConfig reads in config file and ENV variables if set.
func loadConfig(tsmCfgFile string) (*config.TSMConfig, error) {

	var err error

	if tsmCfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(tsmCfgFile)
	} else {

		// Search for config cwd first, then in home directory.
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/dev/tsm")
		viper.AddConfigPath("$HOME/etc")
		viper.SetConfigName("tsm")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		return nil, err
	}
	l.NoticeMsg(fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()))

	tsmCfg := config.NewConfig()
	if err := viper.Unmarshal(&tsmCfg); err != nil {
		return nil, err
	}
	// tsmCfg.CfgFile = viper.ConfigFileUsed()

	if err := tsmCfg.Validate(); err != nil {
		fmt.Printf(err.Error())
		return nil, err
	}

	return tsmCfg, err
}

func formatHostPort(rawHost string) (string, string, error) {

	if strings.Index(rawHost, ":") == -1 {
		rawHost += ":161"
	}

	h, p, _ := net.SplitHostPort(rawHost)
	ips, err := net.LookupHost(h)
	if err != nil {
		return "", "", err
	}

	return ips[0], p, nil

}

func executeCmd(cmd string, cmdSvc cmd.TSMCmdService) {

	var err error

	switch cmd {
	// case "poll":
	// 	err = cmdSvc.Poll()
	case "status":
		err = cmdSvc.Status()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err.Error())
		l.ErrMsg(err.Error())
	}
}

func validCmd(cmd string) bool {
	validCommands := []string{
		// "poll",
		"status",
		// "mb",
	}
	for _, n := range validCommands {
		if cmd == n {
			return true
		}
	}
	return false
}

func defineGlobalFlags(appCfg *appConfig) {

	flag.BoolVar(&appCfg.debug, "d", false, "enable debug logging")
	flag.BoolVar(&appCfg.debug, "debug", false, "enable debug logging")
	flag.StringVar(&appCfg.cfgFile, "c", "", "specify TSM config file")
	flag.StringVar(&appCfg.cfgFile, "config", "", "specify TSM config file")
	flag.StringVar(&appCfg.runAsUser, "u", appCfg.runAsUser, "specify username instead of booger")
	flag.StringVar(&appCfg.runAsUser, "user", appCfg.runAsUser, "specify user to run as")
	flag.StringVar(&appCfg.community, "community", appCfg.community, "specify snmp read community")
	flag.Parse()

}

func main() {

	var err error
	var tsmCfg *config.TSMConfig
	var appCfg = &appConfig{
		debug:     false,
		cfgFile:   "",
		cmd:       "",
		host:      "",
		port:      "",
		runAsUser: "nrts",
		community: "public",
		tsmCfg:    nil,
	}

	defineGlobalFlags(appCfg)

	err = initLogging(appCfg.debug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating logger (fprintf)")
		log.Fatal("error creating logger (log.fatal)")
	}

	err = appCfg.readCLI(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.ErrMsg(err.Error())
		flag.Usage()
		os.Exit(1)
	}

	err = logStartup()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.ErrMsg(err.Error())
		os.Exit(1)
	}

	err = setUser(appCfg.runAsUser)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.ErrMsg(err.Error())
		os.Exit(1)
	}

	// read tsm config file
	tsmCfg, err = loadConfig(appCfg.cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		l.ErrMsg(err.Error())
		os.Exit(1)
	}

	tuiLizer := tui.NewTui(appCfg.host, appCfg.port)

	snmpSvc := snmp.NewSnmpService()
	cmdSvc := cmd.NewTSMCmdService(
		appCfg.host, appCfg.port, appCfg.community, flag.Args(),
		snmpSvc, tsmCfg, tuiLizer)

	executeCmd(appCfg.cmd, cmdSvc)

	l.NoticeMsg("%s shutting down", os.Args[0])
}
