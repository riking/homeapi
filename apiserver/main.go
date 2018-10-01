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
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/riking/homeapi/intra"
)

func main() {
	fmt.Println("starting")
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/healthcheck", HTTPHealthCheck)
	apiMux.HandleFunc("/minecraftstatus.html", HTTPMCServers)
	apiMux.HandleFunc("/factoriostatus.html", HTTPFactorio)

	factorioModFS := &ModZipFilesystem{
		BaseDir:    "/tank/home/mcserver/Factorio",
		MatchRegex: regexp.MustCompile(`\A/([a-zA-Z0-9-]+)/mods\.zip\z`),
	}
	apiMux.Handle("/factoriomods/", http.StripPrefix("/factoriomods/", factorioModFS.Setup()))
	minecraftModFS.BaseDir = "/tank/home/mcserver"
	apiMux.Handle("/minecraftmods/", http.StripPrefix("/minecraftmods/", minecraftModFS.Setup()))

	apiMux.Handle("/rssbinge/", http.StripPrefix("/rssbinge/", http.HandlerFunc(HTTPRSSBinge)))

	oauthMux := http.NewServeMux()
	oauthMux.HandleFunc("/discourse", intra.HTTPDiscourseSSO)
	oauthMux.HandleFunc("/callback", intra.HTTPOauthCallback)

	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", http.StripPrefix("/api", apiMux))
	rootMux.Handle("/oauth/", http.StripPrefix("/oauth", oauthMux))
	rootMux.HandleFunc("/42/", curlKiller(http.StripPrefix("/42/", http.FileServer(http.Dir("/tank/www/home.riking.org/42")))))

	err := http.ListenAndServe("127.0.0.1:2201", rootMux)
	if err != nil {
		log.Fatalln(err)
	}
}

func pgrep(search string) ([]int32, error) {
	bytes, err := exec.Command("pgrep", search).Output()
	if exErr, ok := err.(*exec.ExitError); ok {
		if exErr.ProcessState != nil && exErr.ProcessState.Success() == false {
			// no processes
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	strPids := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	pids := make([]int32, len(strPids))
	for i := range pids {
		p, err := strconv.Atoi(strPids[i])
		if err != nil {
			return nil, err
		}
		pids[i] = int32(p)
	}
	return pids, nil
}

// ---

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func HTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.(stringWriter).WriteString("ok\n")
}

// ---
