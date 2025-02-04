package kodik

import (
	"encoding/base64"
	"fmt"
	"kodik_anime_dl/pkgs/utils/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	videoInfoTypeRe = regexp.MustCompile(`videoInfo\.type = '(\w+)';`)
	videoInfoHashRe = regexp.MustCompile(`videoInfo\.hash = '([a-fA-F0-9]+)';`)
	videoInfoIDRe   = regexp.MustCompile(`videoInfo\.id = '(\d+)';`)
)

type FormData struct {
	Hash string
	Type string
	ID   string
}

type Episode struct {
	Link string `json:"link"`
}

type Translation struct {
	Title        string
	Type         string
	EpisodeCount int
	Link         string
	Episodes     map[int]Episode
}

type ShikimoriInfo struct {
	Poster   string
	EngName  string
	RuName   string
	Duration string
	AiredOn  string
	URL      string
	ID       string
}

type SearchResult struct {
	Empty         bool
	MediaType     string
	Title         string
	ShikimoriInfo ShikimoriInfo
	Translations  map[int]Translation
}

// GetLink Получает ссылку на эпизод или фильм.
func (r SearchResult) GetLink(translationID, episodeNumber int) (string, error) {
	translation, err := r.GetTranslationByID(translationID)

	if err != nil {
		return "", err
	}

	if r.MediaType == "anime" {
		return translation.Link, nil
	}

	episode, ok := translation.Episodes[episodeNumber]

	if ok {
		return episode.Link, nil
	}

	return "", fmt.Errorf("ссылка на эпизод %d не существует", episodeNumber)
}

func (r SearchResult) GetTranslationByID(translationID int) (Translation, error) {
	translation, ok := r.Translations[translationID]

	if !ok {
		return Translation{}, fmt.Errorf("перевод с Id (%d) не найден", translationID)
	}

	return translation, nil
}

func rot13(input string) string {
	output := make([]rune, len(input))
	for i, char := range input {
		switch {
		case char >= 'a' && char <= 'z':
			output[i] = (char-'a'+13)%26 + 'a'
		case char >= 'A' && char <= 'Z':
			output[i] = (char-'A'+13)%26 + 'A'
		default:
			output[i] = char
		}
	}
	return string(output)
}

func decodeSrc(src string) (string, error) {
	convertedSrc := rot13(src)

	for len(convertedSrc)%4 != 0 {
		convertedSrc += "="
	}

	decodedSrc, err := base64.StdEncoding.DecodeString(convertedSrc)
	stringSrc := string(decodedSrc)

	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(stringSrc, "https") {
		return fmt.Sprintf("https:%s", stringSrc), nil
	}

	return stringSrc, err
}

func parseBodyValue(body string, re *regexp.Regexp) (string, error) {
	matches := re.FindStringSubmatch(body)

	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("не удалось получить значение по регулярному выражению: %s", re.String())
}

func getShikimoriInfo(shikimoriID string) (ShikimoriInfo, error) {
	domain := "https://shikimori.one"
	url := fmt.Sprintf("%s/api/animes/%s", domain, shikimoriID)
	body, err := http.GetString(url)

	if err != nil {
		return ShikimoriInfo{}, err
	}

	poster := gjson.Get(body, "image.original")

	if !poster.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить постер с shikimori.one/api, ID: %s", shikimoriID)
	}

	engName := gjson.Get(body, "name")

	if !engName.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить имя с shikimori.one/api, ID: %s", shikimoriID)
	}

	ruName := gjson.Get(body, "russian")

	if !ruName.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить русское имя с shikimori.one/api, ID: %s", shikimoriID)
	}

	duration := gjson.Get(body, "duration")

	if !ruName.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить длительность с shikimori.one/api, ID: %s", shikimoriID)
	}

	animeURL := gjson.Get(body, "url")

	if !animeURL.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить \"url\" с shikimori.one/api, ID: %s", shikimoriID)
	}

	airedOn := gjson.Get(body, "aired_on")

	if !airedOn.Exists() {
		return ShikimoriInfo{}, fmt.Errorf("не удалось получить дату выхода с shikimori.one/api, ID: %s", shikimoriID)
	}

	return ShikimoriInfo{
		ID:       shikimoriID,
		Poster:   fmt.Sprintf("%s%s", domain, poster.String()),
		EngName:  engName.String(),
		RuName:   ruName.String(),
		Duration: duration.String(),
		URL:      fmt.Sprintf("%s%s", domain, animeURL.String()),
		AiredOn:  airedOn.String(),
	}, err
}

