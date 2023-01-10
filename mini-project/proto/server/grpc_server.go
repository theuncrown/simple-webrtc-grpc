package main

import (
	"context"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	pb "testgrpc/proto/signalling"

	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

var users = make([]string, 0)

// var dummyUsers = []string{"3281e47d-eb09-47cf-bda4-58e44725ede5", "3ac7a826-ec9f-406f-9675-8d4a0ba0699a"}
// var dummyMessage = [][]string{
// 	{"3281e47d-eb09-47cf-bda4-58e44725ede5", "Hi"},
// 	{"3281e47d-eb09-47cf-bda4-58e44725ede5", "Test 123"},
// 	{"3ac7a826-ec9f-406f-9675-8d4a0ba0699a", "Halo"},
// 	{"3ac7a826-ec9f-406f-9675-8d4a0ba0699a", "Apa ada orang?"},
// }

//var dummyMessage = [][]string{}

var userChats [][]string

type server struct {
	pb.UnimplementedLoginServer
	mu sync.Mutex
}

func (s *server) MessagingBidi(stream pb.Login_MessagingBidiServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		uid := in.GetUid()
		message := in.GetMessage()
		temp := [][]string{{uid, message}}

		userChats = append(userChats, temp...)
		fmt.Println(userChats)
		size := len(userChats)

		log.Printf("Array length : %v", size)

		for i := 0; i < size; i++ {
			sender := userChats[i][0]
			senderMessage := userChats[i][1]
			resp := pb.ChatRequest{Uid: sender, Message: senderMessage}
			log.Printf("UID : %v", sender)
			log.Printf("Message : %v", senderMessage)
			if err := stream.Send(&resp); err != nil {
				log.Printf("send error %v", err)
			}
		}
		userChats = nil
	}
}

func (s *server) TestLogin(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	var uuid = in.GetUid()
	var pass = in.GetPassword()

	//Decode the base64
	sDec, _ := b64.StdEncoding.DecodeString(in.GetPassword())

	log.Printf("Received: %v", uuid)
	log.Printf("Received: %v", pass)
	log.Printf("Received: %v", string(sDec))
	decPass := string(sDec)

	if decPass != uuid {
		log.Printf("incorrect credential")
	}

	users = append(users, uuid)
	var message = ""

	log.Printf("User(s) Count: %v User(s)", len(users))
	log.Print(users)

	return &pb.LoginReply{Result: "Hi client " + uuid + message}, nil
}

func (s *server) TestLogout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	log.Printf("Received: %v", in.GetUid())
	var uuid = in.GetUid()
	//remove specific uuid from array
	//remove(users, uuid)
	for i, v := range users {
		if v == uuid {
			users = append(users[:i], users[i+1:]...)
		}
	}
	log.Printf("Removed %v", uuid)
	return &pb.LogoutReply{Result: "Goodbye " + uuid}, nil
}

// Getting total user that online
func (s *server) OnlineUserInfo(ctx context.Context, in *pb.OnlineInfoRequest) (*pb.OnlineInfoReply, error) {
	var count = strconv.Itoa(len(users))
	var message = "Total User(s) Online: " + count + " User(s)"
	return &pb.OnlineInfoReply{Result: message}, nil
}

// Getting message from other uid from slice
func (s *server) RecvMessage(in *pb.ChatRequest, chatService pb.Login_RecvMessageServer) error {
	log.Printf("fetch response message for id : %v", in.Uid)
	var sender = ""
	var senderMessage = ""
	//size := len(dummyMessage)
	size := len(userChats)

	for i := 0; i < size; i++ {
		// if dummyMessage[i][0] != in.Uid {
		// 	sender = dummyMessage[i][0]
		// 	senderMessage = dummyMessage[i][1]
		// 	resp := pb.ChatReply{Uid: sender, Message: senderMessage}
		// 	log.Printf("UID : %v", sender)
		// 	log.Printf("Message : %v", senderMessage)
		// 	if err := chatService.Send(&resp); err != nil {
		// 		log.Printf("send error %v", err)
		// 	}
		// }
		if userChats[i][0] != in.Uid {
			sender = userChats[i][0]
			senderMessage = userChats[i][1]
			resp := pb.ChatReply{Uid: sender, Message: senderMessage}
			log.Printf("UID : %v", sender)
			log.Printf("Message : %v", senderMessage)
			if err := chatService.Send(&resp); err != nil {
				log.Printf("send error %v", err)
			}
		}
	}

	//dummyMessage = nil
	userChats = nil

	return nil
}

// add messages to slice
func (s *server) SendMessage(ctx context.Context, in *pb.ChatRequest) (*pb.ChatReply, error) {
	temp := [][]string{{in.Uid, in.Message}}
	//dummyMessage = append(dummyMessage, temp...)
	userChats = append(userChats, temp...)

	var message = "Pesan berhasil terkirim"

	return &pb.ChatReply{Result: message}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterLoginServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
