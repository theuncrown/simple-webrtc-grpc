package main

import (
	"context"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	pb "miniprojek2/signaling"
	"os"
	"strconv"
	"time"

	"encoding/json"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	//generate new UUID for user
	newUUID = uuid.New().String()
	addr    = flag.String("addr", "localhost:50051", "the address to connect to")
	//client UUID
	uid = flag.String("Uid", newUUID, "user Uid")
	//encode client UUID to password
	password = b64.StdEncoding.EncodeToString([]byte(newUUID))

	conn   *grpc.ClientConn
	client pb.SignalingClient

	peerConnection *webrtc.PeerConnection

	oppUuid     string
	isInitiator bool
	stream      pb.Signaling_SignalClient
)

func main() {
	var err error

	flag.Parse()
	args := os.Args[1:]

	isInitiator, _ = strconv.ParseBool(args[len(args)-1])
	log.Println(isInitiator)
	flag.Parse()

	conn, err = grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client = pb.NewSignalingClient(conn)

	defer conn.Close()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI()

	// Create a new RTCPeerConnection
	fmt.Println("Creating new RTC peer connection...")
	peerConnection, err = api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i != nil {
			iceCandInit := i.ToJSON()

			enc, err := json.Marshal(iceCandInit)
			if err != nil {
				log.Println("json encode error..")
			}

			stream.Send(&pb.SignalRequest{Uuid: *uid, Payload: &pb.SignalRequest_Ice{Ice: &pb.IceCandidate{
				Uid:           oppUuid,
				CandidateInit: string(enc),
			}}})
		}
	})

	runSignalBidi(client)

}

func runSignalBidi(client pb.SignalingClient) {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	stream, err = client.Signal(ctx)
	if err != nil {
		log.Fatalf("client.SignalBidi failed: %v", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("client.LoginBidi failed: %v", err)
			}
			switch payload := in.Payload.(type) {
			case *pb.SignalReply_Login:
				log.Println("login success..")
			case *pb.SignalReply_Offer:
				log.Println("receive offer..")
				var offer webrtc.SessionDescription

				log.Println("decode offer SDP..")
				err := json.Unmarshal(payload.Offer.Sdp, &offer)
				log.Printf("offer : %v", offer)

				if err != nil {
					log.Println("set remote offer..")
				}

				if err := peerConnection.SetRemoteDescription(offer); err != nil {
					log.Printf("SetRemoteDescription error: %v", err)
				}

				log.Println("create answer..")
				answ, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					log.Printf("error : %v", err)
				}
				log.Printf("answer = %v", answ)
				peerConnection.SetLocalDescription(answ)
				resp, encError := json.Marshal(answ)
				if encError != nil {
					log.Println("Encode error..")
				}

				stream.Send(&pb.SignalRequest{
					Uuid: *uid,
					Payload: &pb.SignalRequest_Answer{
						Answer: &pb.AnswerRequest{
							Uid: oppUuid,
							Sdp: resp,
						},
					},
				})

			case *pb.SignalReply_Answer:
				var answ webrtc.SessionDescription
				err := json.Unmarshal(payload.Answer.Sdp, &answ)
				if err != nil {
					log.Printf("Decrypt answer error : %v", err)
				}

				setRemoteDesc_err := peerConnection.SetRemoteDescription(answ)
				if setRemoteDesc_err != nil {
					log.Printf("Set Remote Description error : %v", setRemoteDesc_err)
				}

			case *pb.SignalReply_Ice:
				log.Println("receive ice..")
				var iceCand webrtc.ICECandidateInit
				err := json.Unmarshal([]byte(payload.Ice.CandidateInit), &iceCand)
				if err != nil {
					log.Printf("ice error: %v", err)
				}
				log.Println(iceCand)
				peerConnection.AddICECandidate(iceCand)
			case *pb.SignalReply_OtherUuid:
				if oppUuid == "" {
					oppUuid = payload.OtherUuid
					if isInitiator == true {

						// temporary
						peerConnection.CreateDataChannel("test data channel", nil)

						log.Println("create offer..")
						offer, err := peerConnection.CreateOffer(nil)
						if err != nil {
							log.Printf("Create Offer error : %v", err)
						}

						log.Println("set local description..")
						setLocalDesc_err := peerConnection.SetLocalDescription(offer)
						if setLocalDesc_err != nil {
							log.Printf("Set Local Desc error : %v", setLocalDesc_err)
						}

						log.Println("encode SDP..")
						encSdp, encSdp_err := json.Marshal(offer)
						if encSdp_err != nil {
							log.Printf("encrypt SDP error : %v", encSdp_err)
						}

						stream.Send(&pb.SignalRequest{
							Uuid: *uid,
							Payload: &pb.SignalRequest_Offer{
								Offer: &pb.OfferRequest{
									//target uid + target spd
									Uid: oppUuid,
									Sdp: encSdp},
							},
						})
					}
				}
			}
		}
	}()

	stream.Send(&pb.SignalRequest{
		Uuid: *uid,
		Payload: &pb.SignalRequest_Login{
			Login: &pb.LoginRequest{
				Uid:      *uid,
				Password: password,
			},
		},
	})
	Heartbeat(stream)
}

func Heartbeat(stream pb.Signaling_SignalClient) {
	for range time.Tick(time.Second * 1) {
		err := stream.Send(&pb.SignalRequest{Uuid: *uid, Payload: &pb.SignalRequest_PingUid{PingUid: *uid}})
		if err != nil {
			log.Fatalf("Error : %v", err)
		}
	}
}
