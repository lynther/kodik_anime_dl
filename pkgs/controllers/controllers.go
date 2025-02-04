package controllers

import (
	"fmt"
	"kodik_anime_dl/pkgs/config"
	"kodik_anime_dl/pkgs/kodik"
	"kodik_anime_dl/pkgs/kodik/downloader"
	"kodik_anime_dl/pkgs/utils"
	"net/http"
	"path"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AppState struct {
	SearchResult  kodik.SearchResult
	EpisodeNumber int
	translationID int
}

func (s *AppState) Clear() {
	s.SearchResult = kodik.SearchResult{}
	s.translationID = 0
	s.EpisodeNumber = 0
}

var (
	appState = &AppState{}
)

func ErrorPage(c *gin.Context, err error) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": err.Error(),
	})
}

func Home(c *gin.Context) {
	appState.Clear()
	c.HTML(http.StatusOK, "index.tmpl", nil)
}

func Search(c *gin.Context) {
	shikimoriID := c.DefaultPostForm("shikimori_id", "1")
	searchResult, err := kodik.SearchByShikimoriID(config.KodikToken, shikimoriID)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	appState.SearchResult = searchResult

	if searchResult.Empty {
		c.HTML(http.StatusNotFound, "404.tmpl", nil)
		return
	}

	c.Redirect(http.StatusFound, "/translations")
}

func Translations(c *gin.Context) {
	searchResult := appState.SearchResult

	if len(appState.SearchResult.Translations) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	animeTitle := fmt.Sprintf("%s / %s",
		searchResult.ShikimoriInfo.RuName,
		searchResult.ShikimoriInfo.EngName,
	)

	c.HTML(http.StatusOK, "translations.tmpl", gin.H{
		"mediaType":         searchResult.MediaType,
		"poster":            searchResult.ShikimoriInfo.Poster,
		"airedOn":           searchResult.ShikimoriInfo.AiredOn,
		"duration":          searchResult.ShikimoriInfo.Duration,
		"shikimoriUrl":      searchResult.ShikimoriInfo.URL,
		"animeTitle":        animeTitle,
		"animeTitleRu":      searchResult.ShikimoriInfo.RuName,
		"translations":      searchResult.Translations,
		"totalTranslations": len(searchResult.Translations),
		"shikimoriID":       searchResult.ShikimoriInfo.ID,
	})
}

func Translation(c *gin.Context) {
	var err error
	var translation kodik.Translation

	if len(appState.SearchResult.Translations) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	translationIDStr := c.PostForm("translation_id")
	translationID, err := strconv.Atoi(translationIDStr)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	mediaType := appState.SearchResult.MediaType
	appState.translationID = translationID

	if mediaType != "anime-serial" {
		translation, err = appState.SearchResult.GetTranslationByID(translationID)
		if err != nil {
			ErrorPage(c, err)
			return
		}

		translation.Episodes[0] = kodik.Episode{Link: translation.Link}
	}

	c.Redirect(http.StatusFound, "/episodes")
}

func Episodes(c *gin.Context) {
	if len(appState.SearchResult.Translations) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	translationID := appState.translationID
	searchResult := appState.SearchResult
	translation, err := searchResult.GetTranslationByID(translationID)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	episodes := translation.Episodes

	if len(episodes) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	episodeNumbers := make([]int, 0)

	for episodeNumber := range episodes {
		episodeNumbers = append(episodeNumbers, episodeNumber)
	}

	sort.Ints(episodeNumbers)

	c.HTML(http.StatusOK, "episodes.tmpl", gin.H{
		"animeTitle":       searchResult.Title,
		"translationTitle": translation.Title,
		"mediaType":        searchResult.MediaType,
		"episodes":         episodeNumbers,
		"episodeCount":     len(episodeNumbers),
	})
}

func Episode(c *gin.Context) {
	if len(appState.SearchResult.Translations) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	episodeNumberIDStr := c.PostForm("episode_number")
	episodeNumber, err := strconv.Atoi(episodeNumberIDStr)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	appState.EpisodeNumber = episodeNumber
	c.Redirect(http.StatusFound, "/download")
}

func DownloadStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"state":      downloader.State,
		"is_running": downloader.IsRunning,
		"progress":   downloader.Progress,
		"total":      downloader.Total,
	})
}

func Download(c *gin.Context) {
	var err error

	if len(appState.SearchResult.Translations) == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	episodeNumber := appState.EpisodeNumber
	translationID := appState.translationID
	searchResult := appState.SearchResult

	outputFileName := fmt.Sprintf("%s-Серия-%d-Перевод-%s.mp4",
		searchResult.Title,
		episodeNumber,
		searchResult.Translations[translationID].Title,
	)

	if searchResult.MediaType == "anime" {
		outputFileName = fmt.Sprintf("%s-Перевод-%s.mp4",
			searchResult.Title,
			searchResult.Translations[translationID].Title,
		)
	}

	outputFileName = utils.SanitizeFilename(outputFileName)
	link, err := searchResult.GetLink(translationID, episodeNumber)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	srcLink, err := kodik.GetVideoLink(link)

	if err != nil {
		ErrorPage(c, err)
		return
	}

	tempDirPath := config.TempDir
	downloadDirPath := config.DownloadDir

	if err = utils.Mkdir(tempDirPath); err != nil {
		ErrorPage(c, err)
		return
	}

	if err = utils.Mkdir(downloadDirPath); err != nil {
		ErrorPage(c, err)
		return
	}

	outFilePath := path.Join(downloadDirPath, outputFileName)

	if err = downloader.DownloadVideo(srcLink, tempDirPath, outFilePath, config.ConcurrencyLimit); err != nil {
		ErrorPage(c, err)
		return
	}

	c.FileAttachment(outFilePath, outputFileName)
}
