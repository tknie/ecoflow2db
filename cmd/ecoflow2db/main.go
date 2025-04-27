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

	"github.com/tknie/ecoflow2db"
	"github.com/tknie/services"
)

func init() {
	ecoflow2db.StartLog("ecoflow2db.log")
}

func main() {
	create := false

	flag.IntVar(&ecoflow2db.LoopSeconds, "t", ecoflow2db.DefaultSeconds, "The minutes between REST API queries")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.Parse()

	services.ServerMessage("Start ecoflow2db application v%s (build at %v)", ecoflow2db.Version, ecoflow2db.BuildDate)
	services.ServerMessage("Loop in API each %d seconds", ecoflow2db.LoopSeconds)
	ecoflow2db.InitDatabase()
	ecoflow2db.InitEcoflow()

}
