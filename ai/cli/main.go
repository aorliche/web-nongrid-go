package main

import (
    "fmt"
    ai "github.com/aorliche/web-nongrid-go/ai"
)

func main() {
    nplay := 2
    board := ai.MakeTraditional(4, nplay)
    history := []*ai.Board{board}
    recvChan := make(chan bool)
    sendChans := make([]chan bool, 0)
    for i := 0; i < nplay; i++ {
        sendChans = append(sendChans, make(chan bool))
        go ai.Loop(i, &history, sendChans[i], recvChan, 10, 1000, 15)
    }
    for {
        fmt.Println("A", board)
        for i := 0; i < nplay; i++ {
            sendChans[i] <- true
        }
        if board.GameOver(history) {
            fmt.Println("D", board)
            fmt.Println(board.GetScores())
            break
        }
        val := <- recvChan
        board = history[len(history) - 1]
        fmt.Println("C", val, board)
    }
}
