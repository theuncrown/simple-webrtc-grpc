# About project
this is my simple personal project about PION Webrtc and GRPC.
Written in GO.

Link:
- WebRTC : https://webrtc.org/
- PION main website: https://pion.ly/
- PION WebRTC : https://github.com/pion/webrtc

I'll try to add the UI soon.

# How to run
- run ```go mod tidy```
- open 3 terminal (1 for server, 2 for client)
- on terminal 1 : ```go run server/server.go```
- on terminal 2 : ```go run client/main.go -initiator false``` ( YOU NEED TO RUN THIS FIRST )
- on terminal 3 : ```go run client/main.go -initiator true```
