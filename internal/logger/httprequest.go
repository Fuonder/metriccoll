package logger

import (
	"fmt"
	"time"
)

type RequestData struct {
	url       string
	method    string
	timeStart time.Time
}

func NewRequestData() *RequestData {
	return &RequestData{url: "", method: "", timeStart: time.Now()}
}

func (r *RequestData) Set(uri string, method string) {
	r.url = uri
	r.method = method
}

//func (r *RequestData) GetTimeSpent() (time.Duration, error) {
//	if r.timeStart.IsZero() {
//		return time.Duration(0), fmt.Errorf("request start time must be greater than zero")
//	}
//	return time.Since(r.timeStart), nil
//
//}

func (r *RequestData) GetMethod() string {
	return r.method
}
func (r *RequestData) GetURI() string {
	return r.method
}

func (r *RequestData) String() string {
	return fmt.Sprintf("request:\n\tURI:\t%s\n\tMETHOD:\t%s\n\tTIMESTART:\t%s\n\n",
		r.url, r.method, r.timeStart.String())
}
