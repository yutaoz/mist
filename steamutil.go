package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
)

type ByName []Game

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type Game struct {
	AppID                    int    `json:"appid"`
	Name                     string `json:"name"`
	PlaytimeForever          int    `json:"playtime_forever"`
	ImgIconURL               string `json:"img_icon_url"`
	HasCommunityVisibleStats bool   `json:"has_community_visible_stats"`
	PlaytimeWindowsForever   int    `json:"playtime_windows_forever"`
	PlaytimeMacForever       int    `json:"playtime_mac_forever"`
	PlaytimeLinuxForever     int    `json:"playtime_linux_forever"`
	PlaytimeDeckForever      int    `json:"playtime_deck_forever"`
	RtimeLastPlayed          int64  `json:"rtime_last_played"`
	PlaytimeDisconnected     int    `json:"playtime_disconnected"`
}

type LocalGame struct{}

type Response struct {
	GameCount int    `json:"game_count"`
	Games     []Game `json:"games"`
}

type JSONResponse struct {
	Response Response `json:"response"`
}

func GetAllGames(cfg map[string]string) []Game {
	apikey := GetApiKey(cfg)
	steamid := GetSteamId(cfg)
	resp, err := http.Get("https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=" + apikey + "&steamid=" + steamid + "&format=json&include_appinfo=true&include_played_free_games=true")
	if resp.StatusCode == 401 {
		log.Fatal("Error with steam api, probably incorrect key")
	}
	if err != nil {
		log.Fatal("wat")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jsonResponse JSONResponse
	e := json.Unmarshal([]byte(body), &jsonResponse)
	if e != nil {
		log.Fatal(err)
	}

	games := jsonResponse.Response.Games

	sort.Sort(ByName(games))

	return games
}

func GetInstalledGames(cfg map[string]string) []string {
	steamPath := GetSteamPath(cfg)
	files, _ := os.ReadDir(steamPath)
	appIds := extractAppIds(files)
	return appIds

}

func extractAppIds(files []os.DirEntry) []string {
	var appIDs []string
	re := regexp.MustCompile(`appmanifest_(\d+)\.acf`)
	for _, file := range files {
		if match := re.FindStringSubmatch(file.Name()); match != nil {
			appIDs = append(appIDs, match[1])
		}
	}
	return appIDs
}
