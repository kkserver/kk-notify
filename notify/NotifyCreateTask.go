package notify

import (
	"github.com/kkserver/kk-lib/kk/app"
)

type NotifyCreateTaskResult struct {
	app.Result
	Notify *Notify `json:"notify,omitempty"`
}

type NotifyCreateTask struct {
	app.Task
	Url      string `json:"url"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	MaxCount int    `json:"maxCount"`
	Expires  int64  `json:"expires"`
	Result   NotifyCreateTaskResult
}

func (task *NotifyCreateTask) GetResult() interface{} {
	return &task.Result
}

func (task *NotifyCreateTask) GetInhertType() string {
	return "notify"
}

func (task *NotifyCreateTask) GetClientName() string {
	return "Notify.Create"
}
