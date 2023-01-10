/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	b64 "encoding/base64"
	"flag"
	"io"
	"log"
	pb "testgrpc/proto/signalling"
	"time"

	"github.com/google/uuid"
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
)

var dummyUsers = []string{"3281e47d-eb09-47cf-bda4-58e44725ede5", "3ac7a826-ec9f-406f-9675-8d4a0ba0699a"}

var conn, err = grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
var ctx, cancel = context.WithTimeout(context.Background(), 600*time.Second)

var a = dummyUsers[0]
var client = pb.NewLoginClient(conn)

func main() {
	flag.Parse()
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	defer cancel()

	//a := dummyUsers[0]
	//b := b64.StdEncoding.EncodeToString([]byte(a))

	// Login service
	login, login_err := client.TestLogin(ctx, &pb.LoginRequest{Uid: *uid, Password: password})
	if err != nil {
		log.Printf("error: %v", login_err)
	}
	log.Printf("%s", login.GetResult())

	//go Heartbeat(ctx)

	//go printMessage(ctx)

	//delayed the next action for 60 secs
	//time.Sleep(10 * time.Second)

	// test, tests_err := c.SendMessage(ctx, &pb.ChatRequest{Uid: *uid, Message: "Halo"})
	// if err != nil {
	// 	log.Printf("error: %v", tests_err)
	// }
	// log.Printf("%s", test.Result)

	runMessagingBidi(client)

	//time.Sleep(20 * time.Second)
	//Logout service
	logout, logout_err := client.TestLogout(ctx, &pb.LogoutRequest{Uid: *uid})
	if err != nil {
		log.Printf("error: %v", logout_err)
	}
	log.Printf("%s", logout.GetResult())
}

// getting online user info every 2 seconds
func Heartbeat(ctx context.Context) {
	for range time.Tick(time.Second * 2) {
		t, err := client.OnlineUserInfo(ctx, &pb.OnlineInfoRequest{Uid: *uid})
		if err != nil {
			log.Fatalf("Error : %v", err)
		}
		log.Printf("%s", t.GetResult())
	}
}

func printMessage(ctx context.Context) {
	for range time.Tick(time.Second * 2) {
		chat, chat_err := client.RecvMessage(ctx, &pb.ChatRequest{Uid: *uid})
		for {
			msg, err := chat.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("error: %v", chat_err)
			}
			log.Println(msg)
		}
	}
}

func runMessagingBidi(client pb.LoginClient) {
	messages := []*pb.ChatRequest{
		{
			Uid:     a,
			Message: "Hi",
		}, {
			Uid:     a,
			Message: "Halo",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.MessagingBidi(ctx)
	if err != nil {
		log.Fatalf("client.MessagingBidi failed: %v", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("client.MessagingBidi failed: %v", err)
			}
			log.Printf("Got message from %v : %v", in.Uid, in.Message)
		}
	}()
	for _, message := range messages {
		if err := stream.Send(message); err != nil {
			log.Fatalf("%v", err)
		}
	}
	stream.CloseSend()
	<-waitc
}
