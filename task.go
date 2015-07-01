package gtasks

import (
	"reflect"
	"sync"
	"time"
)

// New returns a new *runner to register tasks
func New() *Runner {
	return &Runner{
		tasks: make(map[string]*Task),
		mu:    sync.RWMutex{},
	}
}

// Runner holds tasks that should be executed
type Runner struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// Run will start the tasks
func (r *Runner) Run() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.tasks {
		go task.Run()
	}
}

// Cancel stops a single task from executing: see Task.Cancel
func (r *Runner) Cancel(name string) {
	r.mu.Lock()
	if t, ok := r.tasks[name]; ok {
		t.Cancel()
		delete(r.tasks, name)
	}
	r.mu.Unlock()
}

// CancelAll stops all tasks from executing: see Task.Cancel
func (r *Runner) CancelAll() {
	r.mu.Lock()
	for name, t := range r.tasks {
		t.Cancel()
		delete(r.tasks, name)
	}
	r.mu.Unlock()
}

// Get returns a task by name
func (r *Runner) Get(name string) *Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.tasks[name]; ok {
		return t
	}
	return nil
}

// All returns all task
func (r *Runner) All() map[string]*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tasks
}

// Add adds a new task to a runner
func (r *Runner) Add(name string, f func(chan bool)) *Task {
	t := NewTask(f)
	r.mu.Lock()
	r.tasks[name] = t
	r.mu.Unlock()
	return t
}

// NewTask creates a new taks an inits the needed variables
func NewTask(f func(chan bool)) *Task {
	t := &Task{
		f:          f,
		cancelchan: make(chan bool),
		listeners:  make([]chan interface{}, 0),
	}
	return t
}

// Task represents a task that can be run.
// It contains a at least runnable function
// that accepts a chan bool which will be closed
// if the task is canceled
type Task struct {
	f          func(chan bool)
	cancelchan chan bool
	after      chan interface{}
	listeners  []chan interface{}
	once       bool
	runStart   time.Time
	running    bool
}

// Cancel stops a task. Executing functions
// should monitor if the cancelchan is closed
func (t *Task) Cancel() {
	close(t.cancelchan)
}

// After sets the channel which will start the task.
// If it is not set the task will run immediately
// when calling Run and return after that
func (t *Task) After(c interface{}) *Task {
	t.after = Wrap(c)
	return t
}

// Now will send a message over the After channel
// to start the task
func (t *Task) Now() *Task {
	if t.after != nil {
		go func() {
			t.after <- struct{}{}
		}()
	}

	return t
}

// Once ensures a task is only ran once
func (t *Task) Once() *Task {
	t.once = true
	return t
}

// Subscribe returns a channel that can be used in the After func.
// This way tasks can be depedent on each other.
func (t *Task) Subscribe() chan interface{} {
	c := make(chan interface{}, 1) // always make room for 1 item to be non-blocking
	t.listeners = append(t.listeners, c)
	return c
}

// Run will run a task
func (t *Task) Run() {
	if t.after == nil {
		t.exec()
		return
	}

	for {
		select {
		case _, opened := <-t.cancelchan:
			if opened == false {
				return
			}
		case <-t.after:
			t.exec()
			if t.once {
				return
			}
		}
	}
}

func (t *Task) exec() {
	t.running = true
	t.runStart = time.Now()
	t.f(t.cancelchan)
	t.running = false
	for _, l := range t.listeners {
		select {
		case l <- true:
		default:
		}
	}
}

// Wrap is copied from https://github.com/eapache/channels
// Wrap takes any readable channel type (chan or <-chan but not chan<-) and
// exposes it as a SimpleOutChannel for easy integration with existing channel sources.
// It panics if the input is not a readable channel.
func Wrap(ch interface{}) chan interface{} {
	t := reflect.TypeOf(ch)
	if t.Kind() != reflect.Chan || t.ChanDir()&reflect.RecvDir == 0 {
		panic("channels: input to Wrap must be readable channel")

	}

	realChan := make(chan interface{}) // buffer chan

	if t.Elem().Kind() == reflect.Interface {
		return ch.(chan interface{})
	}
	go func() {
		v := reflect.ValueOf(ch)
		for {
			x, ok := v.Recv()
			if !ok {
				close(realChan)
				return

			}
			select {
			case realChan <- x.Interface():
			default:
			}

		}

	}()

	return realChan
}
