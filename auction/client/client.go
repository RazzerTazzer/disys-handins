package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	proto "auction/grpc"

	"google.golang.org/grpc"
)

type client struct {
	conn *grpc.ClientConn
	name string
	port int
	timestamp int
}

var (
	port = flag.Int("port", 8080, "port number")
	name = flag.String("name", "Bob", "next port number")
)

func main() {	
	flag.Parse()


	// Create a new client
	client := &client{
		port: *port,
		name: *name,
		timestamp: 0,
	}

	f, err := os.OpenFile("logfile."+ strconv.Itoa(client.port), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// Connect to the server
	client.connect(5557)
	go startHeartbeat(client)

	for {
		var input string
		_, _ = fmt.Scanln(&input)

		if input == "currentBid" {
			askForBid(client)
		}

		if input == "bid" {
			bid(client)
	
		}

		if input == "result" {
			getResult(client)
		}

	}
		
}

func bid(c *client) {
	// Implement logic to place a bid
	log.Println("Enter your bid:")
	var bidValue int
	_, err := fmt.Scanln(&bidValue)
	if err != nil {
		log.Printf("Error reading bid value: %v\n", err)
		return
	}

	// Call the bid function with the entered bid value
	makeBid(c, bidValue)
}

func makeBid(c *client, bidValue int) {
	c.timestamp++
	log.Printf("Placing bid: %d\n", bidValue)

    bidClient := proto.NewAuctionClient(c.conn)
    result, err := bidClient.Bid(context.Background(), &proto.BidMessage{Bid: int64(bidValue), Name: c.name, Timestamp: int64(c.timestamp)})
    if err != nil {
        log.Printf("Error sending vote request to next node: %v\n", err)
    } else {
		if (int(result.Timestamp) > c.timestamp) {
			c.timestamp = int(result.Timestamp)+1
		} 
		log.Printf("Successfully made a bid\n Timestamp: %d\n", c.timestamp)
	}
}


func getResult(client *client) {
	client.timestamp++

	log.Printf("Getting result from auction \n Timestamp: %d\n", client.timestamp)
	resultClient := proto.NewAuctionClient(client.conn)
	result, err := resultClient.AskForResult(context.Background(), &proto.AskForResultMessage{Timestamp: int64(client.timestamp), Name: client.name})
	if err != nil {
		log.Printf("Error asking for result: %v\n", err)
	} else {
		if result.Timestamp > int64(client.timestamp) {
			client.timestamp = int(result.Timestamp)+1
		}
		//log result and timestamp
		log.Printf("Result: %d\n Timestamp: %d\n", result.Price, result.Timestamp)
	}
}

func askForBid(client *client) {
	client.timestamp++
	log.Printf("Asking for the current bid \n Timestamp: %d\n", client.timestamp)
	askForBidClient := proto.NewAuctionClient(client.conn)
	result, err := askForBidClient.AskForPrice(context.Background(), &proto.AskForPriceMessage{Timestamp: int64(client.timestamp), Name: client.name})
	if err != nil {
		log.Printf("Error asking for result: %v\n", err)
	} else {
		if result.Timestamp > int64(client.timestamp) {
			client.timestamp = int(result.Timestamp)+1
		}
		//log result and timestamp
		if result.Active {
			log.Printf("The auction is still going on and the current bid: %d\n Timestamp: %d\n", result.Price, result.Timestamp)
		} else {
		log.Printf("The auction is over and ended on the price: %d\n Timestamp: %d\n", result.Price, result.Timestamp)
		}
	}
}


func startHeartbeat(client *client) {
	const heartbeatInterval = 5 * time.Second

	// Periodically send heartbeat to the next node
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		err := sendHeartbeat(client)
		if err != nil {
			log.Printf("Error sending heartbeat: %v\n", err)
			log.Printf("Trying to connect to the next node\n")
			err := client.connect(5556)
			if err != nil {
				log.Printf("Error connecting to the next node: %v\n", err)
			} else {
				go startHeartbeat(client)
			}
		}
	}
}

func (c *client) connect(port int) error {
	c.timestamp++
	conn, err := grpc.Dial("localhost:" + strconv.Itoa(port), grpc.WithInsecure())
	if err != nil {
		log.Printf("could not connect to localhost:%d: %v\n", port, err)
		return err
	}
	log.Printf("Connected to the server at port: %d\n Timestamp: %d\n", port, c.timestamp)
	c.conn = conn
	return nil
}
func sendHeartbeat(client *client) error {
	heartbeatClient := proto.NewHeartbeatServiceClient(client.conn)
	_, err := heartbeatClient.Heartbeat(context.Background(), &proto.HeartbeatRequest{})
	return err
}
