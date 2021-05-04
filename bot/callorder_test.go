package bot

import (
	"fmt"
	"sync"

	"github.com/golang/mock/gomock"
)

// Helper stuff for enforcing ordering between gomock calls.

// A group of calls, which form a DAG.
type group struct {
	first []*gomock.Call
	last  []*gomock.Call
}

func (g *group) closeOnDone(done chan<- struct{}) {
	var wg sync.WaitGroup
	for _, call := range g.last {
		wg.Add(1)
		call.Do(func(...interface{}) { wg.Done() })
	}
	go func() {
		wg.Wait()
		close(done)
	}()
}

func asGroup(callOrGroup interface{}) *group {
	if call, ok := callOrGroup.(*gomock.Call); ok {
		return &group{
			first: []*gomock.Call{call},
			last:  []*gomock.Call{call},
		}
	}
	if g, ok := callOrGroup.(*group); ok {
		return g
	}
	panic(fmt.Errorf("got %T, want *gomock.Call or *group", callOrGroup))
}

// A checkpoint after a group.
type checkpoint chan struct{}

func newCheckpoints() (checkpoint, checkpoint, checkpoint, checkpoint, checkpoint) {
	return make(chan struct{}),
		make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
}

// Enforces no ordering between the input calls or groups. Returns a combined group.
func anyOrder(callsOrGroups ...interface{}) *group {
	res := &group{}
	for _, x := range callsOrGroups {
		g := asGroup(x)
		res.first = append(res.first, g.first...)
		res.last = append(res.last, g.last...)
	}
	return res
}

// Makes calls in each group require calls in the previous group to have executed. Returns a
// combined group. Checkpoints are triggered when the previous group has executed.
func inOrder(callsOrGroupsOrCheckpoints ...interface{}) *group {
	res := &group{}
	lastGroup := &group{}
	for i, x := range callsOrGroupsOrCheckpoints {
		if chk, ok := x.(checkpoint); ok {
			if i == 0 {
				// Can probably be made to work in this case but I don't need it
				panic(fmt.Errorf("a checkpoint cannot be first"))
			}
			lastGroup.closeOnDone(chk)
			continue
		}
		g := asGroup(x)
		if i == 0 {
			res.first = g.first
		}
		for _, lastCall := range lastGroup.last {
			for _, firstCall := range g.first {
				firstCall.After(lastCall)
			}
		}
		lastGroup = g
	}
	res.last = lastGroup.last
	return res
}
