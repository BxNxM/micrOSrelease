package usb

type StageState string

const (
	StageWaiting StageState = "waiting"
	StageRunning StageState = "running"
	StageDone    StageState = "done"
	StageSkipped StageState = "skipped"
	StageFailed  StageState = "failed"
)

type Stage struct {
	Name         string
	State        StageState
	Detail       string
	CanReconnect bool
}
