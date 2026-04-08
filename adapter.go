/*
* Copyright 2025-2026 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package ecoflow2db

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"os"
	"sort"
	"time"

	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

const DefaultIntermediateSize = 15

const SELECT_GET_ALL_PARAMETER = `
with battery as (
select
	to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI') as timest,
	row_number() over (partition by to_char(dq.eco_timestamp at TIME zone 'GMT' , 'YYYYMMDD HH24MI')) as rn,
	eco_bms_bmsstatus_actsoc as batfill
from
	{{ .EcoflowTable }} dq
where
	dq.eco_serial_number = upper('{{ .BatteryConverterSerialNumber }}')
	and eco_timestamp at TIME zone 'GMT' >= NOW() - '30 minute'::interval
order by
	dq.eco_timestamp desc
),
adapter as (
select
	dq.eco_timestamp at TIME zone 'GMT',
	to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI') as timest,
	row_number() over (partition by to_char(dq.eco_timestamp at TIME zone 'GMT', 'YYYYMMDD HH24MI')) as rn,
		(eco_20_1_pv1inputwatts + eco_20_1_pv2inputwatts)/ 10 as "solargen" ,
	abs(least( eco_20_1_batinputwatts::float / 10, 0)) as "batinput",
	abs(greatest( eco_20_1_batinputwatts::float / 10, 0)) as "batout",
	eco_20_1_genewatt / 10 as "housein",
	abs(eco_20_1_gridconswatts::float / 10) as "gridwatts",
	eco_20_1_invdemandwatts / 10 as "requested",
	eco_20_1_bmsreqchgamp as "batreqfill"
from
	{{ .EcoflowTable }} dq
where
	dq.eco_serial_number = upper('{{ .ConverterSerialNumber }}')
		and dq.eco_timestamp at TIME zone 'GMT' >= NOW() - '30 minute'::interval
	order by
		dq.eco_timestamp desc
)
select
		h.inserted_on ,
	b.batfill,
	h.powercurr,
	h.powerout,
	a.solargen,
	a.batinput ,
	a.batout ,
	a.housein ,
	a.gridwatts,
	a.requested ,
	a.batreqfill
from
	battery b
inner join {{ .EnergyTable }} h on
	b.timest = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
inner join adapter a on
	a.timest = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
where
	b.rn = 1
	and a.rn = 1
order by
	b.timest desc,
	h.inserted_on desc
`

type parameter struct {
	timestamp  time.Time
	solargen   int64
	batinput   float64
	batout     float64
	housein    int64
	gridwatts  float64
	requested  int64
	batreqfill int64
	powercurr  int32
	powerout   int32
	batfill    int64
}

func (p *parameter) toString() string {
	return fmt.Sprintf("%15v %10v %10v %10v %10v %10v %10v %10v %10v %10v %10v",
		p.timestamp.Format(shortLayout),
		p.solargen,
		p.batinput,
		p.batout,
		p.housein,
		p.gridwatts,
		p.requested,
		p.batreqfill,
		p.powercurr,
		p.powerout,
		p.batfill)
}

func StartFlow(test bool) {
	counter := 0
	if adapter.DefaultConfig.IntermediateSize == 0 {
		adapter.DefaultConfig.IntermediateSize = DefaultIntermediateSize
	}
	services.ServerMessage("Start Ecoflow flow analyze loop with %d seconds wait time", FlowLoopSeconds)
	for {
		cFlow, err := ReadCurrentFlow()
		if err != nil {
			services.ServerMessage("Ecoflow analyze error: %v", err)
		} else {
			log.Log.Debugf("Ecoflow analyze called")
			AnalyzeEnergyHistory(cFlow, test)
		}
		counter++
		select {
		case <-httpDone:
			services.ServerMessage("Ecoflow analyze loop is stopped")
			return
		case <-time.After(time.Second * time.Duration(FlowLoopSeconds)):
		}
	}
}

func ReadCurrentFlow() ([]*parameter, error) {
	log.Log.Debugf("Read current flow")
	tn := adapter.DatabaseConfig.Table
	tnHome := adapter.DatabaseConfig.EnergyTable
	tmpl, err := template.New("sql").Parse(SELECT_GET_ALL_PARAMETER)
	if err != nil {
		panic(err)
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, struct {
		EcoflowTable                 string
		EnergyTable                  string
		ConverterSerialNumber        string
		BatteryConverterSerialNumber string
	}{EcoflowTable: tn, EnergyTable: tnHome,
		ConverterSerialNumber:        os.ExpandEnv(adapter.EcoflowConfig.MicroConverter[0]),
		BatteryConverterSerialNumber: os.ExpandEnv(adapter.EcoflowConfig.Battery[0])})
	if err != nil {
		panic(err)
	}
	readid := connnectDatabase()
	headerOutput := true
	lastLimitEntries := make([]*parameter, 0)
	fieldMap := make(map[string]int)
	err = readBatch(readid, tn, buffer.String(), func(search *common.Query, result *common.Result) error {
		if headerOutput {
			log.Log.Debugf("LEN: %d->%d", len(fieldMap), len(result.Fields))
			header := ""
			for i, field := range result.Fields {
				switch result.Rows[i].(type) {
				case time.Time:
					header += fmt.Sprintf("%15s ", field)
				default:
					header += fmt.Sprintf("%10s ", field)
				}
				fieldMap[field] = i
			}
			log.Log.Debugf("Header: %s", header)
			headerOutput = false
		}
		p := &parameter{}
		p.timestamp = result.Rows[fieldMap["timezone"]].(time.Time)
		p.solargen = result.Rows[fieldMap["solargen"]].(int64)
		p.batinput = result.Rows[fieldMap["batinput"]].(float64)
		p.batout = result.Rows[fieldMap["batout"]].(float64)
		p.housein = result.Rows[fieldMap["housein"]].(int64)
		p.gridwatts = result.Rows[fieldMap["gridwatts"]].(float64)
		p.requested = result.Rows[fieldMap["requested"]].(int64)
		p.batreqfill = result.Rows[fieldMap["batreqfill"]].(int64)
		p.powercurr = result.Rows[fieldMap["powercurr"]].(int32)
		p.powerout = result.Rows[fieldMap["powerout"]].(int32)
		p.batfill = result.Rows[fieldMap["batfill"]].(int64)
		lastLimitEntries = append(lastLimitEntries, p)
		log.Log.Debugf("Record: %s", p.toString())
		return nil
	})
	if err != nil {
		log.Log.Errorf("Read flow error: %v (sql = %s)", err, buffer.String())
		return nil, err
	}
	if len(lastLimitEntries) == 0 {
		log.Log.Infof("No flow entries found, sql call: %s", buffer.String())
	}
	log.Log.Infof("Read %d flow entries", len(lastLimitEntries))
	return lastLimitEntries, nil
}

func AnalyzeEnergyHistory(lastLimitEntries []*parameter, test bool) {
	if len(lastLimitEntries) == 0 {
		log.Log.Infof("No last limit entries found")
		return
	}
	converter := os.ExpandEnv(adapter.EcoflowConfig.MicroConverter[0])
	services.ServerMessage("Requested:  %d", lastLimitEntries[0].requested)
	services.ServerMessage("Powerout:   %d", lastLimitEntries[0].powerout)
	lastRequested := lastLimitEntries[0].requested
	newRequested := lastRequested
	powerout := lastLimitEntries[0].powerout
	if powerout > 0 {
		log.Log.Infof("PowerOut  found:  %d", powerout)
		log.Log.Infof("PowerCurr found:  %d", lastLimitEntries[0].powercurr)
		reduceToRequest := float64(lastLimitEntries[0].requested) - float64(powerout) -
			float64(adapter.DefaultConfig.IntermediateSize)
		log.Log.Infof("Reduce to:  %f last=%d", reduceToRequest, lastRequested)
		if reduceToRequest > float64(adapter.DefaultConfig.BaseRequest) {
			newRequested = int64(reduceToRequest)
		} else {
			newRequested = adapter.DefaultConfig.BaseRequest
		}
		if adapter.DefaultConfig.DynamicRequest && !test && newRequested != lastRequested {
			client.SetEnvironmentPowerConsumption(converter, float64(newRequested))
		}
		return
	}
	if lastLimitEntries[0].housein == 0 {
		if lastLimitEntries[0].requested > adapter.DefaultConfig.BaseRequest {
			log.Log.Debugf("Set base request: %d", adapter.DefaultConfig.BaseRequest)
			newRequested = adapter.DefaultConfig.BaseRequest
		}
	}

	log.Log.Debugf("Housein:    %d", lastLimitEntries[0].housein)
	sort.SliceStable(lastLimitEntries, func(i, j int) bool {
		return lastLimitEntries[i].powercurr < lastLimitEntries[j].powercurr
	})
	for _, l := range lastLimitEntries {
		log.Log.Infof("LAST LIMIT:" + l.toString())
		l.powercurr = int32(math.Max(float64(l.powercurr), float64(adapter.DefaultConfig.UpperBatLimit)))
	}
	l := len(lastLimitEntries)
	median := float64(0)
	if l > 0 {
		if l%2 == 0 {
			median = float64(lastLimitEntries[l/2-1].powercurr+lastLimitEntries[l/2].powercurr) / 2
		} else {
			median = float64(lastLimitEntries[l/2].powercurr)
		}
	}

	newRequested = lastLimitEntries[0].housein + int64(lastLimitEntries[0].powercurr) - lastRequested
	log.Log.Infof("Base:     %d", adapter.DefaultConfig.BaseRequest)
	log.Log.Infof("Last:     %d", lastRequested)
	log.Log.Infof("Minmum:   %d", lastLimitEntries[0].powercurr)
	services.ServerMessage("BatOut:   %f", lastLimitEntries[0].batout)
	log.Log.Infof("BatIn:   %f", lastLimitEntries[0].batinput)
	log.Log.Infof("Maxima:   %d", lastLimitEntries[len(lastLimitEntries)-1].powercurr)
	services.ServerMessage("Median:   %f", median)
	log.Log.Infof("Needed:   %f", lastLimitEntries[0].powercurr+int32(lastRequested))
	if log.IsDebugLevel() {
		log.Log.Debugf("Old:      %d", lastRequested)
		log.Log.Debugf("Power:    %d", lastLimitEntries[0].housein+int64(lastLimitEntries[0].powercurr))
		log.Log.Debugf("NewDiff:  %d", newRequested)
		log.Log.Debugf("MedPower: %d", lastRequested+int64(median))
		log.Log.Debugf("Max:      %d", adapter.DefaultConfig.UpperBatLimit)
	}
	newRequested = lastRequested + newRequested + adapter.DefaultConfig.IntermediateSize
	if newRequested < adapter.DefaultConfig.BaseRequest {
		newRequested = adapter.DefaultConfig.BaseRequest
	}
	if newRequested > adapter.DefaultConfig.UpperBatLimit {
		newRequested = adapter.DefaultConfig.UpperBatLimit
	}
	services.ServerMessage("New power consumption:      %d > %d", newRequested, adapter.DefaultConfig.BaseRequest)
	if newRequested > adapter.DefaultConfig.BaseRequest {
		log.Log.Infof("Set request to converter %s: %d", converter, newRequested)
		if adapter.DefaultConfig.DynamicRequest && !test && newRequested != lastRequested {
			client.SetEnvironmentPowerConsumption(converter, float64(newRequested))
		} else {
			log.Log.Infof("Test = %v, New requested is same as last requested, no change: %d", test, newRequested)
		}
	}
}
