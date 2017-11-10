package models

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	OrderList map[string]*Order
)

var (
	database string
	password string
	status   string
)

var username string
var address []string
var isAzure = true

var hosts string

var insightskey = os.Getenv("INSIGHTSKEY")
var mongoURL = os.Getenv("MONGOURL")
var source = os.Getenv("SOURCE")

// Order represents the order json
type Order struct {
	ID                string  `required:"false" description:"CosmoDB ID - will be autogenerated"`
	EmailAddress      string  `required:"true" description:"Email address of the customer"`
	PreferredLanguage string  `required:"false" description:"Preferred Language of the customer"`
	Product           string  `required:"false" description:"Product ordered by the customer"`
	Total             float64 `required:"false" description:"Order total"`
	Source            string  `required:"false" description:"Source channel e.g. App Service, Container instance, K8 cluster etc"`
	Status            string  `required:"true" description:"Order Status"`
}

func init() {
	OrderList = make(map[string]*Order)
}

func AddOrder(order Order) (orderId string) {

	return orderId
}

func ProcessOrderInMongoDB(order Order) (orderId string) {

	//	database = utils.GetEnvVarOrExit("DATABASE")
	//	password = utils.GetEnvVarOrExit("PASSWORD")

	database = "k8orders"

	/* 	// DialInfo holds options for establishing a session with a MongoDB cluster.
	   	dialInfo := &mgo.DialInfo{
	   		Addrs:    []string{fmt.Sprintf("%s.documents.azure.com:10255", database)}, // Get HOST + PORT
	   		Timeout:  60 * time.Second,
	   		Database: database, // It can be anything
	   		Username: database, // Username
	   		Password: password, // PASSWORD
	   		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
	   			return tls.Dial("tcp", addr.String(), &tls.Config{})
	   		},
	   	}

	   	// Create a session which maintains a pool of socket connections
	   	// to our MongoDB.
	   	session, err := mgo.DialWithInfo(dialInfo) */
	session, err := mgo.Dial(mongoURL)

	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		status = "Can't connect to mongo, go error %v\n"
		os.Exit(1)
	}

	defer session.Close()

	// SetSafe changes the session safety mode.
	// If the safe parameter is nil, the session is put in unsafe mode, and writes become fire-and-forget,
	// without error checking. The unsafe mode is faster since operations won't hold on waiting for a confirmation.
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode.
	session.SetSafe(&mgo.Safe{})

	// get collection
	collection := session.DB(database).C("orders")

	// Get Document from collection
	result := Order{}
	log.Println("!!Looking for ", "{", "id:", order.ID, ",", "status:", "Open", "}")
	//err = collection.FindId(bson.ObjectIdHex(order.ID)).One(&result)
	//	err = collection.Find(bson.ObjectIdHex(order.ID)).One(&result)

	//query = fmt.Println('"', order.ID, '"')
	//log.Println(query, order.ID)
	err = collection.Find(bson.M{"id": order.ID, "status": "Open"}).One(&result)

	if err != nil {
		log.Fatal("Error finding record: ", err)
		return
	}

	log.Println("set status: Processed")

	change := bson.M{"$set": bson.M{"status": "Processed"}}
	err = collection.Update(result, change)
	if err != nil {
		log.Fatal("Error updating record: ", err)
		return
	}

	//	Let's write only if we have a key
	if insightskey != "" {
		t := time.Now()
		client := appinsights.NewTelemetryClient(insightskey)
		client.TrackEvent("Process Order " + source + ": " + order.ID)
		client.TrackTrace(t.String())
	}

	// Let's place on the file system
	f, err := os.Create("/orders/" + order.ID + ".json")
	check(err)

	fmt.Fprintf(f, "{", "id:", order.ID, ",", "status:", "Processed", "}")

	// Issue a `Sync` to flush writes to stable storage.
	f.Sync()

	return order.ID
}

func check(e error) {
	if e != nil {
		log.Println("order volume not mounted")
	}
}
