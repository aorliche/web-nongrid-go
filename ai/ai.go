package ai

import (
    //"fmt"
    //"sort"
    "time"
)

func Loop(me int, history *[]*Board, inChan chan bool, outChan chan bool, depth int, timeMillis int, nTop int) {
    var board *Board
    for { 
        state := <- inChan 
        board = (*history)[len(*history)-1]
        // false state indicates player disconnect
        // concession or two passes
        if !state || board.GameOver(*history) {
            break
        }
        next := Search(*history, me, depth, timeMillis, nTop)
        if next == nil {
            time.Sleep(100 * time.Millisecond)
            continue
        }
        *history = append(*history, next)
        outChan <- true
    }
}

// Set up iterative deepening
func Search(history []*Board, me int, depth int, timeMillis int, nTop int) *Board {
    startTime := time.Now()
    board := history[len(history)-1]
    stats := board.GetStats()
    var res *Board
    for d := 1; d < depth; d++ {
        _, fn, fin := SearchDeep(stats, history, me, d, startTime, timeMillis, nTop)
        /*if st != nil {
            if d == 1 {
                fmt.Println("board", board)
            }
            fmt.Println("me", me, "d", d, "see", board, "val", board.Eval(stats, me))
        }*/
        if fn != nil && fin {
            res = fn()
        } else {
            break;
        }
    }
    return res
}

// Iterative deepening worker
// Standard minimax without alpha-beta pruning
// Allow players to make consecutive moves
// If the game rules allow it
func SearchDeep(stats *Stats, history []*Board, me int, d int, startTime time.Time, timeMillis int, nTop int) (*Board, func()*Board, bool) {
    if d == 0 {
        return history[len(history)-1], nil, true
    }
    if time.Since(startTime).Milliseconds() > int64(timeMillis) {
        return nil, nil, false
    }
    board := history[len(history)-1]
    fns := board.GetCandidates(history, me)
    if len(fns) == 0 {
        return nil, nil, true
    }
    vals := make([]float64, len(fns))
    boards := make([]*Board, len(fns))
    // Sort fns by heuristic eval
    /*fnVals := make([]float64, len(fns))
    for i,fn := range fns {
        fnVals[i] = fn().Eval(stats, me)
    }
    sort.Slice(fns, func(i, j int) bool {
        return fnVals[i] > fnVals[j]
    })*/
    for i,fn := range fns {
        // Just evaluate the top N heuristic choices
        /*if i >= nTop {
            break
        }*/
        next := fn()
        if !next.GameOver(history) {
            // Check opponents responses 
            // Don't assume that a valid move can be found
            vals2 := make([]float64, board.NPlayers)
            boards2 := make([]*Board, board.NPlayers)
            nHist := AddToHistory(history, next)
            for j := 0; j < board.NPlayers; j++ {
                n, _, fin := SearchDeep(stats, nHist, j, d-1, startTime, timeMillis, nTop)
                if fin {
                    if n != nil {
                        vals2[j] = n.Eval(stats, me)
                        boards2[j] = n
                    }
                } else {
                    return nil, nil, false
                }
            }
            // Min of minimax
            best := -1
            for j := 0; j < len(vals2); j++ {
                if boards2[j] != nil && (best == -1 || vals2[j] < vals2[best]) {
                    best = j
                }
            }
            if best != -1 {
                vals[i] = vals2[best]
                boards[i] = boards2[best]
            }
        } else {
            vals[i] = next.Eval(stats, me)
            boards[i] = next
        }
    }
    best := -1
    for i := 0; i < len(vals); i++ {
        if boards[i] != nil && (best == -1 || vals[i] > vals[best]) {
            best = i
        }
    }
    /*if best == -1 {
        return nil, nil, false
    }*/
    return boards[best], fns[best], true
}
