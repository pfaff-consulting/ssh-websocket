package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pfaff-consulting/ssh-websocket/internal/failban"
	"golang.org/x/crypto/ssh"
)

type WSMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Cols int    `json:"cols,omitempty"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
	// todo: generate SSH key by aws and use it as domain like xxx.exenv.pfaff.app
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, config *Config, failBan *failban.Manager) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	ip := r.RemoteAddr
	if failBan.IsBlocked(ip) {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("blocked"))
		return
	}

	_, msg, err := ws.ReadMessage()
	if err != nil {
		failBan.AddFailedAttempt(ip)
		log.Printf("Error while reading auth data: %v", err)
		return
	}

	var connectionInfo SSHConnectionInfo
	if err := json.Unmarshal(msg, &connectionInfo); err != nil {
		failBan.AddFailedAttempt(ip)
		log.Printf("Error while parsing auth data: %v", err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte("invalid login format"))
		return
	}
	connectionInfo.Host = config.Ssh.Host
	connectionInfo.Port = config.Ssh.Port

	sshConn, err := tryConnectToSsh(connectionInfo)
	if err != nil {
		failBan.AddFailedAttempt(ip)
		log.Printf("SSH auth error: %v", err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte("login failed"))
		return
	}

	handleSshStream(ws, sshConn)
}

func handleSshStream(wsConn *websocket.Conn, sshConn *SSHConnection) {
	defer func(Client *ssh.Client) {
		_ = Client.Close()
	}(sshConn.Client)
	defer func(Session *ssh.Session) {
		_ = Session.Close()
	}(sshConn.Session)

	go func() {
		_, _ = io.Copy(&wsWriter{wsConn}, sshConn.Stdout)
	}()
	go func() {
		_, _ = io.Copy(&wsWriter{wsConn}, sshConn.Stderr)
	}()

	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket closed: %v", err)
			return
		}

		var wsMsg WSMessage

		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			continue

		}
		switch wsMsg.Type {
		case "data":
			if _, err := sshConn.Stdin.Write([]byte(wsMsg.Data)); err != nil {
				log.Printf("Error writing to SSH stdin: %v", err)
				return
			}
		case "resize":
			if err := sshConn.Session.WindowChange(wsMsg.Rows, wsMsg.Cols); err != nil {
				log.Printf("Error changing window size: %v", err)
			}
		default:
			log.Printf("Unknown message type: %s", wsMsg.Type)
		}
	}
}
