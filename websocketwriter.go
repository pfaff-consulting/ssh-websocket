package main

import "github.com/gorilla/websocket"

type wsWriter struct {
	ws *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	err := w.ws.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}
