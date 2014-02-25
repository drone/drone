package queue

import (
	"github.com/drone/drone/pkg/build/script"
	. "github.com/drone/drone/pkg/model"
	"runtime"
)

// A Queue dispatches tasks to workers.
type Queue struct {
	tasks chan<- *BuildTask
}

// BuildTasks represents a build that is pending
// execution.
type BuildTask struct {
	Repo   *Repo
	Commit *Commit
	Build  *Build

	// Build instructions from the .drone.yml
	// file, unmarshalled.
	Script *script.Build
}

var defaultQueue = Start(runtime.NumCPU()) // TEMPORARY; INJECT PLEASE

var Add = defaultQueue.Add // TEMPORARY; INJECT PLEASE

func Start(workers int) *Queue {
	// get the number of CPUs. Since builds
	// tend to be CPU-intensive we should only
	// execute 1 build per CPU.
	// must be at least 1
	// if ncpu < 1 {
	// 	ncpu = 1
	// }

	tasks := make(chan *BuildTask)

	queue := &Queue{tasks: tasks}

	// spawn a worker for each CPU
	for i := 0; i < workers; i++ {
		worker := worker{}
		go worker.work(tasks)
	}

	return queue
}

// Add adds the task to the build queue.
func (q *Queue) Add(task *BuildTask) {
	q.tasks <- task
}
