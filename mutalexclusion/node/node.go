package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	proto "mutal/grpc"

	"google.golang.org/grpc"
)

type Node struct {
	proto.UnimplementedAccessServer
	proto.UnimplementedElectionServer
	port             int
	nextPort         int
	nextNextPort     int
	isLeader         bool
	votedFor         int
	conn             *grpc.ClientConn
	Queue            Queue
	resourceAvailable bool
	timestamp        int64
}

type Queue struct {
	Elements []int
	Size     int
}

var (
	port        = flag.Int("port", 0, "port number")
	nextPort    = flag.Int("next", 0, "next port number")
	nextNextPort = flag.Int("nextnext", 0, "next next port number")
)

func main() {
	flag.Parse()

	// Create a new server
	node := &Node{
		port:        *port,
		nextPort:    *nextPort,
		nextNextPort: *nextNextPort,
		votedFor:    *port,
		Queue:       Queue{Size: 100},
		timestamp:   0,
	}

	// set output to log file
	f, err := os.OpenFile(strconv.Itoa(node.port)+"logfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// Start the server
	go startServer(node)

	// Wait for input
	for {
		var input string
		_, _ = fmt.Scanln(&input)

		if input == "vote" {
			SendVote(node)
		}

		if input == "connect" {
			connectToServer(node, node.nextPort)
		}

		if input == "leader" {
			log.Printf("Timestamp: %d - The leader is: %d\n", node.timestamp, node.votedFor)
		}

		if input == "access" {
			if !node.isLeader {
				connectToServer(node, node.votedFor)
				requestAccess(node)
			} else {
				node.Queue.Enqueue(node.port)
			}
		}
	}
}

// leader stuff: when a node requests access they are added to the queue
func (n *Node) RequestAccess(ctx context.Context, in *proto.AccessMessage) (*proto.AccessResponse, error) {
	if n.timestamp < in.Timestamp {
		n.timestamp = in.Timestamp
	}
	n.timestamp++

	log.Printf("Timestamp: %d - Access requested by %d\n", n.timestamp, in.Id)
	n.Queue.Enqueue(int(in.Id))

	return &proto.AccessResponse{Ok: false}, nil
}

// when it is a node's turn to access the shared resource they send a request
func giveAccess(node *Node) {
	node.timestamp++
	accsesClient := proto.NewAccessClient(node.conn)
	_, err := accsesClient.GetAccess(context.Background(), &proto.AccessMessage{Id: int64(node.port)})
	if err != nil {
		log.Printf("Error sending access request to the next node: %v\n", err)
	}
}

// queue handler for leader
func handleQueue(node *Node) {
	for {
		if node.resourceAvailable {
			if !node.Queue.IsEmpty() {
				portToAccess := node.Queue.Dequeue()
				if portToAccess != node.port {
					node.resourceAvailable = false
					connectToServer(node, portToAccess)
					giveAccess(node)
				} else {
					node.resourceAvailable = false
					writeSharedFile(node)
					node.resourceAvailable = true
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// when a node is done accessing the shared resource they send a release request
func (n *Node) ReleaseAccess(ctx context.Context, in *proto.AccessMessage) (*proto.AccessResponse, error) {
	if n.timestamp < in.Timestamp {
		n.timestamp = in.Timestamp
	}
	n.timestamp++

	log.Printf("Timestamp: %d - Access released by %d\n", n.timestamp, in.Id)
	n.resourceAvailable = true
	return &proto.AccessResponse{Ok: true}, nil
}

// other nodes: When the leader tells them to access the shared resource they send a request
func (n *Node) GetAccess(ctx context.Context, in *proto.AccessMessage) (*proto.AccessResponse, error) {
	if n.timestamp < in.Timestamp {
		n.timestamp = in.Timestamp
	}
	n.timestamp++

	log.Printf("Timestamp: %d - Access granted to %d\n", n.timestamp, in.Id)
	go writeSharedFile(n)
	return &proto.AccessResponse{Ok: true}, nil
}

func writeSharedFile(node *Node) {
	// Open the shared file in append mode, create it if it doesn't exist
	file, err := os.OpenFile("logshared.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Write data to the file
	node.timestamp++

	message := fmt.Sprintf("Timestamp: %d - I have access: %d\n", node.timestamp, node.port)
	_, err = file.WriteString(message)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	// Sleep for 5 seconds
	time.Sleep(10 * time.Second)

	node.timestamp++
	message = fmt.Sprintf("Timestamp: %d - I return access: %d\n", node.timestamp, node.port)
	_, err = file.WriteString(message)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	if !node.isLeader {
		go releaseAccess(node)
	}
}

func releaseAccess(node *Node) {
	node.timestamp++
	log.Printf("Timestamp: %d - Releasing access\n", node.timestamp)
	accessClient := proto.NewAccessClient(node.conn)
	_, err := accessClient.ReleaseAccess(context.Background(), &proto.AccessMessage{Id: int64(node.port), Timestamp: node.timestamp})
	if err != nil {
		log.Printf("Error releasing access: %v\n", err)
	}
}

func requestAccess(node *Node) {
	node.timestamp++
	log.Printf("Timestamp: %d - Requesting access\n", node.timestamp)
	accessClient := proto.NewAccessClient(node.conn)
	_, err := accessClient.RequestAccess(context.Background(), &proto.AccessMessage{Id: int64(node.port), Timestamp: node.timestamp})
	if err != nil {
		log.Printf("Error requesting access: %v\n", err)
	}
}

func startServer(server *Node) {
	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("Started server at port: %d\n", server.port)

	// Register the grpc server and serve its listener
	proto.RegisterElectionServer(grpcServer, server)
	proto.RegisterAccessServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
}

// return connection to the next node
func connectToServer(node *Node, server int) {
	// Connect to the next node
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(server), grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Error connecting to the server: %v\n", err)
	}
	node.timestamp++
	fmt.Printf("Timestamp: %d - Connected to the server at port: %d\n", node.timestamp, server)
	node.conn = conn
}

func (node *Node) RequestVote(ctx context.Context, in *proto.ElectionMessage) (*proto.ElectionResponse, error) {
	if node.timestamp < in.Timestamp {
		node.timestamp = in.Timestamp
	}
	node.timestamp++

	log.Printf("Timestamp: %d - Received vote request from %d\n", node.timestamp, in.Id)

	if in.Id == int64(node.port) {
		log.Printf("Timestamp: %d - I am the leader\n", node.timestamp)
		node.isLeader = true
		node.resourceAvailable = true
		go handleQueue(node)
		return &proto.ElectionResponse{Ok: true}, nil
	}

	if in.Id > int64(node.port) {
		log.Printf("Timestamp: %d - I am not the leader\n", node.timestamp)
		node.isLeader = false
		node.votedFor = int(in.Id)

		// send vote request to the next node
		SendVote(node)
		return &proto.ElectionResponse{Ok: true}, nil
	}

	if in.Id < int64(node.port) {
		log.Printf("Timestamp: %d - I could be the leader\n", node.timestamp)
		node.isLeader = false
		node.votedFor = node.port
		SendVote(node)
		return &proto.ElectionResponse{Ok: true}, nil
	}

	log.Printf("Timestamp: %d - Something must have gone wrong in RequestVote\n", node.timestamp)
	return nil, nil
}

func SendVote(node *Node) {
	node.timestamp++
	log.Printf("Timestamp: %d - Sending vote request to the next node\n", node.timestamp)

	electionClient := proto.NewElectionClient(node.conn)
	_, err := electionClient.RequestVote(context.Background(), &proto.ElectionMessage{Id: int64(node.votedFor), Timestamp: node.timestamp})
	if err != nil {
		log.Printf("Error sending vote request to the next node: %v\n", err)
	}
}

// queue for access requests implementation

func (q *Queue) Enqueue(elem int) {
	if q.GetLength() == q.Size {
		fmt.Println("Overflow")
		return
	}
	q.Elements = append(q.Elements, elem)
}

func (q *Queue) Dequeue() int {
	if q.IsEmpty() {
		fmt.Println("Underflow")
		return 0
	}
	element := q.Elements[0]
	if q.GetLength() == 1 {
		q.Elements = nil
		return element
	}
	q.Elements = q.Elements[1:]
	return element // Slice off the element once it is dequeued.
}

func (q *Queue) GetLength() int {
	return len(q.Elements)
}

func (q *Queue) IsEmpty() bool {
	return len(q.Elements) == 0
}
