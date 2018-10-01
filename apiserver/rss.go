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
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/riking/homeapi/apiserver/rss-data"
)

type rssItem struct {
	URL      string
	Title    string
	Desc     string `json:"description"`
	CustDate time.Time
}

func (f rssItem) Description() string {
	if f.Desc != "" {
		return f.Desc
	}
	return f.Title
}

func (f rssItem) Date() string {
	return f.CustDate.Format(http.TimeFormat)
}

type infoFileFmt struct {
	FeedTitle       string `json:"title"`
	FeedDescription string `json:"description"`
	FeedLink        string `json:"link"`

	StartAt     time.Time `json:"start_at"`
	StartOffset int       `json:"start_offset"`
	PerDay      float64   `json:"per_day"`

	RawItems []rssItem `json:"-"`
	Items    []rssItem `json:"-"`
	Now      time.Time `json:"-"`
}

func (f *infoFileFmt) ItemOffset(now time.Time) int {
	daysScaled := float64(now.Sub(f.StartAt).Hours()) / 24 * f.PerDay
	return int(daysScaled)
}

func (f *infoFileFmt) TimeForOffset(offset int) time.Time {
	oneItem := time.Duration(float64(24*time.Hour) / f.PerDay)
	return f.StartAt.Add(oneItem * time.Duration(offset-f.StartOffset))
}

func (f *infoFileFmt) FeedLastUpdated() string {
	return f.TimeForOffset(f.ItemOffset(f.Now)).Format(http.TimeFormat)
}

func (f *infoFileFmt) TTL() string {
	untilNextOffset := f.TimeForOffset(f.ItemOffset(f.Now) + 1).Sub(f.Now)
	if untilNextOffset < 30*time.Minute {
		untilNextOffset = 30 * time.Minute
	} else {
		untilNextOffset += 30 * time.Minute
	}
	return fmt.Sprintf("%d", untilNextOffset/time.Second)
}

var rgxRSSName = regexp.MustCompile(`^[a-z0-9A-Z_-]+$`)
var rssTmpl = template.Must(template.New("rss.xml").Parse(string(rss_data.MustAsset("rss.xml"))))

const rssDataDir = `/tank/www/home.riking.org/rssbinge/`

func HTTPRSSBinge(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		http.Error(w, "wrong number of slashes in path.\nshould be: /rssbinge/feedname/rss.xml", http.StatusBadRequest)
		return
	}
	if !rgxRSSName.MatchString(parts[0]) {
		http.Error(w, "bad rss feed name", http.StatusBadRequest)
		return
	}

	infoF, err := os.Open(fmt.Sprintf(rssDataDir+"/%s/info.json", parts[0]))
	if err != nil {
		http.Error(w, "feed not found", http.StatusNotFound)
		return
	}
	var infoFile infoFileFmt
	err = json.NewDecoder(infoF).Decode(&infoFile)
	infoF.Close()
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprint("bad info.json content: ", err), http.StatusInternalServerError)
		return
	}

	var curTime time.Time = time.Now()
	if t := r.URL.Query().Get("at"); t != "" {
		curTime, err = time.Parse(time.RFC3339, t)
		if err != nil {
			http.Error(w, "bad 'at' query value, want RFC3339", http.StatusBadRequest)
		}
	}
	infoFile.Now = curTime

	itemF, err := os.Open(fmt.Sprintf(rssDataDir+"/%s/content.json", parts[0]))
	if err != nil {
		http.Error(w, "feed not found (content.json)", http.StatusNotFound)
		return
	}
	err = json.NewDecoder(itemF).Decode(&infoFile.RawItems)
	itemF.Close()
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprint("bad content.json content: ", err), http.StatusInternalServerError)
		return
	}

	switch parts[1] {
	case "rss.xml":
		w.Header().Set("Content-Type", "text/xml; charset=UTF-8")

		lastItemIdx := infoFile.ItemOffset(curTime) + infoFile.StartOffset
		if lastItemIdx >= len(infoFile.RawItems) {
			lastItemIdx = len(infoFile.RawItems) - 1
		}
		firstItemIdx := lastItemIdx - 10
		if firstItemIdx < 0 {
			firstItemIdx = 0
		}

		infoFile.Items = infoFile.RawItems[firstItemIdx : lastItemIdx+1]
		for i := firstItemIdx; i <= lastItemIdx; i++ {
			infoFile.RawItems[i].CustDate = infoFile.TimeForOffset(i)
		}

		err = rssTmpl.Execute(w, &infoFile)
		if err != nil {
			fmt.Println(err)
			fmt.Fprintf(w, "\n\nERROR: %s", err)
		}
	default:
		http.Error(w, "unknown request", http.StatusNotFound)
	}
}
