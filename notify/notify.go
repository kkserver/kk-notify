package notify

import (
	"crypto/md5"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/kkserver/kk-lib/kk"
	"github.com/kkserver/kk-lib/kk/app"
	"github.com/kkserver/kk-lib/kk/app/remote"
	"math/rand"
	"time"
)

const NotifyStatusNone = 0
const NotifyStatusOK = 200
const NotifyStatusExpires = 300
const NotifyStatusFail = 400

type Notify struct {
	Id       int64  `json:"id"`
	Url      string `json:"url"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Code     string `json:"code"`
	Status   int    `json:"status"`
	Count    int    `json:"count"`
	MaxCount int    `json:"maxCount"`
	Expires  int64  `json:"expires"`
	Ctime    int64  `json:"ctime"`
}

func (N *Notify) NewCode() string {
	m := md5.New()
	m.Write([]byte(fmt.Sprintf("%d %d %d", N.Id, time.Now().UnixNano(), rand.Intn(100000))))
	v := m.Sum(nil)
	return hex.EncodeToString(v)
}

type INotifyApp interface {
	app.IApp
	GetDB() (*sql.DB, error)
	GetPrefix() string
	GetNotifyTable() *kk.DBTable
	GetCA() *x509.CertPool
}

type NotifyApp struct {
	app.App
	DB *app.DBConfig

	Remote *remote.Service

	Notify      *NotifyService
	NotifyTable kk.DBTable

	ca *x509.CertPool
}

func (C *NotifyApp) GetDB() (*sql.DB, error) {
	return C.DB.Get(C)
}

func (C *NotifyApp) GetPrefix() string {
	return C.DB.Prefix
}

func (C *NotifyApp) GetNotifyTable() *kk.DBTable {
	return &C.NotifyTable
}

func (C *WeixinApp) GetCA() *x509.CertPool {
	if C.ca == nil {
		C.ca = x509.NewCertPool()
		C.ca.AppendCertsFromPEM(pemCerts)
	}
	return C.ca
}
