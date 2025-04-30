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
	sync "sync"
	"time"

	"github.com/tknie/services"
)

type statMqtt struct {
	mu          sync.Mutex
	mqttCounter uint64
	httpCounter uint64
}

var StatLoopMinutes = time.Duration(5)

var mapStatMqtt = make(map[string]*statMqtt)

type statDatabase struct {
	counter uint64
}

var mapStatDatabase = make(map[string]*statDatabase)

func getStatEntry(serialNumber string) *statMqtt {
	if s, ok := mapStatMqtt[serialNumber]; ok {
		return s
	} else {
		stat := &statMqtt{}
		mapStatMqtt[serialNumber] = stat
		return stat
	}

}

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
				buffer.WriteString("Statistics:\n")
				for k, v := range mapStatMqtt {
					buffer.WriteString(fmt.Sprintf("  %s got http=%03d mqtt=%03d messages\n", k, v.httpCounter, v.mqttCounter))
				}
				for k, v := range mapStatDatabase {
					buffer.WriteString(fmt.Sprintf("  %s inserted %03d records\n", k, v.counter))
				}
				services.ServerMessage(buffer.String())
			case <-quit:
				ticker.Stop()
				services.ServerMessage("Statistics are stopped")
				return
			}
		}
	}()

}
