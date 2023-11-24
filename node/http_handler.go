package node

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/Filecoin-Titan/titan-container/api/types"
	cliutil "github.com/Filecoin-Titan/titan-container/cli/util"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	pingPeriod = time.Second * 10
	pingWait   = time.Second * 15
)

type WebsocketHandler struct {
	Client provider.Client
}

type terminalSizeQueue struct {
	resize chan remotecommand.TerminalSize
}

func (w *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	ret, ok := <-w.resize
	if !ok {
		return nil
	}
	return &ret
}

func (w *WebsocketHandler) ShellHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		id := path.Base(req.URL.Path)

		upgrader := websocket.Upgrader{
			ReadBufferSize:  0,
			WriteBufferSize: 0,
		}

		params := req.URL.Query()

		isTty := true
		connectStdin := true

		tty := params.Get("tty")
		if len(tty) > 0 {
			isTty = tty == "1"
		}

		conn, err := upgrader.Upgrade(writer, req, nil)
		if err != nil {
			log.Errorf("upgrader %v", err)
			return
		}

		var command []string

		for i := 0; true; i++ {
			v := params.Get(fmt.Sprintf("cmd%d", i))
			if 0 == len(v) {
				break
			}
			command = append(command, v)
		}

		var podIndex int
		podVar := params.Get("podIndex")
		if len(podVar) > 0 {
			val, err := strconv.ParseInt(podVar, 10, 64)
			if err != nil {
				log.Errorf("parse integer %v", err)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			podIndex = int(val)
		}

		var stdinPipeOut *io.PipeWriter
		var stdinPipeIn *io.PipeReader
		wg := &sync.WaitGroup{}

		var tsq remotecommand.TerminalSizeQueue
		var terminalSizeUpdate chan remotecommand.TerminalSize
		if isTty {
			terminalSizeUpdate = make(chan remotecommand.TerminalSize, 1)
			tsq = &terminalSizeQueue{resize: terminalSizeUpdate}
		}

		if connectStdin {
			stdinPipeIn, stdinPipeOut = io.Pipe()
			wg.Add(1)
			go websocketHandler(wg, conn, stdinPipeOut, terminalSizeUpdate)
		}

		l := &sync.Mutex{}
		stdout := cliutil.NewWsWriterWrapper(conn, types.ShellCodeStdout, l)
		stderr := cliutil.NewWsWriterWrapper(conn, types.ShellCodeStderr, l)

		subctx, subcancel := context.WithCancel(req.Context())
		wg.Add(1)
		go shellPingHandler(subctx, wg, conn)

		var stdinForExec io.Reader
		if connectStdin {
			stdinForExec = stdinPipeIn
		}
		result, err := w.Client.Exec(subctx, types.DeploymentID(id), podIndex, stdinForExec, stdout, stderr, command, isTty, tsq)
		subcancel()

		responseData := types.ShellResponse{}
		var resultWriter io.Writer
		encodeData := true
		resultWriter = cliutil.NewWsWriterWrapper(conn, types.ShellCodeResult, l)

		responseData.ExitCode = result.Code

		if err != nil {
			responseData.Message = err.Error()
		}

		if encodeData {
			encoder := json.NewEncoder(resultWriter)
			err = encoder.Encode(responseData)
		} else {
			_, err = resultWriter.Write([]byte{})
		}

		_ = conn.Close()

		wg.Wait()

		if stdinPipeIn != nil {
			_ = stdinPipeIn.Close()
		}

		if stdinPipeOut != nil {
			_ = stdinPipeOut.Close()
		}

		if terminalSizeUpdate != nil {
			close(terminalSizeUpdate)
		}
	}
}

func shellPingHandler(ctx context.Context, wg *sync.WaitGroup, ws *websocket.Conn) {
	defer wg.Done()
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			const pingWriteWaitTime = 5 * time.Second
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(pingWriteWaitTime)); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func websocketHandler(wg *sync.WaitGroup, conn *websocket.Conn, stdout io.Writer, terminalSizeUpdate chan<- remotecommand.TerminalSize) {
	defer wg.Done()

	for {
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pingWait))
		})

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if messageType != websocket.BinaryMessage || len(data) == 0 {
			continue
		}

		msgID := data[0]
		msg := data[1:]
		switch msgID {
		case types.ShellCodeStdin:
			_, err := stdout.Write(msg)
			if err != nil {
				return
			}
		case types.ShellCodeTerminalResize:
			var size remotecommand.TerminalSize
			r := bytes.NewReader(msg)
			// Unpack data, its just binary encoded data in big endian
			err = binary.Read(r, binary.BigEndian, &size.Width)
			if err != nil {
				return
			}
			err = binary.Read(r, binary.BigEndian, &size.Height)
			if err != nil {
				return
			}

			log.Debug("terminal resize received", "width", size.Width, "height", size.Height)
			if terminalSizeUpdate != nil {
				terminalSizeUpdate <- size
			}
		default:
			log.Error("unknown message ID on websocket", "code", msgID)
			return
		}
	}
}
