/*
* Copyright 2025 Thorsten A. Knieling
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
	"sort"
	"time"

	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

const SELECT_GET_ALL_PARAMETER = `
select
	h.inserted_on at TIME zone 'UTC' at TIME zone 'GMT',
	(dhzgqq.eco_20_1_pv1inputwatts + dhzgqq.eco_20_1_pv2inputwatts)/ 10 as "solargen" ,
	abs(least( dhzgqq.eco_20_1_batinputwatts::float / 10, 0)) as "batinput",
	abs(greatest( dhzgqq.eco_20_1_batinputwatts::float / 10, 0)) as "batout",
	dhzgqq.eco_20_1_genewatt / 10 as "housein",
	abs(dhzgqq.eco_20_1_gridconswatts::float / 10) as "gridwatts",
	dhzgqq.eco_20_1_invdemandwatts / 10 as "requested",
	dhzgqq.eco_20_1_bmsreqchgamp  as "batreqfill",
	h.powercurr,
	h.powerout,
	drzsq.eco_bms_bmsstatus_actsoc as batfill
from
	{{ .EcoflowTable }} dhzgqq,
	{{ .EnergyTable }} h, {{ .EcoflowTable }} drzsq 
where
	dhzgqq.eco_serial_number = upper('hw51zeh49g9q1664') and drzsq.eco_serial_number = upper('r331zeb5sgbu0300')
	and h.inserted_on >= NOW() - '30 minute'::INTERVAL
	and dhzgqq.eco_timestamp >= NOW() - '30 minute'::INTERVAL
	and drzsq.eco_timestamp >= NOW() - '30 minute'::INTERVAL
	and to_char(to_timestamp(dhzgqq.eco_20_1_utctime) , 'YYYYMMDD HH24MI') = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
	and to_char(drzsq.eco_timestamp , 'YYYYMMDD HH24MI') = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
order by
	h.inserted_on desc,
	dhzgqq.eco_timestamp desc
limit 10
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
	for {
		counter++
		select {
		case <-httpDone:
			services.ServerMessage("Ecoflow analyze loop is stopped")
			return
		case <-time.After(time.Second * time.Duration(FlowLoopSeconds)):
			cFlow, err := ReadCurrentFlow()
			if err != nil {
				services.ServerMessage("Ecoflow analyze error: %v", err)
			} else {
				services.ServerMessage("Ecoflow analyze called")
				AnalyzeEnergyHistory(cFlow, test)
			}
		}
	}
}

func ReadCurrentFlow() ([]*parameter, error) {

	tn := adapter.DatabaseConfig.EcoflowTable
	tnHome := adapter.DatabaseConfig.EnergyTable
	tmpl, err := template.New("sql").Parse(SELECT_GET_ALL_PARAMETER)
	if err != nil {
		panic(err)
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, struct {
		EcoflowTable string
		EnergyTable  string
	}{EcoflowTable: tn, EnergyTable: tnHome})
	if err != nil {
		panic(err)
	}
	readid := connnectDatabase()
	headerOutput := false
	fieldMap := make(map[string]int)
	lastLimitEntries := make([]*parameter, 0)
	err = readBatch(readid, tn, buffer.String(), func(search *common.Query, result *common.Result) error {
		if !headerOutput {
			for i, field := range result.Fields {
				switch result.Rows[i].(type) {
				case time.Time:
					header += fmt.Sprintf("%15s ", field)
				default:
					header += fmt.Sprintf("%10s ", field)
				}
				fieldMap[field] = i
			}
			fmt.Println(header)
			headerOutput = true
		}
		// fmt.Printf("MMM %#v\n", fieldMap)
		// fmt.Printf("MMM %#v\n", result.Fields)
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
		// fmt.Printf("parameter %v %#v\n", p.timestamp, p)
		lastLimitEntries = append(lastLimitEntries, p)
		fmt.Println(p.toString())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return lastLimitEntries, nil
}

func AnalyzeEnergyHistory(lastLimitEntries []*parameter, test bool) {
	if len(lastLimitEntries) == 0 {
		return
	}
	log.Log.Debugf("Requested:  %d", lastLimitEntries[0].requested)
	log.Log.Debugf("Powerout:   %d", lastLimitEntries[0].powerout)
	lastRequested := lastLimitEntries[0].requested
	powerout := lastLimitEntries[0].powerout
	if powerout > 0 {
		log.Log.Debugf("Reduce by:  %d", lastLimitEntries[0].powerout)
		reduceToRequest := float64(lastLimitEntries[0].requested) - float64(lastLimitEntries[0].powerout) - 10
		log.Log.Debugf("Reduce to:  %d", reduceToRequest)
		if reduceToRequest > float64(adapter.DefaultConfig.BaseRequest) {
			SetEnvironmentPowerConsumption(reduceToRequest)
			lastRequested = int64(reduceToRequest)
		} else {
			SetEnvironmentPowerConsumption(float64(adapter.DefaultConfig.BaseRequest))
			lastRequested = adapter.DefaultConfig.BaseRequest
		}
	}
	newRequested := lastLimitEntries[0].requested
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
	fmt.Println(header)
	for _, l := range lastLimitEntries {
		fmt.Println(l.toString())
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

	log.Log.Debugf("Old:      %d", newRequested)
	log.Log.Debugf("Median:   %d", median)
	log.Log.Debugf("Minmum:   %d", lastLimitEntries[0].powercurr)
	log.Log.Debugf("Maxima:   %d", lastLimitEntries[len(lastLimitEntries)-1].powercurr)
	log.Log.Debugf("Power:    %d", lastLimitEntries[0].housein+int64(lastLimitEntries[0].powercurr))
	newRequested = lastLimitEntries[0].housein + int64(lastLimitEntries[0].powercurr) - lastRequested
	log.Log.Debugf("Last:     %d", lastRequested)
	fmt.Println("NewDiff:  ", newRequested)
	fmt.Println("MedPower: ", lastRequested+int64(median))
	newRequested = lastRequested + newRequested
	if newRequested < adapter.DefaultConfig.BaseRequest {
		newRequested = adapter.DefaultConfig.BaseRequest
	}
	if newRequested > adapter.DefaultConfig.MaxRequest {
		newRequested = adapter.DefaultConfig.MaxRequest
	}
	fmt.Println("New:      ", newRequested)
	if newRequested > adapter.DefaultConfig.BaseRequest {
		fmt.Printf("Set request: %d\n", newRequested)
		if !test {
			SetEnvironmentPowerConsumption(float64(newRequested))
		}
	}

}
