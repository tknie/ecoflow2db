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

	flag.IntVar(&ecoflow2db.LoopSeconds, "t", ecoflow2db.LoopSeconds, "The minutes between REST API queries")
	flag.BoolVar(&ecoflow2db.MqttDisable, "m", false, "Disable MQTT listener")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.Parse()

	services.ServerMessage("Loop in API each %d seconds", ecoflow2db.LoopSeconds)
	ecoflow2db.InitDatabase()
	ecoflow2db.InitEcoflow()

}
