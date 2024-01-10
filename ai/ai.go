package ai

import (
    //"fmt"
    "math"
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
    // End early if you win right away
    if len(history) >= 2 && board.Equals(history[len(history)-2]) && stats.Scores[me] > stats.Scores[1-me] {
        res := board.Clone()
        res.Turn += 1
        return res
    }
    var res *Board
    for d := 1; d < depth; d++ {
        //_, fn, fin := SearchDeep(stats, history, me, d, startTime, timeMillis, nTop)
        _, fn, fin, _ := SearchDeepAlphaBeta(stats, history, me, d, math.Inf(-1), math.Inf(1), true, startTime, timeMillis)
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

func max(a float64, b float64) float64 {
    if a > b {
        return a
    }
    return b
}

func min(a float64, b float64) float64 {
    if a < b {
        return a
    }
    return b
}

func SearchDeepAlphaBeta(stats *Stats, history []*Board, me int, depth int, alpha float64, beta float64, maxNotMin bool, startTime time.Time, timeMillis int) (*Board, func()*Board, bool, float64) {
    if depth == 0 {
        return history[len(history)-1], nil, true, history[len(history)-1].Eval(stats, me)
    }
    if time.Since(startTime).Milliseconds() > int64(timeMillis) {
        return nil, nil, false, 0
    }
    board := history[len(history)-1]
    var me2 int
    if maxNotMin {
        me2 = me
    } else {
        me2 = 1-me
    }
    fns := board.GetCandidates(history, me2)
    if len(fns) == 0 {
        var val float64
        if maxNotMin {
            val = math.Inf(-1)
        } else {
            val = math.Inf(1)
        }
        return nil, nil, true, val
    }
    var v float64
    var resBoard *Board
    var resFn func()*Board
    if maxNotMin {
        v = math.Inf(-1)
    } else {
        v = math.Inf(1)
    }
    for _,fn := range fns {
        next := fn()
        if next.GameOver(history) {
            return next, fn, true, next.Eval(stats, me)
        }
        nHist := AddToHistory(history, next)
        n, _, fin, val := SearchDeepAlphaBeta(stats, nHist, me, depth-1, alpha, beta, !maxNotMin, startTime, timeMillis) 
        if !fin {
            return nil, nil, false, 0
        }
        if maxNotMin {
            if val > v {
                v = val
                resBoard = n
                resFn = fn
                alpha = max(alpha, v)
            }
            if v >= beta {
                return resBoard, resFn, true, v
            }
        } else {
            if val < v {
                v = val
                resBoard = n
                resFn = fn
                beta = min(beta, v)
            }
            if v <= alpha {
                return resBoard, resFn, true, v
            }
        }
    }
    return resBoard, resFn, true, v
}

// Iterative deepening worker
// Standard minimax without alpha-beta pruning
// Allow players to make consecutive moves
// If the game rules allow it
/*func SearchDeep(stats *Stats, history []*Board, me int, d int, alpha int, beta int, startTime time.Time, timeMillis int, nTop int) (*Board, func()*Board, bool) {
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
    for i,fn := range fns {
        next := fn()
        if !next.GameOver(history) {
            // Check opponents responses 
            // Don't assume that a valid move can be found
            vals2 := make([]float64, board.NPlayers)
            boards2 := make([]*Board, board.NPlayers)
            nHist := AddToHistory(history, next)
            for j := 0; j < board.NPlayers; j++ {
                n, _, fin := SearchDeep(stats, nHist, j, d-1, alpha, beta, startTime, timeMillis, nTop)
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
    return boards[best], fns[best], true
}*/
