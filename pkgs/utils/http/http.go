package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0"
)

func PostString(url string, payload *strings.Reader) (string, error) {
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return "", fmt.Errorf("%s ошибка создания запроса: %w", url, err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", fmt.Errorf("ошибка получения ответа от сервера : %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка получения ответа от сервера: %s, статус код : %d", url, resp.StatusCode)
	}

	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("%s ошибка чтения ответа от сервера: %w", url, err)
	}

	return string(response), nil
}

func GetString(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return "", fmt.Errorf("%s ошибка создания запроса: %w", url, err)
	}

	req.Header.Set("User-Agent", UserAgent)
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", fmt.Errorf("ошибка получения ответа от сервера : %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка получения ответа от сервера: %s, статус код : %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("%s ошибка чтения ответа от сервера: %w", url, err)
	}

	return string(response), nil
}

func GetBody(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return nil, fmt.Errorf("%s ошибка создания запроса: %w", url, err)
	}

	req.Header.Set("User-Agent", UserAgent)
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("ошибка получения ответа от сервера : %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения ответа от сервера: %s, статус код : %d", url, resp.StatusCode)
	}

	return resp.Body, nil
}

func GetBodyWithRetries(url string, retries uint, sleepTime time.Duration) (io.ReadCloser, error) {
	var (
		tries = retries
		body  io.ReadCloser
		err   error
	)

	if retries == 0 {
		tries = 3
	}

	for tries > 0 {
		body, err = GetBody(url)

		if err != nil {
			time.Sleep(sleepTime)
			tries--
		} else {
			return body, nil
		}
	}

	return nil, err
}

func GetStringWithRetries(url string, retries uint, sleepTime time.Duration) (string, error) {
	var (
		tries = retries
		body  string
		err   error
	)

	if retries == 0 {
		tries = 3
	}

	for tries > 0 {
		body, err = GetString(url)

		if err != nil {
			time.Sleep(sleepTime)
			tries--
		} else {
			return body, nil
		}
	}

	return "", err
}
