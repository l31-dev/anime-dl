// l31   /\
//     _/./
//  ,-'    `-:..-'/
// : o )      _  (
// "`-....,--; `-.\
//     `'
// https://puffer.fish/~l31

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "watch" {
		watch(os.Args[2:])
	} else {
		fmt.Println("Usage: program watch [options] <title>")
	}
}

func watch(args []string) {
	var startEpisode, endEpisode int
	var lang string

	flagSet := flag.NewFlagSet("watch", flag.ExitOnError)
	flagSet.IntVar(&startEpisode, "start", 1, "start episode number")
	flagSet.IntVar(&endEpisode, "end", 1, "end episode number")
	flagSet.StringVar(&lang, "l", "vostfr", "language")
	flagSet.Parse(args)

	title := flagSet.Arg(0)
	if title == "" {
		log.Fatal("Title is required")
	}

	if endEpisode < startEpisode {
		endEpisode = startEpisode
	}

	encodedTitle := url.QueryEscape(title)
	animeID, realTitle := fetchAnimeInfo(encodedTitle)

	// Create a temporary directory for storing downloaded files and playlist
	tempDir := createTempDir(realTitle)
	defer os.RemoveAll(tempDir)

	playlistFile := filepath.Join(tempDir, "playlist.m3u8")
	createPlaylist(playlistFile)

	for episode := startEpisode; episode <= endEpisode; episode++ {
		animeVideo := fetchAnimeVideo(animeID, episode, lang)
		tempFile := createTempFile(tempDir, realTitle, episode)
		downloadAnimeVideo(animeVideo, tempFile)
		modifyTempFile(tempFile)
		appendToPlaylist(playlistFile, tempFile)
	}

	playAnime(playlistFile)
}

func fetchAnimeInfo(title string) (int, string) {
	resp, err := http.Get(fmt.Sprintf("https://api.gazes.fr/anime/animes?title=%s", title))
	if err != nil {
		log.Fatalf("Error fetching anime info: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Error decoding anime info: %v", err)
	}

	if len(result.Data) == 0 {
		log.Fatal("No anime data found")
	}

	animeID := result.Data[0].ID
	realTitle := strings.ReplaceAll(result.Data[0].Title, " ", "-")
	return animeID, realTitle
}

func fetchAnimeVideo(animeID, episode int, lang string) string {
	resp, err := http.Get(fmt.Sprintf("https://api.gazes.fr/anime/animes/%d/%d", animeID, episode))
	if err != nil {
		log.Fatalf("Error fetching anime: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data map[string]struct {
			VideoURI string `json:"videoUri"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Error decoding anime video: %v", err)
	}

	video, found := result.Data[lang]
	if !found {
		video, found = result.Data["vostfr"]
	}
	if !found {
		log.Fatal("No valid video URI found")
	}

	return video.VideoURI
}

func createTempDir(realTitle string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Error generating random bytes: %v", err)
	}

	tempDir := fmt.Sprintf("/tmp/%s_%s", realTitle, hex.EncodeToString(bytes))
	if err := os.Mkdir(tempDir, 0755); err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}

	return tempDir
}

func createTempFile(tempDir, realTitle string, episode int) string {
	tempFileName := fmt.Sprintf("%s_%d.m3u8", realTitle, episode)
	return filepath.Join(tempDir, tempFileName)
}

func createPlaylist(playlistFile string) {
	file, err := os.Create(playlistFile)
	if err != nil {
		log.Fatalf("Error creating playlist file: %v", err)
	}
	defer file.Close()
}

func appendToPlaylist(playlistFile, tempFile string) {
	file, err := os.OpenFile(playlistFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening playlist file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%s\n", tempFile)); err != nil {
		log.Fatalf("Error writing to playlist file: %v", err)
	}
}

func downloadAnimeVideo(animeVideo, tempFile string) {
	resp, err := http.Get(animeVideo)
	if err != nil {
		log.Fatalf("Error downloading anime video: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading video response body: %v", err)
	}

	if err := os.WriteFile(tempFile, body, 0644); err != nil {
		log.Fatalf("Error writing to temp file: %v", err)
	}
}

func modifyTempFile(tempFile string) {
	input, err := os.ReadFile(tempFile)
	if err != nil {
		log.Fatalf("Error reading temp file: %v", err)
	}

	output := strings.ReplaceAll(string(input), "{PROXY_URL}", "https://proxy.ketsuna.com")
	if err := os.WriteFile(tempFile, []byte(output), 0644); err != nil {
		log.Fatalf("Error writing modified temp file: %v", err)
	}
}

func playAnime(playlistFile string) {
	cmd := exec.Command("vlc", "--playlist-autostart", playlistFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running VLC: %v", err)
	}
}
