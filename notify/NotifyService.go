package notify

import (
	"crypto/tls"
	"fmt"
	"github.com/kkserver/kk-lib/kk"
	"github.com/kkserver/kk-lib/kk/app"
	"log"
	"net/http"
	"time"
)

type NotifyService struct {
	app.Service

	Get    *NotifyTask
	Create *NotifyCreateTask

	in chan bool
}

func (S *NotifyService) Handle(a app.IApp, task app.ITask) error {
	return app.ServiceReflectHandle(a, task, S)
}

func (S *NotifyService) HandleInitTask(a INotifyApp, task *app.InitTask) error {

	S.in = make(chan bool, 1024)

	go func() {

		for {

			log.Println("NotifyService", "Runloop")

			err := func() error {

				db, err := a.GetDB()

				if err != nil {
					return err
				}

				rows, err := kk.DBQuery(db, a.GetNotifyTable(), a.GetPrefix(), " WHERE status=? ORDER BY id ASC", NotifyStatusNone)

				if err != nil {
					return err
				}

				defer rows.Close()

				v := Notify{}
				scanner := kk.NewDBScaner(&v)

				for rows.Next() {

					err = scanner.Scan(rows)

					if err != nil {
						return err
					}

					if v.Expires > 0 && v.Ctime+v.Expires <= time.Now().Unix() {

						v.Status = NotifyStatusExpires

						_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"status": true})

						continue
					}

					go func(v Notify) {

						err := func() error {

							v.Code = v.NewCode()

							_, err := kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true})

							if err != nil {
								return err
							}

							client := &http.Client{
								Transport: &http.Transport{
									TLSClientConfig: &tls.Config{RootCAs: a.GetCA()},
								},
							}

							resp, err := client.Get(fmt.Sprintf("%s?code=%s", v.Url, v.Code))

							if err != nil {
								return err
							} else if resp.StatusCode == 200 {
								var body = make([]byte, resp.ContentLength)
								_, _ = resp.Body.Read(body)
								defer resp.Body.Close()
								vv := string(body)
								if vv == "SUCCESS" {
									return nil
								}
								return app.NewError(ERROR_NOTIFY, vv)
							} else {
								var body = make([]byte, resp.ContentLength)
								_, _ = resp.Body.Read(body)
								defer resp.Body.Close()
								return app.NewError(ERROR_NOTIFY, fmt.Sprintf("[%d] %s", resp.StatusCode, string(body)))
							}

						}()

						if err != nil {

							log.Println("NotifyService", "Runloop", err)

							v.Code = ""
							v.Count = v.Count + 1
							v.Errmsg = err.Error()

							if v.MaxCount > 0 && v.Count >= v.MaxCount {
								v.Status = NotifyStatusFail
								_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true, "status": true, "count": true, "errmsg": true})
							} else {
								_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true, "count": true, "errmsg": true})
							}

						} else {
							v.Code = ""
							v.Status = NotifyStatusOK
							_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true, "status": true})
						}

					}(v)

				}

				return nil
			}()

			if err != nil {
				log.Println("NotifyService", "Runloop", "Fail", err)
			} else {
				log.Println("NotifyService", "Runloop", "OK")
			}

			select {
			case <-S.in:
				continue
			default:
				go func() {
					time.Sleep(6 * time.Second)
					S.in <- true
				}()
			}

			<-S.in
		}

	}()

	return nil
}

func (S *NotifyService) HandleNotifyCreateTask(a INotifyApp, task *NotifyCreateTask) error {

	var db, err = a.GetDB()

	if err != nil {
		task.Result.Errno = ERROR_NOTIFY
		task.Result.Errmsg = err.Error()
		return nil
	}

	v := Notify{}

	v.Url = task.Url
	v.Type = task.Type
	v.Content = task.Content
	v.MaxCount = task.MaxCount
	v.Expires = task.Expires
	v.Ctime = time.Now().Unix()

	_, err = kk.DBInsert(db, a.GetNotifyTable(), a.GetPrefix(), &v)

	if err != nil {
		task.Result.Errno = ERROR_NOTIFY
		task.Result.Errmsg = err.Error()
		return nil
	}

	S.in <- true

	task.Result.Notify = &v

	return nil
}

func (S *NotifyService) HandleNotifyTask(a INotifyApp, task *NotifyTask) error {

	var db, err = a.GetDB()

	if err != nil {
		task.Result.Errno = ERROR_NOTIFY
		task.Result.Errmsg = err.Error()
		return nil
	}

	v := Notify{}

	if task.Id != 0 {

		rows, err := kk.DBQuery(db, a.GetNotifyTable(), a.GetPrefix(), " WHERE id=?", task.Id)

		if err != nil {
			task.Result.Errno = ERROR_NOTIFY
			task.Result.Errmsg = err.Error()
			return nil
		}

		defer rows.Close()

		if rows.Next() {

			scanner := kk.NewDBScaner(&v)

			err = scanner.Scan(rows)

			if err != nil {
				task.Result.Errno = ERROR_NOTIFY
				task.Result.Errmsg = err.Error()
				return nil
			}

		} else {
			task.Result.Errno = ERROR_NOTIFY
			task.Result.Errmsg = "Not Found notify"
			return nil
		}

	} else if task.Code != "" {

		rows, err := kk.DBQuery(db, a.GetNotifyTable(), a.GetPrefix(), " WHERE code=?", task.Code)

		if err != nil {
			task.Result.Errno = ERROR_NOTIFY
			task.Result.Errmsg = err.Error()
			return nil
		}

		defer rows.Close()

		if rows.Next() {

			scanner := kk.NewDBScaner(&v)

			err = scanner.Scan(rows)

			if err != nil {
				task.Result.Errno = ERROR_NOTIFY
				task.Result.Errmsg = err.Error()
				return nil
			}

		} else {
			task.Result.Errno = ERROR_NOTIFY
			task.Result.Errmsg = "Not Found notify"
			return nil
		}

	} else {
		task.Result.Errno = ERROR_NOTIFY_NOT_FOUND_ID
		task.Result.Errmsg = "Not Found notify id"
		return nil
	}

	task.Result.Notify = &v

	return nil
}
