package notify

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/kkserver/kk-lib/kk"
	"github.com/kkserver/kk-lib/kk/app"
	"github.com/kkserver/kk-lib/kk/dynamic"
	"github.com/kkserver/kk-lib/kk/json"
	"log"
	"net/http"
	"sort"
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

				rows, err := kk.DBQuery(db, a.GetNotifyTable(), a.GetPrefix(), " WHERE status=? ORDER BY ASC", NotifyStatusNone)

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

					if v.Ctime+v.Expires <= time.Now().Unix() {

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

							resp, err := client.Post(fmt.Sprintf("%s?code=%s", v.Url, v.Code), v.Type, bytes.NewReader([]byte(v.Content)))

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

							if v.Count >= v.MaxCount {
								v.Status = NotifyStatusFail
								_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true, "status": true, "count": true})
							} else {
								_, _ = kk.DBUpdateWithKeys(db, a.GetNotifyTable(), a.GetPrefix(), &v, map[string]bool{"code": true, "count": true})
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
				log.Println("NotifyService", "Runloop", err)
			}

			time.Sleep(6 * time.Second)

		}

	}()

	return nil
}
