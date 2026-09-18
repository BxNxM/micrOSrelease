package usb

import "context"

type stageTracker struct {
	stages    []Stage
	observers []func([]Stage)
}

func newStageTracker(names []string, observers []func([]Stage)) *stageTracker {
	tracker := &stageTracker{stages: make([]Stage, len(names)), observers: observers}
	for index, name := range names {
		tracker.stages[index] = Stage{Name: name, State: StageWaiting}
	}
	tracker.emit()
	return tracker
}

func (tracker *stageTracker) run(ctx context.Context, index int, operation func() error) error {
	tracker.stages[index].State = StageRunning
	tracker.emit()
	if err := ctx.Err(); err != nil {
		tracker.stages[index].State = StageFailed
		tracker.emit()
		return err
	}
	if err := operation(); err != nil {
		tracker.stages[index].State = StageFailed
		tracker.emit()
		return err
	}
	tracker.stages[index].CanReconnect = false
	tracker.stages[index].State = StageDone
	tracker.emit()
	return nil
}

func (tracker *stageTracker) emit() {
	for _, observer := range tracker.observers {
		if observer != nil {
			observer(append([]Stage(nil), tracker.stages...))
		}
	}
}

func (tracker *stageTracker) skip(index int) {
	tracker.stages[index].State = StageSkipped
	tracker.emit()
}

func (tracker *stageTracker) reconnecting(index int, detail string) {
	tracker.stages[index].CanReconnect = true
	tracker.detail(index, detail)
}

func (tracker *stageTracker) detail(index int, detail string) {
	tracker.stages[index].Detail = detail
	tracker.emit()
}
