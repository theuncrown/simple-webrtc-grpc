package main

import (
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	pb "miniprojek2/signaling"
	"net"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"google.golang.org/grpc"
)

var port = flag.Int("port", 50051, "The server port")

type server struct {
	pb.UnimplementedSignalingServer
	users   []string
	mu      sync.Mutex
	message map[string]*deque.Deque
}

type sdpResp struct {
}

func NewServer() *server {
	s := &server{
		users:   make([]string, 0),
		message: make(map[string]*deque.Deque),
	}
	return s
}

// func (s *server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
// 	var uuid = in.GetUid()

// 	//Decode the base64
// 	strDec, _ := b64.StdEncoding.DecodeString(in.GetPassword())
// 	decPass := string(strDec)

// 	if decPass != uuid {
// 		log.Printf("incorrect credential")
// 	}

// 	s.users = append(s.users, uuid)
// 	var message = ""

// 	return &pb.LoginReply{Result: "Hi client " + uuid + message}, nil
// }

func (s *server) GetLogin(uuid string) string {
	resp := ""
	for _, v := range s.users {
		if v != uuid {
			resp = v
		}
	}
	return resp
}

func (s *server) Signal(stream pb.Signaling_SignalServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch payload := in.Payload.(type) {

		case *pb.SignalRequest_Login:
			log.Println("signal -> login")
			uuid := payload.Login.Uid

			strDec, _ := b64.StdEncoding.DecodeString(payload.Login.Password)
			decPass := string(strDec)

			if decPass != uuid {
				log.Printf("incorrect credential")
			}

			s.users = append(s.users, uuid)
			stream.Send(&pb.SignalReply{
				Uuid: uuid,
				Payload: &pb.SignalReply_Login{
					Login: &pb.LoginReply{
						Uid:    uuid,
						Result: "login success..",
					},
				},
			})
			s.message[uuid] = deque.New()
		case *pb.SignalRequest_Offer:
			log.Println("signal -> offer")
			s.mu.Lock()
			message := &pb.SignalReply{
				Uuid: payload.Offer.Uid,
				Payload: &pb.SignalReply_Offer{
					Offer: &pb.OfferReceive{
						Uid: payload.Offer.Uid,
						Sdp: payload.Offer.Sdp,
					},
				},
			}
			//taro ke antrian map
			s.message[payload.Offer.Uid].PushBack(message)
			s.mu.Unlock()

		case *pb.SignalRequest_Answer:
			log.Println("signal -> answer")
			s.mu.Lock()
			msg := &pb.SignalReply{
				Uuid: payload.Answer.Uid,
				Payload: &pb.SignalReply_Answer{
					Answer: &pb.AnswerReceive{
						Uid: payload.Answer.Uid,
						Sdp: payload.Answer.Sdp,
					},
				},
			}
			s.message[payload.Answer.Uid].PushBack(msg)
			s.mu.Unlock()

		case *pb.SignalRequest_Ice:
			log.Println("signal -> ice")
			s.mu.Lock()
			ice := &pb.SignalReply{
				Uuid: payload.Ice.Uid,
				Payload: &pb.SignalReply_Ice{
					Ice: &pb.IceCandidate{
						Uid:           payload.Ice.Uid,
						CandidateInit: payload.Ice.CandidateInit,
					},
				},
			}
			s.message[payload.Ice.Uid].PushBack(ice)
			s.mu.Unlock()

		case *pb.SignalRequest_PingUid:
			log.Println("ping received")

			s.mu.Lock()
			loginResp := s.GetLogin(payload.PingUid)
			if loginResp != "" {
				s.message[payload.PingUid].PushBack(&pb.SignalReply{
					Uuid: payload.PingUid,
					Payload: &pb.SignalReply_OtherUuid{
						OtherUuid: loginResp,
					},
				})
			}
			s.mu.Unlock()

			resp := s.message[payload.PingUid]
			for {
				s.mu.Lock()
				if resp.Len() > 0 {
					if rep, ok := resp.PopFront().(*pb.SignalReply); ok {
						stream.Send(rep)
					}
				} else {
					s.mu.Unlock()
					break
				}
				s.mu.Unlock()
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	server := NewServer()
	pb.RegisterSignalingServer(s, server)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
