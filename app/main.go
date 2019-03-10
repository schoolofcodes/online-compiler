package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	b64 "encoding/base64"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/namsral/flag"
)

const (
	writeWait        = 10 * time.Second
	maxMessageSize   = 10000000
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	closeGracePeriod = 10 * time.Second
)

var (
	addr    = flag.String("url", ":8080", "Websocket address to serve on")
	root    = flag.String("root", "./", "Root folder where application files will be stored")
)

type Message struct {
	Name 	string	`json:"name"`
	Content	string 	`json:"content"`
}

var upgrader = websocket.Upgrader{}
var imageName = "frolvlad/alpine-gxx"
var cli, _ = client.NewEnvClient()

func process(ws *websocket.Conn, done chan struct{}) {
	defer close(done)
	defer ws.Close()

	ws.SetReadLimit(maxMessageSize)
	err := ws.SetReadDeadline(time.Now().Add(pongWait))
	isError(err)
	ws.SetPongHandler(func(string) error { _ = ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// first message will be the code
	// all following messages will be inputs
	// code and inputs are base64 encoded
	_, message, err := ws.ReadMessage()
	panicIfError(err)
	ctx := context.Background()
	code := string(message[:])
	data, err := b64.StdEncoding.DecodeString(code)
	panicIfError(err)
	code = string(data[:])
	code = strings.Replace(code, "\"", "\\\"", -1)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Cmd:             []string{"sh", "-c", "echo \"" + code + "\" | g++ --static -o /main.o -xc++ -"},
		NetworkDisabled: true,
		Image:           imageName,
		Tty:             true,
	}, nil, nil, "")
	panicIfError(err)
	panicIfError(cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}))
	_, err = cli.ContainerWait(ctx, resp.ID)
	panicIfError(err)
	commitResp, err := cli.ContainerCommit(ctx, resp.ID, types.ContainerCommitOptions{})
	_ = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	for {
		_, inputMessage, err := ws.ReadMessage()
		if err != nil {
			break
		}
		ctx := context.Background()
		input := string(inputMessage[:])
		inputData, err := b64.StdEncoding.DecodeString(input)
		panicIfError(err)
		input = string(inputData[:])
		input = strings.Replace(input, "\"", "\\\"", -1)

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Cmd:             []string{"sh", "-c", "echo \"" + input + "\" | ./main.o"},
			NetworkDisabled: true,
			Image:           commitResp.ID,
			Tty:             true,
		}, nil, nil, "")
		panicIfError(err)
		panicIfError(cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}))
		_, err = cli.ContainerWait(ctx, resp.ID)
		panicIfError(err)

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		panicIfError(err)
		_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
		result, err := ioutil.ReadAll(out)
		panicIfError(err)
		if err := ws.WriteMessage(websocket.TextMessage, result); err != nil {
			ws.Close()
		}

		_ = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	}
	_, _ = cli.ImageRemove(ctx, commitResp.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
}

func ping(ws *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			isError(ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)))
		case <-done:
			log.Println("Closing: ", ws.RemoteAddr())
			return
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	go log.Println("Connection from: ", r.RemoteAddr)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	panicIfError(err)

	defer ws.Close()

	done := make(chan struct{})
	go ping(ws, done)
	process(ws, done)

	err = ws.SetWriteDeadline(time.Now().Add(writeWait))
	isError(err)
	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	isError(err)
	time.Sleep(closeGracePeriod)
	ws.Close()
}

func main() {
	ctx := context.Background()
	if cli == nil {
		panic(errors.New("Docker cli not initialized"))
	}

	_, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	panicIfError(err)
	http.HandleFunc("/ws", serveWs)
	log.Println("Starting to serve on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}
