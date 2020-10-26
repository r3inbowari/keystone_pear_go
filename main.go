package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
	"net/http"
)

type Version struct {
	ID    string `json:"id"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
}

var client mqtt.Client

func GetMongoDB() (*mgo.Database, error) {
	host := "mongodb://localhost:27017"
	dbName := "learn_mongodb_golang"
	session, err := mgo.Dial(host)
	if err != nil {
		return nil, err
	}
	db := session.DB(dbName)
	return db, nil
}

var db *mgo.Database

func main() {
	db, _ = GetMongoDB()

	opts := mqtt.NewClientOptions().AddBroker("tcp://r3inbowari.top:1883").SetClientID("sample_go")
	opts.SetUsername("r3inb")
	opts.SetPassword("159463")
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("meshNetwork/from/rootNode/checkupdate", 0, updateCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/bme", 0, bmeCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/apds", 0, apdsCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/logon", 0, lopgonCallback); token.Wait() && token.Error() != nil {
	}

	r := mux.NewRouter()
	r.HandleFunc("/check_update", HandleVersion)
	r.HandleFunc("/bme/{id}", HandleBME)
	log.Fatal(http.ListenAndServe(":2999", r))
}

type Result struct {
	ID string `json:"id"`
}

type MessageBody struct {
	ID        string `json:"id"`
	MID       string `json:"mid"`
	Ts        int    `json:"ts"`
	Operation int    `json:"operation"`
	Data      Data   `json:"data"`
}

type Data struct {
	Type    int       `json:"type"`
	Measure []float64 `json:"measure"`
}

func bmeCallback(client mqtt.Client, msg mqtt.Message) {
fmt.Println(msg.Topic())
}

func apdsCallback(client mqtt.Client, msg mqtt.Message) {
	fmt.Println(msg.Topic())
}

func lopgonCallback(client mqtt.Client, msg mqtt.Message) {
	fmt.Println(msg.Topic())
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
		ID:    result.ID,
	}
	ret, _ := json.Marshal(version)
	client.Publish("meshNetwork/to/rootNode/checkupdate", 1, false, ret)
}

func HandleBME(w http.ResponseWriter, r *http.Request) {

}

func HandleAPDS(w http.ResponseWriter, r *http.Request) {
	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
	}
	ret, _ := json.Marshal(version)
	_, _ = w.Write(ret)
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
