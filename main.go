package main

import (
	"net/http"
	"log"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"strings"
	"fmt"
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

var cursors = make(map[string]Cursor, 100)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	
	sessionID := strings.Split(ws.RemoteAddr().String(), ":")[1]
	
	var cursorsMove = make(map[string]Cursor, len(cursors))
	
	cursorData := make([]Cursor, 0, len(cursors))
	fmt.Println(len(cursors))
	for _, c := range cursors {
		fmt.Printf("%q ", c.SessionID)
	}
	fmt.Println()
	
	for _, cursor := range cursors {
		if cursor.Method != LeaveMethod.String() {
			// fmt.Println("move", cursor.SessionID)
			cursorsMove[cursor.SessionID] = cursor
		} else {
			// fmt.Println("not move", cursor.SessionID)
			delete(cursors, cursor.SessionID)
		}
	}
	for _, cursor := range cursorsMove {
		cursorData = append(cursorData, cursor)
	}
	// fmt.Println("cursorData:", cursorData)
	if err = ws.WriteJSON(cursorData); err != nil {
		log.Printf("error in writeJson: %v", err)
		ws.Close()
		return
	}
	cursor := Cursor{
		SessionID: sessionID,
		Method:    MoveMethod.String(),
		X:         -100,
		Y:         -100,
	}
	cursor.SessionID = sessionID
	cursor.Method = MoveMethod.String()
	
	cursors[sessionID] = cursor
	for {
		if err := ws.ReadJSON(&cursor); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				log.Printf("Client disconnected: %v", sessionID)
				// levCurs := cursors[sessionID]
				cursor.Method = LeaveMethod.String()
				cursors[sessionID] = cursor
				// fmt.Println("cursors", cursor)
			} else {
				log.Printf("error in readJson: %v", err)
			}
			ws.Close()
			return
		}
		cursors[sessionID] = cursor
		// fmt.Println("cursors", cursors[sessionID].Method)
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)
	
	log.Println("http server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
