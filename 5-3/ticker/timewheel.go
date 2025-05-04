package timewheel

import (
	"sync"
	"time"
)

type Task struct {
	ID      string
	Delay 	time.Duration
	Rounds  int
	Execute func()
}

// using transport task into channel
type TaskRequest struct {
    Delay time.Duration
    Task  *Task
}

type TimeWheel struct {
	tick       time.Duration
    wheelSize  int
    slots      []map[string]*Task
    currentPos int
    mu         sync.Mutex

    taskQueue chan *TaskRequest
    cancelMap map[string]int // taskID -> slot

    stopChan chan struct{}
    wg       sync.WaitGroup

    lowerWheel *TimeWheel
}


func NewTimeWheel(tick time.Duration, wheelSize int, lower *TimeWheel) *TimeWheel {
	slots := make([]map[string]*Task, wheelSize);
	for i := range slots {
		slots[i] = make(map[string]*Task);
	}

	return &TimeWheel{
		tick: 		tick,
		slots: 		slots,
		wheelSize: 	wheelSize,
		currentPos: 0,
		taskQueue: 	make(chan *TaskRequest, 1024),
		cancelMap: 	make(map[string]int),
		stopChan: 	make(chan struct{}),
		lowerWheel: lower,
	}
}


func (tw *TimeWheel) AddTask(delay time.Duration, task *Task) {
	tw.taskQueue <- &TaskRequest{
		delay,
		task,
	}
}


func (tw *TimeWheel) Start() {
	tw.wg.Add(1);
	go func() {
		defer tw.wg.Done();
		for {
			select {
			case <-tw.stopChan:
				return;
			default:
				tw.tickHandler();
				time.Sleep(tw.tick);
			}
		}
	}();

	tw.wg.Add(1);
	go func() {
		defer tw.wg.Done();
		for {
			select {
			case <- tw.stopChan:
				return;
			case req := <-tw.taskQueue:
				tw.addTaskInternal(req.Delay, req.Task);
			}
		}
	}();

	if tw.lowerWheel != nil {
		tw.lowerWheel.Start();
	}
}

func (tw *TimeWheel) addTaskInternal(delay time.Duration, task *Task) {
	if delay < tw.tick && tw.lowerWheel != nil {
		tw.lowerWheel.AddTask(delay, task);
		return;
	}

	tw.mu.Lock();
	defer tw.mu.Unlock();

	ticks := int(delay / tw.tick);
	round := ticks / tw.wheelSize;
	pos := (tw.currentPos + ticks) % tw.wheelSize;

	task.Rounds = round;
	tw.slots[pos][task.ID] = task;
	tw.cancelMap[task.ID] = pos;

}

func (tw *TimeWheel) tickHandler() {
	tw.mu.Lock();
	tasks := tw.slots[tw.currentPos];
	for id, task := range tasks {
		if task.Rounds > 0 {
			task.Rounds--;
		} else {
			go task.Execute();
			delete(tw.slots[tw.currentPos], id);
			delete(tw.cancelMap, id);
		}
	}
}

func (tw *TimeWheel) RemoveTask(taskID string) {
	tw.mu.Lock();
	defer tw.mu.Unlock();
	if slot, ok := tw.cancelMap[taskID]; ok {
		delete(tw.slots[slot], taskID);
		delete(tw.cancelMap, taskID)
	}
}

func (tw *TimeWheel) Stop() {
	close(tw.stopChan);
	tw.wg.Wait();
	if tw.lowerWheel != nil {
		tw.lowerWheel.Stop();
	}
}