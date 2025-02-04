package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	TempDir          = "temp"
	DownloadDir      = "download"
	Addr             = "localhost:8080"
	GinMode          = "debug"
	KodikToken       string
	ConcurrencyLimit = 100
)

func LoadEnv() error {
	var err error
	if err = godotenv.Load(); err != nil {
		return err
	}

	Addr = os.Getenv("ADDRESS")
	TempDir = os.Getenv("TEMP_DIR")
	DownloadDir = os.Getenv("DOWNLOAD_DIR")
	KodikToken = os.Getenv("KODIK_TOKEN")
	GinMode = os.Getenv("GIN_MODE")
	ConcurrencyLimit, err = strconv.Atoi(os.Getenv("CONCURRENCY_LIMIT"))

	if err != nil {
		return err
	}

	return nil
}
