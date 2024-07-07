package main

import (
	"net/http"
	"log"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"strings"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Cursor struct {
	SessionID string `json:"sessionId"`
	Method    string `json:"method"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
}

type Method int

const (
	LeaveMethod Method = iota
	MoveMethod
)

func (m Method) String() string {
	switch m {
	case 0:
		return "leave"
	case 1:
		return "move"
	}
	return ""
}

var (
	clients = make(map[string]*websocket.Conn)
	cursors = make(map[string]Cursor, 100)
	ch      = make(chan Cursor)
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	sessionID := strings.Split(ws.RemoteAddr().String(), ":")[1]
	clients[sessionID] = ws
	
	var cursorsMove = make(map[string]Cursor, len(cursors))
	
	cursorData := make([]Cursor, 0, len(cursors))
	
	for _, cursor := range cursors {
		if cursor.Method != LeaveMethod.String() {
			cursorsMove[cursor.SessionID] = cursor
		} else {
			delete(cursors, cursor.SessionID)
			delete(clients, sessionID)
		}
	}
	for _, cursor := range cursorsMove {
		cursorData = append(cursorData, cursor)
	}
	
	cursor := Cursor{
		SessionID: sessionID,
		Method:    MoveMethod.String(),
		X:         -100,
		Y:         -100,
	}
	
	cursors[sessionID] = cursor
	for {
		if err := ws.ReadJSON(&cursor); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				log.Printf("Client disconnected: %v", sessionID)
				cursor.Method = LeaveMethod.String()
				cursors[sessionID] = cursor
			} else {
				log.Printf("error in readJson: %v", err)
			}
			ws.Close()
			return
		}
		cursors[sessionID] = cursor
		ch <- cursor
	}
}

func writeCoords() {
	for {
		cursorData := <-ch
		for _, conn := range clients {
			if err := conn.WriteJSON(cursorData); err != nil {
				log.Printf("error in writeJson: %v", err)
				conn.Close()
			}
			
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)
	
	go writeCoords()
	
	log.Println("http server started on :4567")
	err := http.ListenAndServe(":4567", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
