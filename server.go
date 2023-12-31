package main

import (
    "bytes"
    "encoding/json"
    //"fmt"
    "log"
    "net/http"
    "os"
    "sync"

    "github.com/gorilla/websocket"
    ai "github.com/aorliche/web-nongrid-go/ai"
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
    BoardPlan string
    Mutex sync.Mutex
    Conns []*websocket.Conn
    RecvChan chan bool
    History []*ai.Board
}

type Request struct {
    Key int
    Action string
    Payload string
    Player string
    BoardPlan string
}

type PointsNeighbors struct {
    Points []int
    Neighbors [][]int
}

type AIMove struct {
    Point int
}

func (pn *PointsNeighbors) ToBoard() *ai.Board {
    return &ai.Board{
        Points: pn.Points,
        Neighbors: pn.Neighbors,
        NPlayers: 2,
        Turn: 0,
    }
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

func GetBoards() []string {
    dir, err := os.Open("boards")
    if err != nil {
        log.Println(err)
        return make([]string, 0)
    }
    files, err := dir.Readdir(0)
    if err != nil {
        log.Println(err)
        return make([]string, 0)
    }
    boards := make([]string, 0)
    for _, v := range files {
        if v.IsDir() {
            continue
        }
        boards = append(boards, v.Name())
    }
    return boards
}

func GetBoard(name string) (string, error) {
    dat, err := os.ReadFile("boards/" + name)
    if err != nil {
        log.Println(err)
        return "", err
    }
    return string(dat), err
}

func AddBoard(name string, json string) error {
    err := os.WriteFile("boards/" + name, []byte(json), 0644)
    if err != nil {
        log.Println(err)
        return err
    }
    return err
}

func BoardsSocket(w http.ResponseWriter, r *http.Request) {
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
            // Add a board
            // Reuse some request fields for a different purpose
            case "Save":
                name := req.Player
                exists := false
                for _, b := range GetBoards() {
                    if b == name {
                        exists = true
                        continue
                    }
                }
                var msg string
                if exists {
                    msg = "Board " + name + " already exists"
                    log.Println(msg)
                } else {
                    err = AddBoard(name, req.Payload)
                    if err != nil {
                        log.Println(err)
                        msg = err.Error()
                    } else {
                        msg = "Success"
                    }
                }
                reply := Request{Action: "Save", Payload: msg}
                jsn, _ := json.Marshal(reply)
                err = conn.WriteMessage(websocket.TextMessage, jsn)
                if err != nil {
                    log.Println(err)
                    continue
                }
            // List boards
            case "List":
                boards := GetBoards()
                jsn, _ := json.Marshal(boards)
                reply := Request{Action: "List", Payload: string(jsn)}
                jsn, _ = json.Marshal(reply)
                err = conn.WriteMessage(websocket.TextMessage, jsn)
                if err != nil {
                    log.Println(err)
                    continue
                }
            // Load a board plan
            case "Load":
                name := req.Player
                board, err := GetBoard(name)
                if err != nil {
                    log.Println(err)
                    continue
                }
                reply := Request{Action: "Load", Payload: board}
                jsn, _ := json.Marshal(reply)
                err = conn.WriteMessage(websocket.TextMessage, jsn)
                if err != nil {
                    log.Println(err)
                    continue
                }
        }
    }
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

