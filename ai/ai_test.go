package ai

import (
    "fmt"
    "testing"
)

func TestMakeTraditional(t *testing.T) {
    board := MakeTraditional(3, 2)
    expect := []int{3, 1}
    got := board.Neighbors[0]
    if len(expect) != len(got) {
        t.Errorf("expect %v got %v", expect, got)
    }
    for i := 0; i < len(expect); i++ {
        if expect[i] != got[i] {
            t.Errorf("expect %v got %v", expect, got)
        }
    }
    expect = []int{1, 7, 3, 5}
    got = board.Neighbors[4]
    for i := 0; i < len(expect); i++ {
        if expect[i] != got[i] {
            t.Errorf("expect %v got %v", expect, got)
        }
    }
}

func TestGetScores(t *testing.T) {
    board := MakeTraditional(3, 2)
    board.Points[0] = 0
    scores := board.GetScores()
    expect := []int{9, 0}
    if !Equals(expect, scores) {
        t.Errorf("expect %v got %v", expect, scores)
    }
    board.Points[1] = 1
    board.Points[3] = 1
    scores = board.GetScores()
    expect = []int{1, 8}
    if !Equals(expect, scores) {
        t.Errorf("expect %v got %v", expect, scores)
    }
    board.Points[8] = 0
    scores = board.GetScores()
    expect = []int{2, 2}
    if !Equals(expect, scores) {
        t.Errorf("expect %v got %v", expect, scores)
    }
}

func TestGetLiberties(t *testing.T) {
    board := MakeTraditional(3, 2)
    got := board.GetLiberties(0)
    expect := []int{}
    if !Equals(expect, got) {
        t.Errorf("expect %v got %v", expect, got)
    }
    board.Points[0] = 0
    got = board.GetLiberties(0)
    expect = []int{2}
    if !Equals(expect, got) {
        t.Errorf("expect %v got %v", expect, got)
    }
    board.Points[2] = 0
    got = board.GetLiberties(0)
    expect = []int{2, 2}
    if !Equals(expect, got) {
        t.Errorf("expect %v got %v", expect, got)
    }
    board.Points[1] = 1
    got = board.GetLiberties(0)
    expect = []int{1, 1}
    if !Equals(expect, got) {
        t.Errorf("expect %v got %v", expect, got)
    }
    got = board.GetLiberties(1)
    expect = []int{1}
    if !Equals(expect, got) {
        t.Errorf("expect %v got %v", expect, got)
    }
}

func TestCullCaptured(t *testing.T) {
    board := MakeTraditional(3, 2)
    board.Points[0] = 0
    board.CullCaptured(0)    
    board.CullCaptured(1)    
    expect := []int{0, -1, -1, -1, -1, -1, -1, -1, -1}
    if !Equals(expect, board.Points) {
        t.Errorf("expect %v got %v", expect, board.Points)
    }
    board.Points[1] = 1
    board.Points[3] = 1
    board.CullCaptured(1)    
    expect = []int{0, 1, -1, 1, -1, -1, -1, -1, -1}
    if !Equals(expect, board.Points) {
        t.Errorf("expect %v got %v", expect, board.Points)
    }
    board.CullCaptured(0)
    expect = []int{-1, 1, -1, 1, -1, -1, -1, -1, -1}
    if !Equals(expect, board.Points) {
        t.Errorf("expect %v got %v", expect, board.Points)
    }
}

func TestGetStats(t *testing.T) {
    board := MakeTraditional(3, 2)
    board.Points[0] = 0
    board.Points[1] = 1
    stats := board.GetStats()
    if len(stats.Libs[0]) != 1 || len(stats.Libs[1]) != 1 {
        t.Errorf("bad lengths")
    }
    if !Equals([]int{1}, stats.Libs[0]) {
        t.Errorf("expected %v got %v", []int{1}, stats.Libs[0])
    }
    if !Equals([]int{2}, stats.Libs[1]) {
        t.Errorf("expected %v got %v", []int{2}, stats.Libs[1])
    }
    if !Equals([]float64{2, 0.75}, stats.LibDangers) {
        t.Errorf("expected %v got %v", []float64{2, 0.75}, stats.LibDangers)
    }
    board.Points[6] = 0
    stats = board.GetStats()
    if !Equals([]int{1, 2}, stats.Libs[0]) {
        t.Errorf("expected %v got %v", []int{1, 2}, stats.Libs[0])
    }
    if !Equals([]float64{2.75, 0.75}, stats.LibDangers) {
        t.Errorf("expected %v got %v", []float64{2.75, 0.75}, stats.LibDangers)
    }
}

func TestMoveOnce(t *testing.T) {
    board := MakeTraditional(3, 2)
    stats := board.GetStats()
    history := []*Board{board}
    fns := board.GetCandidates(history, 0)
    for _, fn := range fns {
        next := fn()
        val := next.Eval(stats, 0)
        fmt.Println(next, val)
    }
}
