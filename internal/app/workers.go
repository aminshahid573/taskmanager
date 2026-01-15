package app

import (
	"context"
	"sync"

	"github.com/aminshahid573/taskmanager/internal/worker"
)

// WorkerGroup tracks background worker lifecycle.
type WorkerGroup struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	WG     *sync.WaitGroup
}

// StartWorkers starts all background workers and returns a WorkerGroup
// that can be used to coordinate their shutdown.
func StartWorkers(
	parentCtx context.Context,
	emailWorker *worker.EmailWorker,
	reminderWorker *worker.ReminderWorker,
) *WorkerGroup {
	workerCtx, workerCancel := context.WithCancel(parentCtx)

	var wg sync.WaitGroup

	// Start email worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		emailWorker.Start(workerCtx)
	}()

	// Start reminder worker (cron)
	wg.Add(1)
	go func() {
		defer wg.Done()
		reminderWorker.Start(workerCtx)
	}()

	return &WorkerGroup{
		Ctx:    workerCtx,
		Cancel: workerCancel,
		WG:     &wg,
	}
}

