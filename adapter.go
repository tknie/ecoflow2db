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
	"io"
	"os"
	"time"

	"github.com/stretchr/testify/assert/yaml"
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
	dhzgqq.eco_20_1_bmsreqchgamp  as "batfill",
	h.powercurr,
	h.powerout,
	drzsq.eco_bms_bmsstatus_actsoc as batfill
from
	device_hw51zeh49g9q1664_quota dhzgqq,
	home h, device_r331zeb5sgbu0300_quota drzsq 
where
	h.inserted_on >= NOW() - '30 minute'::INTERVAL
	and dhzgqq.eco_timestamp >= NOW() - '30 minute'::INTERVAL
	and drzsq.eco_timestamp >= NOW() - '30 minute'::INTERVAL
	and to_char(to_timestamp(dhzgqq.eco_20_1_utctime) , 'YYYYMMDD HH24MI') = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
	and to_char(drzsq.eco_timestamp , 'YYYYMMDD HH24MI') = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
order by
	h.inserted_on desc,
	dhzgqq.eco_timestamp desc
limit 4
`

const defaultBaseRequest = 170

type parameter struct {
	timestamp time.Time
	solargen  int64
	batinput  float64
	batout    float64
	housein   int64
	gridwatts float64
	requested int64
	powercurr int32
	powerout  int32
	batfill   int64
}

type adapterConfig struct {
	defaultConfig *defaultConfig `yaml:"default"`
}

type defaultConfig struct {
	baseRequest int64 `yaml:"baseWatt"`
}

var adapter = &adapterConfig{defaultConfig: &defaultConfig{baseRequest: defaultBaseRequest}}

var FlowLoopSeconds = DefaultSeconds

// ReadConfig read config file
func readConfig(file string) ([]byte, error) {
	configFile, err := os.Open(file)
	if err != nil {
		log.Log.Debugf("Open file error: %#v", err)
		return nil, fmt.Errorf("open file err of %s: %v", file, err)
	}
	defer configFile.Close()

	fi, _ := configFile.Stat()
	log.Log.Debugf("File size=%d", fi.Size())
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, configFile)
	if err != nil {
		log.Log.Debugf("Read file error: %#v", err)
		return nil, fmt.Errorf("read file err of %s: %v", file, err)
	}
	fmt.Printf("Config loaded %#v\n", adapter)
	fmt.Printf("Config loaded %d\n", adapter.defaultConfig.baseRequest)
	return buffer.Bytes(), nil
}

func InitFlow(file string) {
	if file != "" {
		fileEnvResolved := os.ExpandEnv(file)

		data, err := readConfig(fileEnvResolved)
		if err != nil {
			log.Log.Fatal("Error loading config: %s", file)
		}
		err = yaml.Unmarshal(data, adapter)
		if err != nil {
			log.Log.Fatal("Error unmarshal config: %s", file)
		}
		if adapter.defaultConfig.baseRequest == 0 {
			adapter.defaultConfig.baseRequest = defaultBaseRequest
		}
	}
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
				analyze(cFlow, test)
			}
		}
	}
}

func ReadCurrentFlow() ([]*parameter, error) {

	tn := "device_quota"
	readid := connnectDatabase()
	headerOutput := false
	fieldMap := make(map[string]int)
	lastLimitEntries := make([]*parameter, 0)
	err := readBatch(readid, tn, SELECT_GET_ALL_PARAMETER, func(search *common.Query, result *common.Result) error {
		if !headerOutput {
			for i, field := range result.Fields {
				switch result.Rows[i].(type) {
				case time.Time:
					fmt.Printf("%15s ", field)
				default:
					fmt.Printf("%10s ", field)
				}
				fieldMap[field] = i
			}
			fmt.Printf("\n")
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
		p.powercurr = result.Rows[fieldMap["powercurr"]].(int32)
		p.powerout = result.Rows[fieldMap["powerout"]].(int32)
		p.batfill = result.Rows[fieldMap["batfill"]].(int64)
		// fmt.Printf("parameter %v %#v\n", p.timestamp, p)
		lastLimitEntries = append(lastLimitEntries, p)
		for _, r := range result.Rows {
			switch v := r.(type) {
			case time.Time:
				fmt.Printf("%10s", v.Format(shortLayout))
			default:
				fmt.Printf("%10v ", r)
			}
		}
		fmt.Printf("\n")
		return nil
	})
	if err != nil {
		return nil, err
	}
	return lastLimitEntries, nil
}

func analyze(lastLimitEntries []*parameter, test bool) {
	fmt.Println(lastLimitEntries[0].timestamp.Sub(lastLimitEntries[1].timestamp), lastLimitEntries[0].powerout)
	fmt.Println("Requested:  ", lastLimitEntries[0].requested)
	fmt.Println("Powerout:   ", lastLimitEntries[0].powerout)
	if lastLimitEntries[0].powerout > 0 {
		fmt.Println("Reduce by:  ", lastLimitEntries[0].powerout)
		reduceToRequest := float64(lastLimitEntries[0].requested) - float64(lastLimitEntries[0].powerout) - 10
		fmt.Println("Reduce to:  ", reduceToRequest)
		if reduceToRequest > float64(adapter.defaultConfig.baseRequest) {
			SetEnvironmentPowerConsumption(reduceToRequest)
		}
	}
	if lastLimitEntries[0].housein == 0 {
		if lastLimitEntries[0].requested > adapter.defaultConfig.baseRequest {
			fmt.Printf("Set base request: %d\n", adapter.defaultConfig.baseRequest)
			if !test {
				SetEnvironmentPowerConsumption(float64(adapter.defaultConfig.baseRequest))
			}
		}
	}

	fmt.Println("Housein:    ", lastLimitEntries[0].housein)
}
