package saku

import (
	"github.com/jinzhu/copier"
)

// TaskCollection is the model of the list of tasks.
type TaskCollection struct {
	tasks   []*task
	taskMap map[string]*task
	mode    RunMode
}

// Creates a new task collection.
func newTaskCollection() *TaskCollection {
	return &TaskCollection{
		tasks:   []*task{},
		taskMap: map[string]*task{},
		mode:    RunModeSequence,
	}
}

// SetRunMode sets the run mode of the collection.
func (tc *TaskCollection) SetRunMode(mode RunMode) {
	tc.mode = mode
}

// Run runs the tasks.
func (tc *TaskCollection) Run(ctx *runContext, stack *taskStack) error {
	var err error
	ctx.l.logStart(tc, stack)

	switch tc.mode {
	case RunModeParallel:
		err = tc.runParallel(ctx, stack)
	case RunModeParallelRace:
		err = tc.runInRace(ctx, stack)
	default:
		err = tc.runSequentially(ctx, stack)
	}

	if err != nil {
		return err
	}

	ctx.l.logEnd(tc, stack)

	return nil
}

func (tc *TaskCollection) runSequentially(ctx *runContext, stack *taskStack) error {
	c := make(chan error)

	for _, t := range tc.tasks {
		go t.run(ctx, c, stack)

		err := <-c

		if err != nil {
			return err
		}
	}

	return nil
}

func (tc *TaskCollection) runParallel(ctx *runContext, stack *taskStack) error {
	c := make(chan error)

	for i := range tc.tasks {
		t := tc.tasks[i]
		go t.run(ctx, c, stack)
	}

	for range tc.tasks {
		err := <-c

		if err != nil {
			tc.abort()
			return err
		}
	}

	return nil
}

func (tc *TaskCollection) runInRace(ctx *runContext, stack *taskStack) error {
	c := make(chan error)

	for i := range tc.tasks {
		go tc.tasks[i].run(ctx, c, stack)
	}

	defer tc.abort()

	return <-c
}

func (tc *TaskCollection) abort() {
	for _, t := range tc.tasks {
		t.abort()
	}
}

// appendNewTask appends a new task of the given level.
func (tc *TaskCollection) appendNewTask(level int, title string) *task {
	t := newTask(level)
	t.setTitle(title)
	tc.tasks = append(tc.tasks, t)
	tc.taskMap[title] = t
	return t
}

func (tc *TaskCollection) gotNewTask(level int, title string) *task {
	if tc.isEmpty() || tc.lastTask().level >= level {
		return tc.appendNewTask(level, title)
	}

	return tc.lastTask().gotNewTask(level, title)
}

func (tc *TaskCollection) isEmpty() bool {
	return len(tc.tasks) == 0
}

func (tc *TaskCollection) lastTask() *task {
	return tc.tasks[len(tc.tasks)-1]
}

func (tc *TaskCollection) findByTitle(title string) *task {
	for _, t := range tc.tasks {
		found := t.findByTitle(title)

		if found != nil {
			return found
		}
	}

	return nil
}

// filterByTitles filters the task by the given titles and returns a new task collection. The tasks in the returned collection is cloned from the original tasks.
func (tc *TaskCollection) filterByTitles(titles []string) *TaskCollection {
	tasks := []*task{}
	taskMap := map[string]*task{}

	for _, title := range titles {
		t0 := newTask(0)
		t1 := tc.findByTitle(title)
		copier.Copy(t0, t1)
		tasks = append(tasks, t0)
		taskMap[title] = t0
	}

	return &TaskCollection{
		tasks:   tasks,
		taskMap: taskMap,
		mode:    tc.mode,
	}
}

func (tc *TaskCollection) titles() []string {
	titles := []string{}

	for _, t := range tc.tasks {
		titles = append(titles, t.title)
	}

	return titles
}

func (tc *TaskCollection) taskCount() int {
	c := 0

	for _, t := range tc.tasks {
		c += t.taskCount()
	}

	return c
}
