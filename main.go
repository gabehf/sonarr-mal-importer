package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/darenliang/jikan-go"
)

type ResponseItem struct {
	Title  string `json:"title"`
	MalId  int    `json:"malId"`
	TvdbId int    `json:"tvdbId"`
}
type AnimeEntry struct {
	TvdbId int         `json:"tvdb_id"`
	MalId  interface{} `json:"mal_id"`
}
type IdList struct {
}

func main() {
	log.Println("Building Anime ID Associations...")
	malToTvdb := buildIdMap()
	http.HandleFunc("/anime", handleAnimeSearch(malToTvdb))
	log.Println("Listening on :3333")
	log.Fatal(http.ListenAndServe(":3333", nil))
}

func handleAnimeSearch(malToTvdb map[int]int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(getAnimeSearch(malToTvdb, r)))
	}
}

func getAnimeSearch(malToTvdb map[int]int, r *http.Request) string {
	q := r.URL.Query()

	hasNextPage := true
	page := 0
	resp := []ResponseItem{}
	for hasNextPage {
		page++
		q.Set("page", strconv.Itoa(page))
		result, err := jikan.GetAnimeSearch(q)
		if err != nil {
			log.Fatal(err)
		}

		// map the data
		for _, item := range result.Data {
			resp = append(resp,
				ResponseItem{
					item.Title,
					item.MalId,
					malToTvdb[item.MalId],
				})
		}
		hasNextPage = result.Pagination.HasNextPage
	}

	respJson, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return string(respJson)
}

func buildIdMap() map[int]int {
	// build the mal -> tvdb association table
	var idListBytes []byte
	resp, err := http.Get("https://raw.githubusercontent.com/Kometa-Team/Anime-IDs/master/anime_ids.json")
	if err != nil {
		log.Fatal(err)
	}
	idListBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var animeMap map[string]AnimeEntry
	err = json.Unmarshal(idListBytes, &animeMap)
	if err != nil {
		log.Fatal(err)
	}
	malToTvdb := make(map[int]int, 0)
	for _, entry := range animeMap {
		if entry.MalId == nil {
			continue
		}
		var malIdList []int
		switch t := reflect.TypeOf(entry.MalId); t.Kind() {
		case reflect.String:
			s := strings.Split(entry.MalId.(string), ",")
			for _, ss := range s {
				id, err := strconv.Atoi(ss)
				if err != nil {
					log.Fatal(err)
				}
				malIdList = append(malIdList, id)
			}
		case reflect.Float64:
			malIdList = append(malIdList, int(entry.MalId.(float64)))
		}
		for _, val := range malIdList {
			malToTvdb[val] = entry.TvdbId
		}
	}
	return malToTvdb
}
