package gtasks

import (
	"fmt"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	i := 0
	t1 := &Task{
		f: func(chan bool) {
			i++
		},
	}

	t1.Run()

	if i != 1 {
		t.Fatal("i should be 1")
	}
}

func TestNow(t *testing.T) {
	i := 0
	tick := time.Tick(time.Second * 10)
	t1 := &Task{
		f: func(chan bool) {
			i++
		},
	}

	t1.Once()
	t1.After(tick)
	t1.Now()
	t1.Run()

	if i != 1 {
		t.Fatal("i should be 1")
	}
}

func TestCancelTask(t *testing.T) {
	i := 0

	r := New()

	r.Add("t1", func(c chan bool) {
		not := time.After(time.Second * 5)
		select {
		case _, open := <-c:
			if open == false {
				return
			}
		case <-not:
			i++
		}
	}).Once()

	r.Run()
	r.Cancel("t1")

	time.Sleep(time.Millisecond * 50)

	if i != 0 {
		t.Fatal("i should be 0")
	}
}

func TestSubscribeTask(t *testing.T) {
	i := 0

	t1 := NewTask(func(chan bool) {
		i++
	})

	t2 := NewTask(func(chan bool) {
		i++
	})

	t2.After(t1.Subscribe())
	t2.Once()
	t1.Run()
	t2.Run()

	if i != 2 {
		t.Fatal("i should be 2")
	}
}

func TestRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	i := 0

	r := New()

	r.Add("t1", func(chan bool) {
		i++
	}).Once()

	r.Add("t2", func(chan bool) {
		i++
	}).Once()

	r.Run()

	time.Sleep(time.Millisecond * 200) // sleep for a short duration

	if i != 2 {
		t.Fatal("i should be 2")
	}
}

func ExampleTaskAfter() {
	tick := time.After(time.Millisecond * 100)

	timedTask := NewTask(func(chan bool) {
		fmt.Println("ran task1")
	})

	timedTask.After(tick)
	timedTask.Once()
	timedTask.Run()

	// Output:
	// ran task1
}
