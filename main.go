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
	var episode int
	var lang string

	flagSet := flag.NewFlagSet("watch", flag.ExitOnError)
	flagSet.IntVar(&episode, "e", 1, "episode number")
	flagSet.StringVar(&lang, "l", "vostfr", "language")
	flagSet.Parse(args)

	title := flagSet.Arg(0)
	if title == "" {
		log.Fatal("Title is required")
	}

	encodedTitle := url.QueryEscape(title)
	animeID, realTitle := fetchAnimeInfo(encodedTitle)
	animeVideo := fetchAnimeVideo(animeID, episode, lang)

	tempFile := createTempFile(realTitle, episode)
	downloadAnimeVideo(animeVideo, tempFile)
	modifyTempFile(tempFile)

	playAnime(tempFile)
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

func createTempFile(realTitle string, episode int) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Error generating random bytes: %v", err)
	}

	tempFileName := fmt.Sprintf("/tmp/%s_%d_%s.m3u8", realTitle, episode, hex.EncodeToString(bytes))
	return tempFileName
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

func playAnime(tempFile string) {
	cmd := exec.Command("vlc", tempFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running VLC: %v", err)
	}
}
