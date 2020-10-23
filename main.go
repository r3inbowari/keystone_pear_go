package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
	"golang.org/x/net/context"
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

	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	insertResult, err := db.InsertOne(context.TODO(), result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted BME Data ->", insertResult.InsertedID)
}

func apdsCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	insertResult, err := apdsConn.InsertOne(context.TODO(), result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted APDS Data ->", insertResult.InsertedID)
}

func lopgonCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	insertResult, err := logonConn.InsertOne(context.TODO(), result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted Logon Data ->", insertResult.InsertedID, "ID ->", result.ID)
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
	var result MessageBody
	vars := mux.Vars(r)

	println(vars["id"])

	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": "",
				"max": bson.M{"$max": "$price"},
			},
		},
	}
	result := []bson.M{}
	err = db.C("product").Pipe(pipeline).All(&result)


	if err != nil {
		_, _ = w.Write([]byte("{\"code\":233,\"msg\":\"not found data\"}"))
	}
	fmt.Printf("[INFO] Get BME value: %+v\n", result)
	ret, _ := json.Marshal(result)
	_, _ = w.Write(ret)
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
