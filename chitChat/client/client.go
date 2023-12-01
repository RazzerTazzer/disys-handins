package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log"
	"os"

	proto "chitChat/grpc"

	"google.golang.org/grpc"
)

type Client struct {
	name string
	port int
	timestamp int64
	conn proto.ChatServiceClient
}

var (
	name = flag.String("name", "lisselotte", "name")
	port = flag.Int("port", 0, "port")

)


func main() {
	flag.Parse() 

	// Create a gRPC connection to the server
	conn, err := grpc.Dial("localhost:5454", grpc.WithInsecure())
	if err != nil {
		log.Printf("Error connecting to the server: %v\n", err)
	
		return
	}
	defer conn.Close()

	// Create a gRPC client
	client := &Client{
	name: *name,
	port: *port,
	timestamp: 0,
	conn: proto.NewChatServiceClient(conn),
	}
	

	// log to file instead of console
	//set output to log file
	f, err := os.OpenFile("logfile."+ client.name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	
	// Start receiving messages in a separate goroutine
	go receiveMessages(client)

	// Example: Send a message
	sendMessage(client.name + " has joined the chat", client)

	log.Println("Enter text (Welcome to the chat. type 'leave' to exit):")

	reader := bufio.NewReader(os.Stdin)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading input:", err)
			return
		}

		input = input[:len(input)-1] // Remove the newline character

		if input == "leave" {
			LeaveChat(client)
		} else {
			sendMessage(input, client)
		}
	}
}


func sendMessage(content string, client *Client) {
	client.timestamp++;

	// Send message to the server
	_, err := client.conn.SendMessage(context.Background(), &proto.ChatMessage{
		Content:   content,
		Timestamp: client.timestamp,
		Port: int32(client.port),	
	})

	if err != nil {
		log.Printf("Error sending message: %v\n", err)
	} else {
		log.Printf("message sent with timestamp: %d\n", client.timestamp)
	}
}

func LeaveChat(client *Client) {
	// Send message to the server
	client.timestamp++;
	_, err := client.conn.LeaveChat(context.Background(), &proto.ChatMessage{
		Content:   client.name + " has left the chat",
		Timestamp: client.timestamp,
		Port: int32(client.port),	
	})

	if err != nil {
		log.Printf("Error sending message: %v\n", err)
	} else {
		log.Printf("told severgoodbye: %d\n", client.timestamp)

	}

	sendMessage(client.name + " has left the chat", client)

}



func receiveMessages(client *Client) {
	stream, err := client.conn.ReceiveMessage(context.Background(), &proto.ChatMessage{Port: int32(client.port)})
	if err != nil {
		log.Printf("Error creating stream: %v\n", err)
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Printf("Error receiving message: %v\n", err)
			break
		}
		// Process the received message
		if msg != nil {
			if msg.Timestamp > client.timestamp {
				client.timestamp = msg.Timestamp
			}
			client.timestamp++
			log.Printf(msg.Content)
			log.Printf("Timestamp: %d\n", msg.Timestamp)
		}
	}
}
