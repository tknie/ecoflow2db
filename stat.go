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
	"time"

	"github.com/tknie/ecoflow"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var StatLoopMinutes = time.Duration(5)

type statDatabase struct {
	counter uint64
}

var mapStatDatabase = make(map[string]*statDatabase)

func getDbStatEntry(tn string) *statDatabase {
	if s, ok := mapStatDatabase[tn]; ok {
		return s
	} else {
		stat := &statDatabase{}
		mapStatDatabase[tn] = stat
		return stat
	}
}

func startStatLoop() {
	ticker := time.NewTicker(StatLoopMinutes * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				var buffer bytes.Buffer
				buffer.WriteString("Statistics at ")
				buffer.WriteString(time.Now().Format(layout))
				buffer.WriteString(":\n")
				buffer.WriteString(ecoflow.StatMqtt())
				for k, v := range mapStatDatabase {
					buffer.WriteString(fmt.Sprintf("  %s inserted %03d records\n", k, v.counter))
				}
				log.Log.Infof(buffer.String())
			case <-quit:
				ticker.Stop()
				services.ServerMessage("Statistics are stopped")
				return
			}
		}
	}()

}
