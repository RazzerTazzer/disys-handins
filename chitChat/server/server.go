package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	proto "chitChat/grpc"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
    proto.UnimplementedChatServiceServer
    mu      sync.Mutex
    timestamp int64
    port    int
    clientMap map[int32]proto.ChatService_ReceiveMessageServer
}

func main() {
    // Log to file instead of consolew
    f, err := os.OpenFile("logfile.server", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("error opening file: %v", err)
    }
    defer f.Close()
    mw := io.MultiWriter(os.Stdout, f)
    log.SetOutput(mw)

    
    // Create a server struct
    server := &Server{
        port:    5454,
        mu:      sync.Mutex{},
        timestamp: 0,
        clientMap: make(map[int32]proto.ChatService_ReceiveMessageServer),
    }

    // Start the server
    go startServer(server)

    // Keep the server running until it is manually quit
	select {}
}

func startServer(server *Server) {

    // Create a new grpc server
    grpcServer := grpc.NewServer()

    // Make the server listen at the given port (convert int port to string)
    listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))

    if err != nil {
        log.Fatalf("Could not create the server %v", err)
    }
    log.Printf("Started server at port: %d\n", server.port)

    // Register the grpc server and serve its listener
    proto.RegisterChatServiceServer(grpcServer, server)
    serveError := grpcServer.Serve(listener)
    if serveError != nil {
        log.Fatalf("Could not serve listener")
    }
}

func (s *Server) SendMessage(ctx context.Context, msg *proto.ChatMessage) ( *emptypb.Empty, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Increment Lamport timestamp
    receivedTimestamp := msg.Timestamp
    s.timestamp = max(s.timestamp, receivedTimestamp) + 1


    // Process the message
    fmt.Printf("Received message: %s\n", msg.Content)
    fmt.Printf("Timestamp: %d\n", msg.Timestamp)

    

    // Broadcast the message to all connected client
    for key, value := range s.clientMap{
    fmt.Println("Key:", key, "Value:", value)
    s.timestamp++
        err := value.Send(&proto.ChatMessage{
            Content:   msg.Content,
            Timestamp: s.timestamp,
        })
        if err != nil {
            fmt.Printf("Error sending message: %v\n", err)
        }   else {
            fmt.Printf("Message sent with timestamp: %d\n", s.timestamp)
        }   
    }


    return &emptypb.Empty{}, nil
}


func (s *Server) ReceiveMessage(in *proto.ChatMessage, stream proto.ChatService_ReceiveMessageServer) error {
	s.mu.Lock()
    s.clientMap[in.Port] = stream
    s.mu.Unlock()

	select{}
}


func (s *Server) LeaveChat(ctx context.Context, in *proto.ChatMessage) (*proto.ChatMessage, error) {
    s.mu.Lock()
	
    delete(s.clientMap, in.Port)
	if in.Timestamp > s.timestamp {
		s.timestamp = in.Timestamp 
	}
	s.timestamp++
    s.mu.Unlock()
    
    return &proto.ChatMessage{
		Content:   in.Content,
		Timestamp: s.timestamp,
		Port: int32(s.port),
	}, nil
}



func max(a, b int64) int64 {
    if a > b {
        return a
    }
    return b
}


