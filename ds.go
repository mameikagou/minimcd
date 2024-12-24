package main

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

type LinearDS[T any] interface {
	Push(T)
	Pop() T
	Peek() T
	IsEmpty() bool
	Length() int
}

// this DS does not provide thread security
type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0),
	}
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}
func (s *Stack[T]) Pop() T {
	l := len(s.items)
	if l == 0 {
		panic("Stack: pop")
	}
	ret := s.items[l-1]
	s.items = s.items[:l-1]
	return ret
}
func (s Stack[T]) Peek() T {
	l := len(s.items)
	if l == 0 {
		panic("Stack: peek")
	}
	return s.items[l]
}
func (s Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}
func (s Stack[T]) Length() int {
	return len(s.items)
}

type op int

const (
	ADD op = iota
	DELETE
)

type modify_t[T constraints.Ordered] struct {
	op op
	id int
	ch chan T
}
type DynamicMultiChan[T constraints.Ordered] struct {
	TX chan T
	RX chan T
	// used        map[int]bool
	// list        []chan T
	// reloadTX    chan struct{}
	// reloadRX    chan struct{}
	// selectCases []reflect.SelectCase
	reply bool
	// delta       *Stack[int]
	modify chan modify_t[T]
}

func NewDynamicMultiChan[T constraints.Ordered](reply bool, m int) *DynamicMultiChan[T] {
	ret := &DynamicMultiChan[T]{
		TX: make(chan T),
		RX: make(chan T),

		reply:  reply,
		modify: make(chan modify_t[T]),
	}
	mode := m
	reloadTX := make(chan struct{})
	reloadRX := make(chan struct{})
	used := make(map[int]bool)
	unused := make(map[int]bool)
	delta := -1
	list := make([]chan T, 0)
	selectCases := make([]reflect.SelectCase, 1)
	go func() {
		for {
			modification, _ := <-ret.modify
			op := modification.op
			id := modification.id
			ch := modification.ch
			reloadTX <- struct{}{}
			if mode == 2 {
				reloadTX <- struct{}{}
			}
			<-reloadRX // ok it's safe to do operation
			if mode == 2 {
				<-reloadRX
			}
			switch op {
			case ADD:
				nelem := reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(ch),
					Send: reflect.Value{},
				}
				if len(unused) == 0 {
					delta = len(list)
					used[len(list)] = true
					list = append(list, ch)
					selectCases = append(selectCases, nelem)
				} else {
					var which int
					for x := range unused {
						which = x
						break
					}
					delta = which
					used[which] = true
					delete(unused, which)
					list[which] = ch
					selectCases[which+1] = nelem
				}
			case DELETE:
				delete(used, id)
				unused[id] = true
				list[id] = nil
				selectCases[id+1] = reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(NullChan),
					Send: reflect.Value{},
				}
				//deleted <- struct{}{} // <-reloadTX //extra communication
			}
			reloadTX <- struct{}{}
			if mode == 2 {
				reloadTX <- struct{}{} //I'm finished
			}
			<-reloadRX
			if mode == 2 {
				<-reloadRX //ok I'll cleanup
			}
			delta = -1
		}
	}()
	go func() {
		selectCases[0] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(reloadTX),
		}
		prevId := -1
		for {
			id, recv, ok := reflect.Select(selectCases)
			if prevId != -1 {
				selectCases[prevId].Dir = reflect.SelectRecv
				selectCases[prevId].Send = reflect.Value{}
				prevId = -1
			}
			if !ok {
				// ret.used.Push(id)
				//
				// delete(used, id)
				ret.modify <- modify_t[T]{DELETE, id - 1, nil}
				<-reloadTX
				reloadRX <- struct{}{} //I'm currently not dealing with other chan
				<-reloadTX             //waiting for you finished
				reloadRX <- struct{}{} //I've read all deltas, you can release them
				continue
			}
			if mode == 1 && id != 0 {
				ret.RX <- To[T](recv) // possible for below to send to Chan?
				if ret.reply {
					msg, _ := <-ret.TX
					selectCases[id].Dir = reflect.SelectSend
					selectCases[id].Send = reflect.ValueOf(msg)
					prevId = id
				}
			} else if id == 0 {
				reloadRX <- struct{}{} //I'm currently not dealing with other chan
				<-reloadTX             //waiting for you finished
				reloadRX <- struct{}{} //I've read all deltas, you can release them
			}
		}
	}()
	if mode == 2 {
		go func() {
			msgList := make([]T, 0)
			for {
				select {
				case <-reloadTX:
					reloadRX <- struct{}{} //I'm currently not dealing with other chan
					<-reloadTX             //waiting for you finished
					if delta != -1 {
						for _, x := range msgList {
							list[delta] <- x
						}
					}
					reloadRX <- struct{}{} //I've read all deltas, you can release them
				case msg, _ := <-ret.TX:
					for id, ch := range list {
						if used[id] {
							ch <- msg
						}
					}
					msgList = append(msgList, msg)
				}
			}
		}()
	}
	return ret
}
func (self DynamicMultiChan[T]) IsReply() bool {
	return self.reply
}
func (self *DynamicMultiChan[T]) Add(ch chan T) {
	self.modify <- modify_t[T]{ADD, -1, ch}
}

// TODO: we need to wait for 2 msg and send 4 msg in chan
