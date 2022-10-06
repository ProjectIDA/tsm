package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	rlog "tsm/log"

	g "github.com/gosnmp/gosnmp"
)

const (
	maxCycleTime int = 99999
	snmpVersion  int = 2
)

// snmpScan holds query results with timestamp
type snmpScan struct {
	TS   time.Time
	Data map[string]string
}

// copy returns a pointer to a copy of the TPDin2Scan struct
func (scan *snmpScan) copy() *snmpScan {
	newdata := make(map[string]string)
	for key, val := range scan.Data {
		newdata[key] = val
	}

	newscan := snmpScan{
		scan.TS,
		newdata,
	}

	return &newscan
}

// snmpService struct object
type snmpService struct {
	host             string
	port             uint64
	ready            bool
	internalInterval time.Duration
	SNMPParams       *g.GoSNMP
	ctx              *context.Context
	mutex            sync.Mutex
	// SampleInterval   time.Duration
	CurrentScan *snmpScan
}

// NewSnmpService constructor
func NewSnmpService() *snmpService {
	// func NewSnmpService() cmd.SNMPService {

	tsdev := snmpService{}
	tsdev.ready = false
	return &tsdev

}

// initialize TPDin2 object
func (tsdev *snmpService) initialize(host, port string) error {

	// var host, portstr string
	var portInt uint64
	var e error

	if portInt, e = strconv.ParseUint(port, 10, 16); e != nil {
		return e
	}

	tsdev.host = host
	tsdev.port = portInt
	tsdev.ready = false
	tsdev.SNMPParams = nil

	rlog.DebugMsg("debug: tp.host:             %s", tsdev.host)
	rlog.DebugMsg("debug: tp.port:             %d", tsdev.port)
	rlog.DebugMsg("debug: tp.internalInterval: %s", tsdev.internalInterval)

	return nil
}

// Connect via SNMP to device
func (tsdev *snmpService) Connect(community string) error {

	if !tsdev.ready {

		snmpParams := &g.GoSNMP{
			Target:    tsdev.host,
			Port:      uint16(tsdev.port),
			Transport: "udp4",
			Community: community,
			Version:   g.Version2c,
			Retries:   0,
			Timeout:   time.Duration(2) * time.Second,
		}

		if err := snmpParams.Connect(); err != nil {
			tsdev.SNMPParams = nil
			return err
		}

		tsdev.SNMPParams = snmpParams
		tsdev.ready = true
	} else {
		return errors.New("error: already connected")
	}

	return nil

}

// QueryOids to get values for all device oids
func (tsdev *snmpService) QueryOids(oids *[]string) (time.Time, map[string]string, error) {

	snmpVals, err := tsdev.SNMPParams.Get(*oids)
	if err != nil {
		return time.Now(), nil, err
	}

	results := make(map[string]string)
	for i, variable := range snmpVals.Variables {

		// the Value of each variable returned by Get() implements
		// interface{}. You could do a type switch...
		switch variable.Type {
		case g.OctetString:
			// fmt.Printf("string: %s\n", string(variable.Value.([]byte)))
			results[(*oids)[i]] = string(variable.Value.([]byte))
		default:
			// ... or often you're just interested in numeric values.
			// ToBigInt() will return the Value as a BigInt, for plugging
			// into your calculations.
			// fmt.Printf("number: %d\n", g.ToBigInt(variable.Value))
			results[(*oids)[i]] = g.ToBigInt(variable.Value).String()
		}
	}

	ts := time.Now().UTC()

	return ts, results, nil
}

// queryDeviceVars queries device for TPDin2 OID values
func (tsdev *snmpService) queryDeviceVars(oids *[]string) error {

	ts, results, err := tsdev.QueryOids(oids)
	if err != nil {
		return err
	}
	tsdev.saveScan(ts, &results)

	return nil
}

// func (tp *TPDin2Device) saveScan(scan *TPDin2Scan) {
func (tsdev *snmpService) saveScan(ts time.Time, results *map[string]string) {

	tsdev.mutex.Lock()
	tsdev.CurrentScan = &snmpScan{ts, *results}
	tsdev.mutex.Unlock()
}

// GetScan retuns a copy of the most recent TPDin2Scan struct
func (tsdev *snmpService) GetScan() (time.Time, *map[string]string, error) {

	if tsdev.CurrentScan == nil {
		return time.Now().UTC(), nil, errors.New("scan unavailable")
	}

	tsdev.mutex.Lock()
	scan := tsdev.CurrentScan.copy()
	tsdev.CurrentScan = nil
	tsdev.mutex.Unlock()

	return scan.TS, &(scan.Data), nil
}

// PollStart start polling the connected device
func (tsdev *snmpService) PollStart(
	ctx context.Context,
	wg *sync.WaitGroup,
	pollOids *[]string,
	sampleInterval time.Duration) error {

	tsdev.internalInterval = sampleInterval / 3.0

	wg.Add(1)
	defer wg.Done()

	if !tsdev.ready {
		rlog.WarningMsg("TSEMC1Device is not connected to host: %s", tsdev.host)
		return fmt.Errorf("TSEMC1Device is not connected to host: %s", tsdev.host)
	}

	// kick off internval polling loop
	go func(ctx context.Context, wg *sync.WaitGroup) {
		wg.Add(1)
		defer wg.Done()

		trigtime := time.Now()

		for {
			trigtime = trigtime.Add(tsdev.internalInterval)

			select {
			case <-time.After(time.Until(trigtime)):
				err := tsdev.queryDeviceVars(pollOids)
				if err != nil {
					rlog.ErrMsg(err.Error())
					continue
				}
			case <-ctx.Done():
				rlog.DebugMsg("debug: context.Done message received, shutting down internal polling loop")
				return
			}
		}

	}(ctx, wg)

	return nil
}

// func (tsdev *TSEMC1Device) InitAndConnect(host, port, snmpCommunity string) (string, error) {
func (tsdev *snmpService) InitAndConnect(host, port, snmpCommunity string) error {

	// var modelGroup string

	err := tsdev.initialize(host, port)
	if err != nil {
		rlog.ErrMsg("unknown error initializing structures for %s:%s, quitting", host, port)
	}

	err = tsdev.Connect(snmpCommunity)
	if err != nil {
		rlog.CritMsg("could not connect to %s:%s, quitting", host, port)
	}

	return err
}

func (tsdev *snmpService) Close() {
	tsdev.SNMPParams.Conn.Close()
}
