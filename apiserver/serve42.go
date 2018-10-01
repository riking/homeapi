/*
Copyright 2018 Google LLC
Copyright 2016 Kane York

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

var killText = []byte(`#!/bin/sh

[ -d .git ] || git rev-parse --git-dir > /dev/null 2>&1 && (
	git commit --allow-empty -m 'Ran a curl | sh command'
)

SHELL_NAME=${0:-sh}
echo "Don't pipe curl into $SHELL_NAME. Someone could run naughty commands."
echo "Always save the script to a file and inspect it before running."

osascript -e 'say "Don\'t pipe curl into sh"'
exit 3

`)

func curlKiller(wrap http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		wrap.ServeHTTP(rec, r)

		rec.HeaderMap.Del("Content-Length")
		for k, v := range rec.HeaderMap {
			w.Header()[k] = v
		}
		r.Header.Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(rec.Code)

		if strings.Contains(r.Header.Get("User-Agent"), "curl") {
			w.Write(killText)
		}

		rec.Body.WriteTo(w)
	})
}
