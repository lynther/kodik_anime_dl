package downloader

import (
	"errors"
	"fmt"
	"io"
	"kodik_anime_dl/pkgs/utils"
	"kodik_anime_dl/pkgs/utils/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"
)

var (
	manifestRe  = regexp.MustCompile(`/\d+\.mp4:hls:manifest\.m3u8`)
	segmentRe   = regexp.MustCompile(`\./\d+\.mp4:hls:seg-\d+-v\d+-a\d+\.ts`)
	segNumberRe = regexp.MustCompile(`seg-(\d+)`)
	Progress    int32
	Total       int32
	IsRunning   bool
	State       string
)

type Segment struct {
	URL      string
	FilePath string
}

func isAllSegmentsDownload(segments []Segment) bool {
	for _, segment := range segments {
		if _, err := os.Stat(segment.FilePath); err != nil {
			return false
		}
	}
	return true
}

func getSegmentFileName(segmentURL string) string {
	segmentNumber := segNumberRe.FindStringSubmatch(segmentURL)[1]
	return fmt.Sprintf("%s.ts", segmentNumber)
}

func getSegments(m3u8Url, tempDirPath string) ([]Segment, error) {
	var segments []Segment
	cutM3u8Url := manifestRe.ReplaceAllString(m3u8Url, "")
	body, err := http.GetString(m3u8Url)

	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(body, "\n") {
		if segmentRe.MatchString(line) {
			segmentURL := fmt.Sprintf("%s%s", cutM3u8Url, line[1:])
			segments = append(segments, Segment{
				URL:      segmentURL,
				FilePath: path.Join(tempDirPath, getSegmentFileName(segmentURL)),
			})
		}
	}

	return segments, nil
}

func downloadSegmentWorker(segmentURL, filePath string) {
	var err error
	var body io.ReadCloser
	var outFile *os.File

	body, err = http.GetBodyWithRetries(segmentURL, 3, 15*time.Second)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer body.Close()

	outFile, err = os.Create(filePath)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer outFile.Close()

	if _, err = io.Copy(outFile, body); err != nil {
		fmt.Println(err)
		return
	}

	atomic.AddInt32(&Progress, 1)
}

func downloadSegments(segments []Segment, concurrencyLimit int) {
	p := pool.New().WithMaxGoroutines(concurrencyLimit)

	for _, segment := range segments {
		p.Go(func() {
			downloadSegmentWorker(segment.URL, segment.FilePath)
		})
	}

	p.Wait()
}

func isExistPath(p string) bool {
	if _, err := os.Stat(p); err != nil {
		return false
	}
	return true
}

func DownloadVideo(videoLink, tempDirPath, outputFilePath string, concurrencyLimit int) error {
	if concurrencyLimit == 0 {
		concurrencyLimit = 200
	}

	if !isExistPath(tempDirPath) {
		return fmt.Errorf("директория не существует: %s", tempDirPath)
	}

	var err error
	segments, err := getSegments(videoLink, tempDirPath)

	if err != nil {
		return err
	}

	IsRunning = true
	Progress = 0
	Total = int32(len(segments))
	State = ""

	downloadSegments(segments, concurrencyLimit)

	if !isAllSegmentsDownload(segments) {
		return errors.New("не все сегменты были скачаны")
	}

	State = "combine"

	if err = combineSegments(segments, outputFilePath); err != nil {
		return err
	}

	err = utils.ClearTmp(tempDirPath)

	if err != nil {
		return err
	}

	IsRunning = false

	return nil
}

func combineSegments(segments []Segment, outputFilePath string) error {
	var err error
	var segmentFile *os.File
	outputFile, err := os.Create(outputFilePath)

	if err != nil {
		return err
	}

	defer outputFile.Close()

	for _, segment := range segments {
		segmentFile, err = os.Open(segment.FilePath)

		if err != nil {
			return err
		}

		if _, err = io.Copy(outputFile, segmentFile); err != nil {
			return err
		}

		segmentFile.Close()
	}

	return nil
}