func getFormData(body string) (FormData, error) {
	videoInfoHash, err := parseBodyValue(body, videoInfoHashRe)
	if len(videoInfoHash) == 0 {
		return FormData{}, err
	}

	videoInfoType, err := parseBodyValue(body, videoInfoTypeRe)
	if len(videoInfoType) == 0 {
		return FormData{}, err
	}

	videoInfoID, err := parseBodyValue(body, videoInfoIDRe)
	if len(videoInfoID) == 0 {
		return FormData{}, err
	}

	return FormData{
		Hash: videoInfoHash,
		Type: videoInfoType,
		ID:   videoInfoID,
	}, nil
}

func GetVideoLink(link string) (string, error) {
	body, err := http.GetString(link)

	if err != nil {
		return "", err
	}

	formData, err := getFormData(body)

	if err != nil {
		return "", err
	}

	payload := strings.NewReader(fmt.Sprintf("hash=%s&id=%s&type=%s",
		formData.Hash,
		formData.ID,
		formData.Type,
	),
	)
	body, err = http.PostString("https://kodik.info/ftor", payload)

	if err != nil {
		return "", err
	}

	src := gjson.Get(body, "links.720.0.src")

	if len(src.String()) == 0 {
		return "", fmt.Errorf("%s не удалось получить ссылку на видео", link)
	}

	decodedSrc, err := decodeSrc(src.String())

	if err != nil {
		return "", err
	}

	return decodedSrc, nil
}

func parseEpisodes(result gjson.Result) map[int]Episode {
	episodes := make(map[int]Episode)
	lastSeason := result.Get("last_season").String()
	season := result.Get(fmt.Sprintf("seasons.%s.episodes", lastSeason))

	for episodeNumberStr, episodeURL := range season.Map() {
		episodeNumber, _ := strconv.Atoi(episodeNumberStr)
		episodes[episodeNumber] = Episode{Link: fmt.Sprintf("https:%s", episodeURL.String())}
	}

	return episodes
}

// TODO Сделать проверку ошибок
// SearchByShikimoriID Ищет аниме сериал или аниме фильм по ID shikimori
// возвращает структуру SearchResult.
func SearchByShikimoriID(token string, shikimoriID string) (SearchResult, error) {
	var translationType string
	translations := make(map[int]Translation)

	url := fmt.Sprintf(
		"https://kodikapi.com/search?token=%s&shikimori_id=%s&with_episodes=true",
		token,
		shikimoriID,
	)
	body, err := http.GetString(url)

	if err != nil {
		return SearchResult{}, err
	}

	total := gjson.Get(body, "total")

	if !total.Exists() {
		return SearchResult{}, fmt.Errorf("не удалось получить результаты поиска: %s", shikimoriID)
	}

	if total.Exists() && total.Int() == 0 {
		return SearchResult{Empty: true}, nil
	}

	results := gjson.Get(body, "results")
	firstResult := results.Array()[0]
	lastSeason := firstResult.Get("last_season").String()
	animeTitle := firstResult.Get("title_orig").String()
	resultType := firstResult.Get("type").String()
	shikimoriInfo, err := getShikimoriInfo(shikimoriID)

	if err != nil {
		return SearchResult{}, err
	}

	for _, result := range results.Array() {
		// Существуют только специальные эпизоды
		if result.Get("seasons.0").Exists() && !result.Get(fmt.Sprintf("seasons.%s", lastSeason)).Exists() {
			continue
		}

		episodeCount := int(result.Get("episodes_count").Int())

		if result.Get("seasons.0").Exists() {
			episodeCount--
		}

		episodes := parseEpisodes(result)

		switch result.Get("translation.type").String() {
		case "voice":
			translationType = "Озвучка"
		case "subtitles":
			translationType = "Субтитры"
		default:
			translationType = "Не известно"
		}

		translations[int(result.Get("translation.id").Int())] = Translation{
			Title:        result.Get("translation.title").String(),
			Type:         translationType,
			EpisodeCount: episodeCount,
			Link:         fmt.Sprintf("https:%s", result.Get("link").String()),
			Episodes:     episodes,
		}
	}

	return SearchResult{
		Title:         animeTitle,
		Translations:  translations,
		MediaType:     resultType,
		ShikimoriInfo: shikimoriInfo,
		Empty:         false,
	}, nil
}
