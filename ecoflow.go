package ecoflow2db

import (
	"context"
	"os"

	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/log"
)

func InitEcoflow() {
	user := os.Getenv("ECOFLOW_USER")
	password := os.Getenv("ECOFLOW_PASSWORD")

	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)
	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		log.Log.Fatalf("Error getting device list: %v", err)
	}
	devices = list

	triggerParameterStore(client)

	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	InitMqtt(user, password)
	<-done
}
