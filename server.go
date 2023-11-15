package main

import (
    "bytes"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "sync"

    "github.com/gorilla/websocket"
)

// Rules engine is handled in Javascript by player starting the game
// If even the host player wants to make a move, they must still
// send a move request to the server
// The server will then relay the move request back to themself, and they
// will send a response back to the server
// This response will then be forwarded to the second player

type Game struct {
    Key int
    Player string
    Json string
    Mutex sync.Mutex
    Conns []*websocket.Conn
}

type Request struct {
    Key int
    Action string
    Payload string
    Player string
}

var games = make(map[int]*Game)
var upgrader = websocket.Upgrader{} // Default options

func NextGameIdx() int {
    max := -1
    for key := range games {
        if key > max {
            max = key
        }
    }
    return max+1
}

func ListSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer conn.Close()
    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return  
        }
        // Do we ever get any other types of messages?
        if msgType != websocket.TextMessage {
            log.Println("Not a text message")
            return
        }
        var req Request
        json.NewDecoder(bytes.NewBuffer(msg)).Decode(&req)
        switch req.Action {
            case "List":
                keys := make([]int, 0)
                for key := range games {
                    // Check if game has not been joined by two players
                    game := games[key]
                    if len(game.Conns) < 2 {
                        keys = append(keys, key)
                    }
                }
                jsn, _ := json.Marshal(keys)
                err = conn.WriteMessage(websocket.TextMessage, jsn)
                if err != nil {
                    log.Println(err)
                    continue
                }
        }
    }
}

func Socket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer conn.Close()
    player := -1
    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return  
        }
        // Do we ever get any other types of messages?
        if msgType != websocket.TextMessage {
            log.Println("Not a text message")
            return
        }
        var req Request
        json.NewDecoder(bytes.NewBuffer(msg)).Decode(&req)
        switch req.Action {
            case "Chat":
                game := games[req.Key]
                if game == nil {
                    log.Println("Game not found")
                    continue    
                }
                p := "Black";
                if player == 1 {
                    p = "White"
                }
                for _, conn := range game.Conns {
                    reply := Request{Action: "Chat", Key: game.Key, Player: p, Payload: req.Payload}
                    jsn, _ := json.Marshal(reply)
                    conn.WriteMessage(websocket.TextMessage, jsn)
                }
            case "New":  
                if player != -1 {
                    log.Println("Player already joined")
                    continue
                }
                player = 0
                game := &Game{Key: NextGameIdx(), Json: req.Payload, Conns: make([]*websocket.Conn, 1), Player: "black"}
                game.Conns[0] = conn
                games[game.Key] = game
                reply := Request{Action: "New", Key: game.Key} 
                jsn, _ := json.Marshal(reply)
                conn.WriteMessage(websocket.TextMessage, jsn)
            case "Join": 
                if player != -1 {
                    log.Println("Player already joined")
                    continue
                }
                player = 1
                game := games[req.Key]
                if len(game.Conns) < 2 {
                    game.Conns = append(game.Conns, conn)
                } else {
                    log.Println("Game full")
                    continue
                }
                // Next player
                reply := Request{Action: "Join", Key: game.Key, Payload: game.Json, Player: game.Player}
                jsn, _ := json.Marshal(reply)
                game.Conns[0].WriteMessage(websocket.TextMessage, jsn)
                game.Conns[1].WriteMessage(websocket.TextMessage, jsn)
            case "Move":
                game := games[req.Key]
                game.Mutex.Lock()
                game.Json = req.Payload
                if player == 0 {
                    game.Player = "white"
                } else {
                    game.Player = "black"
                }
                reply := Request{Action: "Move", Key: game.Key, Payload: game.Json, Player: game.Player}
                jsn, _ := json.Marshal(reply)
                game.Conns[0].WriteMessage(websocket.TextMessage, jsn)
                if len(game.Conns) == 2 {
                    game.Conns[1].WriteMessage(websocket.TextMessage, jsn)
                }
                game.Mutex.Unlock()
        }
    }
}

type HFunc func (http.ResponseWriter, *http.Request)

func Headers(fn HFunc) HFunc {
    return func (w http.ResponseWriter, req *http.Request) {
        //fmt.Println(req.Method)
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers",
            "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
        fn(w, req)
    }
}
func ServeStatic(w http.ResponseWriter, req *http.Request, file string) {
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    http.ServeFile(w, req, file)
}

func ServeLocalFiles(dirs []string) {
    for _, dirName := range dirs {
        fsDir := "static/" + dirName
        dir, err := os.Open(fsDir)
        if err != nil {
            log.Fatal(err)
        }
        files, err := dir.Readdir(0)
        if err != nil {
            log.Fatal(err)
        }
        for _, v := range files {
            //fmt.Println(v.Name(), v.IsDir())
            if v.IsDir() {
                continue
            }
            reqFile := dirName + "/" + v.Name()
            file := fsDir + "/" + v.Name()
            http.HandleFunc(reqFile, Headers(func (w http.ResponseWriter, req *http.Request) {ServeStatic(w, req, file)}))
        }
    }
}

func main() {
    log.SetFlags(0)
    ServeLocalFiles([]string{"", "/js", "/css"})
    http.HandleFunc("/ws", Socket)
    http.HandleFunc("/list", ListSocket)
    log.Fatal(http.ListenAndServe(":8001", nil))
}
