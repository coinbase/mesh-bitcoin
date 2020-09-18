// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// StatusRecorder is used to surface the status
// code of a HTTP response. We must use this wrapping
// because the status code is not exposed by the
// http.ResponseWriter in the http.HandlerFunc.
//
// Inspired by:
// https://stackoverflow.com/questions/53272536/how-do-i-get-response-statuscode-in-golang-middleware
type StatusRecorder struct {
	http.ResponseWriter
	Code int
}

// NewStatusRecorder returns a new *StatusRecorder.
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{w, http.StatusOK}
}

// WriteHeader stores the status code of a response.
func (r *StatusRecorder) WriteHeader(code int) {
	r.Code = code
	r.ResponseWriter.WriteHeader(code)
}

// LoggerMiddleware is a simple logger middleware that prints the requests in
// an ad-hoc fashion to the stdlib's log.
func LoggerMiddleware(loggerRaw *zap.Logger, inner http.Handler) http.Handler {
	logger := loggerRaw.Sugar().Named("server")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := NewStatusRecorder(w)

		inner.ServeHTTP(recorder, r)

		logger.Debugw(
			r.Method,
			"code", recorder.Code,
			"uri", r.RequestURI,
			"time", time.Since(start),
		)
	})
}
