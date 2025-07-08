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
	services.ServerMessage("Start ecoflow2db application v%s (build at %v)", ecoflow2db.Version, ecoflow2db.BuildDate)
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

	flag.IntVar(&ecoflow2db.LoopSeconds, "t", ecoflow2db.LoopSeconds, "The seconds wating between REST API queries")
	flag.IntVar(&statSecs, "s", int(ecoflow2db.StatLoopMinutes), "The minutes waiting between statistics output")
	flag.BoolVar(&ecoflow2db.MqttDisable, "m", false, "Disable MQTT listener")
	flag.BoolVar(&readFlow, "r", false, "Read current flow parameter")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.StringVar(&flowControlFile, "f", "", "Load YAML control file")
	flag.Float64Var(&powervalue, "p", 0, "Set new power value for the power powerstream")

	flag.Parse()

	if powervalue > 0 {
		services.ServerMessage("Set new power value for powerstream to %f", powervalue)
		ecoflow2db.SetEnvironmentPowerConsumption(powervalue)
		return
	}

	ecoflow2db.InitFlow(flowControlFile)
	ecoflow2db.StatLoopMinutes = time.Duration(statSecs)

	ecoflow2db.InitDatabase()
	if readFlow {
		_, err := ecoflow2db.ReadCurrentFlow()
		if err != nil {
			services.ServerMessage("Error calling flow: %v", err)
		}
		return
	}
	services.ServerMessage("Loop in API each %d seconds", ecoflow2db.LoopSeconds)

	ecoflow2db.InitEcoflow()

}
