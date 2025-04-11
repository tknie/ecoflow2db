package ecoflow2db

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/tknie/log"
	"google.golang.org/protobuf/proto"
)

func displayPayload(sn string, payload []byte) bool {
	fmt.Printf("================================================>===============\n")
	log.Log.Debugf("Base64: %s", base64.RawStdEncoding.EncodeToString(payload))
	log.Log.Debugf("Payload %s", FormatByteBuffer("MQTT Body", payload))

	platform := &SendHeaderMsg{}
	err := proto.Unmarshal(payload, platform)
	if err != nil {
		log.Log.Errorf("Unable to parse message message %v: %v", payload, err)
	} else {
		switch platform.Msg.GetCmdId() {
		case 1:

			ih := &InverterHeartbeat{}
			err := proto.Unmarshal(platform.Msg.Pdata, ih)
			if err != nil {
				log.Log.Errorf("Unable to parse pdata message: %v", err)
			} else {
				log.Log.Debugf("-> InverterHearbeat %s\n", ih)
				msgChan <- &storeElement{object: ih, sn: sn}

				log.Log.Debugf("DynamicWatts   %v", ih.GetDynamicWatts())
				log.Log.Debugf("LowerLimit     %v", ih.GetLowerLimit())
				log.Log.Debugf("PermanentWatts %v", ih.GetPermanentWatts())
				log.Log.Debugf("UpperLimit     %v", ih.GetUpperLimit())
				log.Log.Debugf("InstallCountry %v", ih.GetInstallCountry())
				log.Log.Debugf("InvOnOff       %v", ih.GetInvOnOff())
				log.Log.Debugf("Pv10pVolt      %v", ih.GetPv1OpVolt())
				log.Log.Debugf("Pv1InputVolt   %v", ih.GetPv1InputVolt())
				log.Log.Debugf("Pv1InputWatts  %v", ih.GetPv1InputWatts())
				log.Log.Debugf("Pv20pVolt      %v", ih.GetPv2OpVolt())
				log.Log.Debugf("Pv2InputVolt   %v", ih.GetPv2InputVolt())
				log.Log.Debugf("Pv2InputWatts  %v", ih.GetPv2InputWatts())
				log.Log.Debugf("Timestamp      %v", ih.GetTimestamp())
				log.Log.Debugf("Time           %v", time.Unix(int64(ih.GetTimestamp()), 0))
			}
		case 32:
			pp := &PowerPack{}
			err := proto.Unmarshal(platform.Msg.Pdata, pp)
			if err != nil {
				log.Log.Errorf("Unable to parse pdata message: %v", err)
			} else {
				for _, p := range pp.SysPowerStream {
					fmt.Println(p.BatteryPower)
				}
			}
		default:
			displayHeader(platform.Msg)
			fmt.Println("Unknown Cmd ID", platform.Msg.GetCmdId())
			fmt.Printf("received message: %s\n", FormatByteBuffer("MSG Payload", platform.Msg.Pdata))
			fmt.Printf("Base64: %s\n", base64.RawStdEncoding.EncodeToString(payload))
			return false
		}
	}
	return true
}
