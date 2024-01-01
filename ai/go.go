package ai

type Board struct {
    Points []int
    Neighbors [][]int
    NPlayers int
    Turn int
}

func Includes[T comparable](s []T, e T) bool {
    for _, v := range s {
        if v == e {
            return true
        }
    }
    return false
}

func Equals[T comparable](s1 []T, s2 []T) bool {
    if len(s1) != len(s2) {
        return false
    }
    for i := range s1 {
        if s1[i] != s2[i] {
            return false
        }
    }
    return true
}

func InHistory(history []*Board, board *Board) bool {
    for _, b := range history {
        if board.Equals(b) {
            return true
        }
    }
    return false
}

func MakeTraditional(n int, nPlayers int) *Board {
    rc2p := func(r, c int) int {
        return r * n + c
    }
    points := make([]int, n * n)
    neighbors := make([][]int, n * n)
    for r := 0; r < n; r++ {
        for c := 0; c < n; c++ {
            points[rc2p(r, c)] = -1
            ns := []int{}
            if r > 0 {
                ns = append(ns, rc2p(r - 1, c))
            }
            if r < n - 1 {
                ns = append(ns, rc2p(r + 1, c))
            }
            if c > 0 {
                ns = append(ns, rc2p(r, c - 1))
            }
            if c < n - 1 {
                ns = append(ns, rc2p(r, c + 1))
            }
            neighbors[rc2p(r, c)] = ns
        }
    }
    return &Board{
        Points: points,
        Neighbors: neighbors,
        NPlayers: nPlayers,
        Turn: 0,
    }
}

func (board *Board) Clone() *Board {
    points := make([]int, len(board.Points))
    copy(points, board.Points)
    return &Board{
        Points: points,
        Neighbors: board.Neighbors,
        NPlayers: board.NPlayers,
        Turn: board.Turn,
    }
}

func (board *Board) Equals(board2 *Board) bool {
    for i := range board.Points {
        if board.Points[i] != board2.Points[i] {
            return false
        }
    }
    return true
}

func (board *Board) CullCaptured(me int) {
    visited := make(map[int]bool)
    cull := func(p int) {
        frontier := []int{p}
        region := make(map[int]bool)
        foundEmpty := false
        for len(frontier) > 0 {
            p := frontier[0]
            frontier = frontier[1:]
            ns := board.Neighbors[p]
            for _, n := range ns {
                if Includes(frontier, n) || region[n] {
                    continue
                }
                if board.Points[n] == me || board.Points[n] == -1 {
                    if board.Points[n] == -1 {
                        foundEmpty = true
                    } else {
                        frontier = append(frontier, n)
                    }
                }
            }
            visited[p] = true
            region[p] = true
        }
        if !foundEmpty {
            for rp := range region {
                board.Points[rp] = -1
            }
        }
    }
    for p, player := range board.Points {
        if !visited[p] && player == me {
            cull(p)
        }
    }
}

// One for each island
func (board *Board) GetEyes(me int) []int {
    return nil
}

// One for each island
func (board *Board) GetLiberties(me int) []int {
    libs := make([]int, 0)
    visited := make(map[int]bool)
    expand := func(p int) int {
        lib := 0
        frontier := []int{p}
        for len(frontier) > 0 {
            p := frontier[0]
            visited[p] = true
            frontier = frontier[1:]
            ns := board.Neighbors[p]
            for _, n := range ns {
                if Includes(frontier, n) || visited[n] {
                    continue
                }
                if board.Points[n] == me {
                    frontier = append(frontier, n)
                } else if board.Points[n] == -1 {
                    lib += 1
                }
            }
        }
        return lib
    }
    for p, player := range board.Points {
        if !visited[p] && player == me {
            libs = append(libs, expand(p))
        }
    }
    return libs
}

func (board *Board) GetScores() []int {
    visited := make(map[int]bool)
    expandGetEmptyScore := func(p int) (bool, int, int) {
        frontier := []int{p}
        region := make(map[int]bool)
        me := -1
        contested := false
        for len(frontier) > 0 {
            p := frontier[0]
            frontier = frontier[1:]
            ns := board.Neighbors[p]
            for _, n := range ns {
                if Includes(frontier, n) || region[n] {
                    continue;
                }
                player := board.Points[n]
                if player != -1 {
                    if me == -1 {
                        me = player
                    } else if me != player {
                        contested = true
                    }
                } else {
                    frontier = append(frontier, n)
                }
            }
            visited[p] = true
            region[p] = true
        }
        return contested, me, len(region)
    }
    scores := make([]int, board.NPlayers)
    for p, player := range board.Points {
        if visited[p] {
            continue
        }
        if player == -1 {
            contested, player, size := expandGetEmptyScore(p)
            if !contested && player != -1 {
                scores[player] += size
            } 
        } else {
            scores[player] += 1
        }
    } 
    return scores
}

type Stats struct {
    Scores []int
    Libs [][]int
    LibDangers []float64
}

func (board *Board) GetStats() *Stats {
    stats := &Stats{}
    stats.Scores = board.GetScores()
    stats.Libs = make([][]int, 0)
    stats.LibDangers = make([]float64, 0)
    for i := 0; i < board.NPlayers; i++ {
        libs := board.GetLiberties(i)
        stats.Libs = append(stats.Libs, libs)
        danger := 0.0
        for _, lib := range libs {
            switch lib {
                case 1: danger += 2.0
                case 2: danger += 0.75
                case 3: danger += 0.25
            }
        }
        stats.LibDangers = append(stats.LibDangers, danger)
    }
    return stats
}

// 1. Maximize your score
// 2. Minimize your opponent's score
// 3. Maximize your liberties
// 4. Minimize your opponent's liberties
func (board *Board) Eval(before *Stats, me int) float64 {
    after := board.GetStats()
    a := float64(after.Scores[me] - before.Scores[me] + before.Scores[1-me] - after.Scores[1-me])
    b := before.LibDangers[me] - after.LibDangers[me] + after.LibDangers[1-me] - before.LibDangers[1-me]
    return a+b
}

func (board *Board) GameOver(history []*Board) bool {
    if len(history) <= board.NPlayers {
        return false
    }
    // N passes in a row
    for i := len(history) - board.NPlayers; i < len(history); i++ {
        if !history[i].Equals(board) {
            return false
        }
    }
    return true
}

func (board *Board) GetCandidates(history []*Board, me int) []func() *Board {
    cand := make([]func() *Board, 0)
    if board.Turn % board.NPlayers != me {
        return cand
    }
    b := board.Clone()
    b.Turn += 1
    // Pass
    cand = append(cand, func() *Board {
        return b
    })
    for p, player := range board.Points {
        if player == -1 {
            b := board.Clone()
            b.Points[p] = me
            // Order matters
            b.CullCaptured(1-me)
            b.CullCaptured(me)
            if InHistory(history, b) {
                continue
            }
            b.Turn += 1
            cand = append(cand, func() *Board {
                return b
            })
        }
    }
    return cand
}

func AddToHistory(history []*Board, board *Board) []*Board {
    nHist := make([]*Board, len(history)+1)
    copy(nHist, history)
    nHist[len(history)] = board
    return nHist
}