func GameLoop(game *Game, recvChan chan bool, sendChan chan bool) {
    getLastMove := func(prev *ai.Board, cur *ai.Board) int {
        for i := 0; i < len(prev.Points); i++ {
            if cur.Points[i] != -1 && prev.Points[i] != cur.Points[i] {
                return i
            }
        }
        return -1
    }
    board := game.History[0]
    for {
        sendChan <- true
        if board.GameOver(game.History) {
            log.Println("game over")
            log.Println(board.GetScores())
            break
        }
        keepPlaying := <- recvChan
        if !keepPlaying {
            sendChan <- false
            break
        }
        prev := game.History[len(game.History) - 2]
        board = game.History[len(game.History) - 1]
        aimove := AIMove{Point: getLastMove(prev, board)}
        game.Mutex.Lock()
        player := "white"
        if board.Turn % 2 == 1 {
            player = "black"
        }
        jsn2, _ := json.Marshal(aimove)
        reply := Request{Action: "Move-AI", Key: game.Key, Payload: string(jsn2), Player: player}
        jsn, _ := json.Marshal(reply)
        game.Conns[0].WriteMessage(websocket.TextMessage, jsn)
        game.Mutex.Unlock()
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
            case "Concede":
                game := games[req.Key]
                if game == nil {
                    log.Println("Game not found")
                    continue    
                }
                // AI game
                if game.Json == "" {
                    game.RecvChan <- false
                }
                winner := "White"
                if player == 1 {
                    winner = "Black"
                }
                for _, conn := range game.Conns {
                    // AI Game
                    if conn == nil {
                        continue
                    }
                    reply := Request{Action: "Concede", Key: game.Key, Payload: winner}
                    jsn, _ := json.Marshal(reply)
                    conn.WriteMessage(websocket.TextMessage, jsn)
                }
            case "Pass":
                game := games[req.Key]
                if game == nil {
                    log.Println("Game not found")
                    continue    
                }
                next, passing := "white", "Black";
                if player == 1 {
                    next, passing = "black", "White"
                }
                game.Player = next
                for _, conn := range game.Conns {
                    reply := Request{Action: "Pass", Key: game.Key, Player: next, Payload: passing}
                    jsn, _ := json.Marshal(reply)
                    conn.WriteMessage(websocket.TextMessage, jsn)
                }
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
                game := &Game{Key: NextGameIdx(), BoardPlan: req.BoardPlan, Json: req.Payload, Conns: make([]*websocket.Conn, 1), Player: "black"}
                game.Conns[0] = conn
                games[game.Key] = game
                reply := Request{Action: "New", Key: game.Key} 
                jsn, _ := json.Marshal(reply)
                conn.WriteMessage(websocket.TextMessage, jsn)
            case "New-AI":
                player = 0
                // Two cons, one nil to prevent people from joining
                game := &Game{Key: NextGameIdx(), BoardPlan: req.BoardPlan, Json: "", Conns: make([]*websocket.Conn, 2), Player: "black"}
                game.Conns[0] = conn
                games[game.Key] = game
                reply := Request{Action: "New", Key: game.Key} 
                jsn, _ := json.Marshal(reply)
                conn.WriteMessage(websocket.TextMessage, jsn)
                var pn PointsNeighbors 
                err := json.Unmarshal([]byte(req.Payload), &pn)
                if err != nil {
                    log.Println(err)
                    continue
                }
                // Start ai 
                board := pn.ToBoard()
                game.History = []*ai.Board{board}
                recvChan := make(chan bool)
                sendChan := make(chan bool)
                game.RecvChan = recvChan
                for i := 0; i < 2; i++ {
                    if i == player {
                        continue
                    }
                    go ai.Loop(i, &game.History, sendChan, recvChan, 5, 2000, 200)
                }
                go GameLoop(game, recvChan, sendChan)
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
                reply := Request{Action: "Join", Key: game.Key, Payload: game.Json, BoardPlan: game.BoardPlan, Player: game.Player}
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
            case "Move-AI":
                game := games[req.Key]
                game.Mutex.Lock()
                var aimove AIMove
                err := json.Unmarshal([]byte(req.Payload), &aimove)
                if err != nil {
                    log.Println(err)
                    continue
                }
                if player == 0 {
                    game.Player = "white"
                } else {
                    game.Player = "black"
                }
                // Make the move
                nextBoard := game.History[len(game.History)-1].Clone()
                nextBoard.Turn += 1
                if aimove.Point != -1 {
                    nextBoard.Points[aimove.Point] = player
                    nextBoard.CullCaptured(1-player)
                    nextBoard.CullCaptured(player)
                }
                game.History = append(game.History, nextBoard)
                game.RecvChan <- true
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
    http.HandleFunc("/boards", BoardsSocket)
    log.Fatal(http.ListenAndServe(":8001", nil))
}
