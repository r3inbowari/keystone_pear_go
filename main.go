package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"net/http"
)

type Version struct {
	ID    string `json:"id"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
}

var c mqtt.Client

func main() {
	opts := mqtt.NewClientOptions().AddBroker("tcp://r3inbowari.top:1883").SetClientID("sample_go")
	opts.SetUsername("r3inb")
	opts.SetPassword("159463")
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("meshNetwork/from/rootNode/checkupdate", 0, updateCallback); token.Wait() && token.Error() != nil {

	}

	http.HandleFunc("/check_update", HandleVersion)
	log.Fatal(http.ListenAndServe(":2999", nil))
}

type Result struct {
	ID string `json:"id"`
}
func updateCallback(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
	var result Result
	_ = json.Unmarshal(msg.Payload(), &result)

	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
		ID: result.ID,
	}
	ret, _ := json.Marshal(version)
	client.Publish("meshNetwork/to/rootNode/checkupdate", 1, false, ret)
}

func HandleVersion(w http.ResponseWriter, r *http.Request) {
	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
	}
	ret, _ := json.Marshal(version)
	_, _ = w.Write(ret)
}
