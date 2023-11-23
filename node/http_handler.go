package node

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/Filecoin-Titan/titan-container/api/types"
	cliutil "github.com/Filecoin-Titan/titan-container/cli/util"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

type WebsocketHandler struct {
	Client provider.Client
}

type DeploymentExecHandler http.HandlerFunc

var Upgrader websocket.Upgrader

type websocketReader struct {
	client *websocket.Conn
	tsq    *terminalSizeQueue
}

type terminalSizeQueue struct {
	resize chan remotecommand.TerminalSize
}

func (w *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	ret := <-w.resize
	return &ret
}

func (w *websocketReader) Read(p []byte) (n int, err error) {
	messageType, data, err := w.client.ReadMessage()
	if err != nil {
		return 0, err
	}

	if messageType != websocket.BinaryMessage {
		return 0, nil
	}

	msgID := data[0]
	msg := data[1:]

	switch msgID {
	case types.ShellCodeTerminalResize:
		var width, height uint16

		if err = binary.Read(bytes.NewBuffer(msg[:2]), binary.BigEndian, &width); err != nil {
			return
		}

		if err = binary.Read(bytes.NewBuffer(msg[2:4]), binary.BigEndian, &height); err != nil {
			return
		}

		w.tsq.resize <- remotecommand.TerminalSize{Width: width, Height: height}
		return 0, nil
	case types.ShellCodeStdin:
		return copy(p, msg), nil
	default:
	}

	return 0, nil
}

func (w *WebsocketHandler) DeploymentExecHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		id := path.Base(req.URL.Path)

		c, err := Upgrader.Upgrade(writer, req, nil)
		if err != nil {
			log.Errorf("%v", err)
			return
		}

		command := []string{"sh"}
		if req.URL.Query().Get("cmd") != "" {
			command = strings.Split(req.URL.Query().Get("cmd"), ",")
		}

		tsq := &terminalSizeQueue{
			resize: make(chan remotecommand.TerminalSize),
		}

		reader := &websocketReader{
			client: c,
			tsq:    tsq,
		}

		l := &sync.Mutex{}
		stdout := cliutil.NewWsWriterWrapper(c, types.ShellCodeStdout, l)
		stderr := cliutil.NewWsWriterWrapper(c, types.ShellCodeStderr, l)

		if err = w.Client.DeploymentCmdExec(req.Context(), types.DeploymentID(id), reader, stdout, stderr, command, true, tsq); err != nil {
			if err := c.Close(); err != nil {
				log.Errorf("close connection: %v", err)
			}
		}
	}
}
