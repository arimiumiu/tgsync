package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const apiBase = "https://api.telegram.org/bot"

type Uploader struct {
	token  string
	chatID int64
	client *http.Client
}

func New(token string, chatID int64) *Uploader {
	return &Uploader{
		token:  token,
		chatID: chatID,
		client: &http.Client{},
	}
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func (u *Uploader) SendFile(filePath string, topicID int) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("chat_id", strconv.FormatInt(u.chatID, 10))
	if topicID != 0 {
		_ = w.WriteField("message_thread_id", strconv.Itoa(topicID))
	}

	part, err := w.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err = io.Copy(part, f); err != nil {
		return fmt.Errorf("copy file data: %w", err)
	}
	w.Close()

	url := fmt.Sprintf("%s%s/sendDocument", apiBase, u.token)
	resp, err := u.client.Post(url, w.FormDataContentType(), &buf)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !apiResp.OK {
		return fmt.Errorf("telegram api: %s", apiResp.Description)
	}
	return nil
}
