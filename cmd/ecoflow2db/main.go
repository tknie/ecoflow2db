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

package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/tknie/ecoflow2db"
	"github.com/tknie/services"
)

func init() {
	services.ServerMessage("Start ecoflow2db application %s (build at %v)", ecoflow2db.Version, ecoflow2db.BuildDate)
	ecoflow2db.StartLog("ecoflow2db.log")
}

func main() {
	create := false
	ecoflow2db.LoopSeconds = ecoflow2db.DefaultSeconds
	seconds := os.Getenv("ECOFLOW2DB_WAIT_SECONDS")
	if seconds != "" {
		sec, err := strconv.Atoi(seconds)
		if err != nil {
			services.ServerMessage("Invalid wait seconds given, use default %d seconds", ecoflow2db.LoopSeconds)
		} else {
			ecoflow2db.LoopSeconds = sec
		}
	}
	statSecs := 0
	powervalue := float64(0)
	readFlow := false
	flowControlFile := ""
	test := false
	flow := false
	caracon := false
	serialNumber := ""

	flag.IntVar(&ecoflow2db.LoopSeconds, "t", ecoflow2db.LoopSeconds, "The seconds wating between REST API queries")
	flag.IntVar(&statSecs, "s", int(ecoflow2db.StatLoopMinutes), "The minutes waiting between statistics output")
	flag.BoolVar(&ecoflow2db.MqttDisable, "m", false, "Disable MQTT listener")
	flag.BoolVar(&readFlow, "r", false, "Read current flow parameter")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.BoolVar(&flow, "a", false, "Start energy analyze")
	flag.BoolVar(&caracon, "c", false, "Power car AC on")
	flag.BoolVar(&test, "T", false, "Do tests and output only")
	flag.StringVar(&serialNumber, "S", "", "Use serial number")
	flag.StringVar(&flowControlFile, "f", "", "Load YAML control file")
	flag.Float64Var(&powervalue, "p", 0, "Set new power value for the power powerstream")

	flag.Parse()

	if flowControlFile != "" {
		ecoflow2db.LoadConfig(flowControlFile)
	}

	if powervalue > 0 {
		services.ServerMessage("Set new power value for powerstream to %f", powervalue)
		ecoflow2db.SetEnvironmentPowerConsumption(powervalue)
		return
	}

	if caracon {
		services.ServerMessage("Set AC car power on")
		ecoflow2db.SetCarACOn(serialNumber, true)
		return
	}

	ecoflow2db.StatLoopMinutes = time.Duration(statSecs)

	ecoflow2db.InitDatabase()
	if readFlow {
		l, err := ecoflow2db.ReadCurrentFlow()
		if err != nil {
			services.ServerMessage("Error calling flow: %v", err)
		}
		ecoflow2db.AnalyzeEnergyHistory(l, true)
		return
	}
	if flow {
		ecoflow2db.StartFlow(test)
		return
	}
	services.ServerMessage("Loop in API each %d seconds", ecoflow2db.LoopSeconds)

	ecoflow2db.InitEcoflow()

}
