package main

import (
	proto "auction/grpc"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type Node struct {
	proto.UnimplementedElectionServer
	proto.UnimplementedAuctionServer
	proto.UnimplementedHeartbeatServiceServer
	port 	 int
	nextPort int
	nextNextPort int
	isLeader bool
	votedFor int
	conn *grpc.ClientConn
	timestamp int
	autionActive bool
}

type Auction struct {
	mu            sync.Mutex
	currentBid int
	currentBidder string
}

var (
	port = flag.Int("port", 0, "port number")
	nextPort = flag.Int("next", 0, "next port number")
	nextNextPort = flag.Int("nextnext", 0, "next port number")
)

var auction = Auction{currentBid: 0, currentBidder: ""}
 
func main() {
	flag.Parse()

	// Create a new server
	node := &Node{
		port: *port,
		nextPort: *nextPort,
		nextNextPort: *nextNextPort,
		votedFor: *port,
		timestamp: 0,
		autionActive: true,
	}	
	
	f, err := os.OpenFile("logfile."+ strconv.Itoa(node.port), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	go startServer(node)

	for {
		var input string
		_, _ = fmt.Scanln(&input)

		if input == "connect" {
			connectToServer(node, node.nextPort)
			go startHeartbeat(node)
		}
		
		if input == "vote" {
			SendVote(node)
		}

		if input == "leader" {
			log.Printf("Am I the leader? %t\n", node.isLeader)
		}
	}
}

func connectToServer(node *Node, server int) {
    // Connect to next node
    conn, err := grpc.Dial("localhost:" + strconv.Itoa(server), grpc.WithInsecure())
    if err != nil {
        fmt.Printf("Error connecting to the server: %v\n", err)
    } else {
		node.timestamp+= 1
		fmt.Printf("Connected to the server at port: %d\n lamport: %d\n", server, node.timestamp)
		node.conn = conn
		
	}
    
}

func startHeartbeat(node *Node) {
	const heartbeatInterval = 5 * time.Second

	// Periodically send heartbeat to the next node
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		err := sendHeartbeat(node)
		if err != nil {
			log.Printf("Error sending heartbeat: %v\n", err)
			log.Printf("Trying to connect to nextnext node\n")
			connectToServer(node, node.nextNextPort)
			go startHeartbeat(node)
			node.votedFor = *port
			SendVote(node)
			}
		}
	}


func sendHeartbeat(node *Node) error {
	heartbeatClient := proto.NewHeartbeatServiceClient(node.conn)
	_, err := heartbeatClient.Heartbeat(context.Background(), &proto.HeartbeatRequest{})
	return err
}

func (n *Node) Heartbeat(ctx context.Context, in *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	return &proto.HeartbeatResponse{}, nil
}


func startServer(server *Node) {
	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	server.timestamp+= 1
	log.Printf("Started server at port: %d\n lamport: %d\n", server.port, server.timestamp)

	// Register the grpc server and serve its listener
	proto.RegisterHeartbeatServiceServer(grpcServer, server)
	proto.RegisterElectionServer(grpcServer, server)
	proto.RegisterAuctionServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
}


func SendVote(node *Node) {
	node.timestamp+= 1
    log.Printf("Sending vote request to next node\n lamport: %d\n", node.timestamp)

    electionClient := proto.NewElectionClient(node.conn)
    result, err := electionClient.RequestVote(context.Background(), &proto.ElectionMessage{Id: int64(node.votedFor), Timestamp: int64(node.timestamp)})
    if err != nil {
        log.Printf("Error sending vote request to next node: %v\n", err)
    } else { 
		if (result.Timestamp > int64(node.timestamp)) {	
			node.timestamp = int(result.Timestamp)+1
		}
		log.Printf("Received vote response from next node\n lamport: %d\n", node.timestamp)
	}

	 
}

func (node *Node) RequestVote(ctx context.Context, in *proto.ElectionMessage) (*proto.ElectionResponse, error) {
	//check lamport timestamp
	if (in.Timestamp > int64(node.timestamp)) {
		node.timestamp = int(in.Timestamp)+1
	}
	//receive vote request and lamport timestamp
	log.Printf("Received vote request from %d\n lamport: %d\n", in.Id, node.timestamp)

	if in.Id == int64(node.port) {
		log.Printf("I am the leader\n")
		node.isLeader = true
		return &proto.ElectionResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
	}

	if in.Id > int64(node.port) {
		log.Printf("I am not the leader\n")
		node.isLeader = false
		node.votedFor = int(in.Id)

		//send vote request to next node
		SendVote(node)
		return &proto.ElectionResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
	}
	
	if in.Id < int64(node.port) {
		log.Printf("I could be the leader\n")
		node.isLeader = false
		node.votedFor = node.port
		SendVote(node)
		return &proto.ElectionResponse{Ok: true}, nil
	}
	
	log.Printf("something must have went wrong in RequestVote\n")
	return nil, nil	
}


func (node *Node) Bid(ctx context.Context, in *proto.BidMessage) (*proto.BidResponse, error) {
    // Use a mutex to ensure atomicity when updating shared data
    auction.mu.Lock()
    defer auction.mu.Unlock()

	 // Timer for auction to stop
		if (node.autionActive) {
			go func() {
				time.Sleep(60 * time.Second)
				node.autionActive = false
			} ()	
		}

    if in.Timestamp > int64(node.timestamp) {
        node.timestamp = int(in.Timestamp) + 1
    }

    log.Printf("Received bid request from %s\n lamport: %d\n", in.Name, node.timestamp)

    if !node.autionActive {
        log.Printf("Auction is over, no more bids\n")
        return &proto.BidResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
    }

    if !node.isLeader {
        if in.Bid > int64(auction.currentBid) {
            auction.currentBid = int(in.Bid)
            auction.currentBidder = in.Name
            log.Printf("New highest bid: %d\n", auction.currentBid)
            return &proto.BidResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
        }
    }

    if in.Bid < int64(auction.currentBid) {
        log.Printf("Bid is lower than the current bid\n")
        return &proto.BidResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
    }

    auction.currentBid = int(in.Bid)
    auction.currentBidder = in.Name

    // Check if the timeout goroutine has been started
  
        // Your existing code for sending bids to next nodes
        if err := node.sendBidToNextNode(in.Bid, in.Name, node.nextPort); err != nil {
            log.Printf("Error sending bid to next node: %v\n", err)
        }

        // Send bid to next-next node
        if err := node.sendBidToNextNode(in.Bid, in.Name, node.nextNextPort); err != nil {
            log.Printf("Error sending bid to next-next node: %v\n", err)
        }

    return &proto.BidResponse{Ok: true, Timestamp: int64(node.timestamp)}, nil
}



func (node *Node) sendBidToNextNode(bidValue int64, bidderName string, nextPort int) error {
	node.timestamp+= 1
	// Check if node.conn is nil before using it
    if node.conn == nil {
        log.Println("Error: gRPC client connection is nil")
        return errors.New("gRPC client connection is nil")
    }

    connectToServer(node, nextPort)
    bidClient := proto.NewAuctionClient(node.conn)
    _, err := bidClient.Bid(context.Background(), &proto.BidMessage{Bid: bidValue, Name: bidderName, Timestamp: int64(node.timestamp)})
    return err
}


func (node *Node) AskForPrice(ctx context.Context, in *proto.AskForPriceMessage) (*proto.AskForPriceResponse, error) {
	if (in.Timestamp > int64(node.timestamp)) {
		node.timestamp = int(in.Timestamp)+1
	}

	log.Printf("Received ask for price request from %s\n lamport: %d\n", in.Name, node.timestamp)
	if(node.autionActive) {
		return &proto.AskForPriceResponse{Price: int64(auction.currentBid), Timestamp: int64(node.timestamp), Active: true}, nil
	} else {
		return &proto.AskForPriceResponse{Price: int64(auction.currentBid), Timestamp: int64(node.timestamp), Active: false}, nil
	}
}

func (node *Node) AskForResult(ctx context.Context, in *proto.AskForResultMessage) (*proto.AskForResultResponse, error) {
	if (in.Timestamp > int64(node.timestamp)) {
		node.timestamp = int(in.Timestamp)+1
	}

	log.Printf("Received ask for result request from %s\n lamport: %d\n", in.Name, node.timestamp)
	return &proto.AskForResultResponse{Price: int64(auction.currentBid), Timestamp: int64(node.timestamp), Name: auction.currentBidder}, nil
}
