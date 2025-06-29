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
	"fmt"
	"os"
	"time"

	"github.com/tknie/flynn/common"
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
	h.powercurr,
	h.powerout
from
	device_hw51zeh49g9q1664_quota dhzgqq,
	home h
where
	to_char(to_timestamp(dhzgqq.eco_20_1_utctime) , 'YYYYMMDD HH24MI') = to_char(h.inserted_on , 'YYYYMMDD HH24MI')
order by
	h.inserted_on desc,
	dhzgqq.eco_timestamp desc
limit 4
`

const SELECT_GET_PARAMETER = `
 select to_timestamp(dhzgqq.eco_20_1_utctime  ) as timestamp,(dhzgqq.eco_20_1_pv1inputwatts+dhzgqq.eco_20_1_pv2inputwatts)/10 as "solargen" , abs(least( dhzgqq.eco_20_1_batinputwatts::float/10,0)) as "batinput", abs(greatest( dhzgqq.eco_20_1_batinputwatts::float/10,0))  as "batout", dhzgqq.eco_20_1_genewatt / 10 as "housein", abs(dhzgqq.eco_20_1_gridconswatts::float / 10) as "gridwatts", eco_20_1_invdemandwatts / 10 as "requested" from device_hw51zeh49g9q1664_quota dhzgqq ORDER BY eco_timestamp DESC limit 1
`

const SELECT_GET_ENERGY = `
select inserted_on AT TIME ZONE 'UTC' AT TIME ZONE 'GMT'  AS "time", powercurr,powerout   from home order by inserted_on  desc limit 1;
`

type parameter struct {
	timestamp time.Time
	solargen  int64
	batinput  float64
	batout    float64
	housein   int64
	gridwatts float64
	requested int64
	powercurr int64
	powerout  int64
}

func ReadCurrentFlow() {

	sn := os.Getenv("ECOFLOW_DEVICE_SN")
	tn := "device_" + sn + "_quota"
	readid := connnectDatabase()
	readBatch(readid, tn, SELECT_GET_PARAMETER, func(search *common.Query, result *common.Result) error {
		for i, field := range result.Fields {
			fmt.Printf("%10s = %v\n", field, result.Rows[i])
		}
		return nil
	})
	readBatch(readid, tn, SELECT_GET_ENERGY, func(search *common.Query, result *common.Result) error {
		for i, field := range result.Fields {
			fmt.Printf("%10s = %v\n", field, result.Rows[i])
		}
		return nil
	})
	headerOutput := false
	lastLimitEntries := make([]*parameter, 0)
	readBatch(readid, tn, SELECT_GET_ALL_PARAMETER, func(search *common.Query, result *common.Result) error {
		if !headerOutput {
			for i, field := range result.Fields {
				switch result.Rows[i].(type) {
				case time.Time:
					fmt.Printf("%15s ", field)
				default:
					fmt.Printf("%10s ", field)
				}
			}
			fmt.Printf("\n")
			headerOutput = true
		}
		p := &parameter{result.Rows[0].(time.Time), result.Rows[1].(int64), result.Rows[2].(float64),
			result.Rows[3].(float64), result.Rows[4].(int64),
			result.Rows[5].(float64), result.Rows[6].(int64),
			result.Rows[6].(int64), result.Rows[6].(int64)}
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
	fmt.Println(lastLimitEntries[0].timestamp.Sub(lastLimitEntries[1].timestamp), lastLimitEntries[0].solargen)
}
