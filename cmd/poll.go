// Package cmd for CLI commandline
/*
Copyright © 2020 Regents of the University of California

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	"tsm/config"
	rlog "tsm/log"
)

func getSampleInterval(intstr string) (float64, error) {

	val, err := strconv.ParseFloat(intstr, 32)
	if err != nil {
		return 0, err
	}

	if (val < MinSampleInterval.Seconds()) || (val > MaxSampleInterval.Seconds()) {
		rlog.ErrMsg(fmt.Sprintf("invalid sample interval %f must be between %.0f and %.0f seconds",
			val,
			MinSampleInterval.Seconds(),
			MaxSampleInterval.Seconds()))
		return 0, fmt.Errorf("invalid sample interval %f must be between %.0f and %.0f seconds",
			val,
			MinSampleInterval.Seconds(),
			MaxSampleInterval.Seconds())
	}

	return val, nil
}

func formatScan(sampleInterval time.Duration, cfg *config.TSMConfig, ts time.Time, scan *map[string]string) string {

	outstr := fmt.Sprintf(
		"%04d %02d %02d %02d %02d %02d",
		ts.Year(),
		ts.Month(),
		ts.Day(),
		ts.Hour(),
		ts.Minute(),
		ts.Second(),
	)

	outstr += fmt.Sprintf(" %s %s %s %.0f",
		cfg.General.Net,
		cfg.General.Sta,
		cfg.General.Loc,
		sampleInterval.Seconds(),
	)

	_, oidInfos, _ := cfg.DataOidsInfo()
	for _, oidinfo := range oidInfos {
		// fmt.Printf("oidinfo: %v\n", oidinfo)
		oid := oidinfo.Oid
		// outstr += fmt.Sprintf(" %s:%s", oidinfo.Chancode, (*scan)[oid])
		outstr += fmt.Sprintf(" %s:%s", oidinfo.Chancode, oidinfo.ValueString((*scan)[oid]))
	}

	return outstr

}

func logDeviceInfo(ts time.Time, scan *map[string]string) {

	for _, oidinfo := range staticOidInfo {
		rlog.NoticeMsg("%s: %s", oidinfo.Label, (*scan)[oidinfo.Oid])
	}

}

func pollArgsParse(args []string) (time.Duration, error) {

	var dInterval time.Duration

	if len(args) < 1 {
		err := errors.New("not enough parameters, polling interval must be specified")
		return dInterval, err
	}
	intervalSecsf64, err := getSampleInterval(args[2])
	if err != nil {
		return dInterval, err
	}

	fInterval := float32(intervalSecsf64)
	dInterval = time.Duration(fInterval) * time.Second

	return dInterval, nil

}

// Poll the device
func (c *cmdService) Poll() error {

	var (
		modelGroup string
		err        error
	)

	rlog.NoticeMsg(fmt.Sprintf("running %s command on host: %s:%s\n", c.args[0], c.Host, c.Port))

	_, modelGroup, err = c.queryForModel()
	c.TSMCfg.SetModel(modelGroup)
	rlog.NoticeMsg(fmt.Sprintf("Controller identified as model: %s", modelGroup))

	dInterval, err := pollArgsParse(c.args)
	if err != nil {
		return err
	}
	hInterval := dInterval / 2

	rlog.NoticeMsg(fmt.Sprintf("polling interval: %.0f sec(s)\n", dInterval.Seconds()))

	err = c.snmpService.InitAndConnect(c.Host, c.Port, c.Community)
	if err != nil {
		return err
	}
	defer c.snmpService.Close()

	initOids(c)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	err = c.snmpService.PollStart(ctx, &wg, &allOids, dInterval)
	if err != nil {
		rlog.ErrMsg("could not start internal polling loop... quitting")
		cancel()
		wg.Wait()
		return err
	}
	rlog.NoticeMsg("internal polling loop spawned")

	var (
		scan, prevScan *map[string]string
		ts             time.Time
		offset         time.Duration
	)

	targetTime := time.Now().Round(dInterval).Add(dInterval)
	first := true
	scanMissed := false
	scanRepeated := false
	exiting := false

	for !exiting {

		targetTime = targetTime.Add(dInterval)
		rlog.DebugMsg("next target time: %v\n", targetTime.String())

		select {

		case <-time.After(time.Until(targetTime)):

			prevScan = scan
			ts, scan, err = c.snmpService.GetScan()
			if scan == nil {
				if !scanMissed {
					rlog.ErrMsg("no rpm scan available\n")
				}
				scanMissed = true
				first = true
				continue
			}

			rlog.DebugMsg("Scan time:   %s", ts.String())
			for _, oidinfo := range dataOidInfo {
				rlog.DebugMsg("(%s) %s: %s", oidinfo.Chancode, oidinfo.Oid, (*scan)[oidinfo.Oid])
			}

			if first {
				rlog.NoticeMsg("initial rpm scan received")
				logDeviceInfo(ts, scan)
				first = false
			}
			scanMissed = false

		case <-sigdone:
			rlog.DebugMsg("got done signal")
			exiting = true
			continue
		}

		// calcualte offset of scan time from target time.
		// positive offset means scan time is after target time
		offset = ts.Sub(targetTime)

		// see if scan should go with next second, perhaps due to network lag

		if offset > hInterval {
			// really should never get here unless this loop is taking more than an interval to complete
			rlog.WarningMsg("well, this is awkward, a scan from more than 1/2 interval in the future")
			rlog.WarningMsg("incrementing TargetTime by one interval (to catch up) creating a gap")
			targetTime = ts.Round(dInterval) //targetTime.Add(dInterval)
			first = true
		} else if offset < -hInterval {
			// current scan does not appear to be available.
			rlog.ErrMsg("missing scan: current scan time (%v) not found within 1/2 interval of target (%v)", ts, targetTime)

			// if previous scan not alreadcy repeated, repeat previous scan (if it exists)
			// and set flag so con only do this one time in a row.
			if (prevScan != nil) && (!scanRepeated) {
				rlog.WarningMsg("repeating previous scan value")
				scanRepeated = true
				scan = prevScan
			} else {
				// missed scan but can't repeat previous, so there will be a gap
				first = true
			}
			continue
		}

		scanRepeated = false
		// send record to Stdout
		fmt.Printf("%s\n", formatScan(dInterval, c.TSMCfg, ts, scan))

	}
	cancel()
	wg.Wait()

	rlog.NoticeMsg("poll exiting")

	return nil
}
