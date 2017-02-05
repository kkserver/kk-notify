package notify

import (
	"github.com/kkserver/kk-lib/kk/app"
)

type NotifyTaskResult struct {
	app.Result
	Notify *Notify `json:"notify,omitempty"`
}

type NotifyTask struct {
	app.Task
	Id     int64  `json:"id"`
	Code   string `json:"code"`
	Result NotifyTaskResult
}

func (task *NotifyTask) GetResult() interface{} {
	return &task.Result
}

func (task *NotifyTask) GetInhertType() string {
	return "notify"
}

func (task *NotifyTask) GetClientName() string {
	return "Notify.Get"
}
