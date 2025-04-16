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
	"os"
	"os/signal"
	"syscall"

	"github.com/tknie/log"
)

func setupGracefulShutdown(done chan bool) {
	// Create a channel to listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown
	go func() {
		<-signalChan
		log.Log.Infof("Received shutdown signal")

		done <- true
		endHttp()
		close(quit)
	}()
}
