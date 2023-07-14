package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

// TelegramHook is a hook that writes logs of specified LogLevels to specified Writer
type TelegramHook struct {
	c           *http.Client
	authToken   string
	targetID    string
	apiEndpoint string
}

type apiRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type apiResponse struct {
	ErrorCode *int         `json:"error_code,omitempty"`
	Desc      *string      `json:"description,omitempty"`
	Result    *interface{} `json:"result,omitempty"`
	Ok        bool         `json:"ok"`
}

func NewTelegramHook(authToken, targetID string) (*TelegramHook, error) {
	client := &http.Client{}

	apiEndpoint := fmt.Sprintf(
		"https://api.telegram.org/bot%s",
		authToken,
	)

	h := TelegramHook{
		c:           client,
		authToken:   authToken,
		targetID:    targetID,
		apiEndpoint: apiEndpoint,
	}

	return &h, nil
}

func (hook *TelegramHook) sendMessage(msg string) error {
	apiReq := apiRequest{
		ChatID:    hook.targetID,
		Text:      msg,
		ParseMode: "HTML",
	}
	b, err := json.Marshal(apiReq)
	if err != nil {
		return err
	}

	res, err := hook.c.Post(
		strings.Join([]string{hook.apiEndpoint, "sendmessage"}, "/"),
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("telegram hook: encountered error when issuing request to Telegram API: %s", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Error(err)
		}
	}(res.Body)

	apiRes := apiResponse{}
	err = json.NewDecoder(res.Body).Decode(&apiRes)
	if err != nil {
		return err
	}

	if !apiRes.Ok {
		msg := "Received error response from Telegram API"
		if apiRes.ErrorCode != nil {
			msg = fmt.Sprintf("%s (error code %d)", msg, *apiRes.ErrorCode)
		}
		if apiRes.Desc != nil {
			msg = fmt.Sprintf("%s: %s", msg, *apiRes.Desc)
		}
		return fmt.Errorf(msg)
	}

	return nil
}

// Fire will be called when some logging function is called with current hook It will format log entry to string and write it to appropriate writer
func (hook *TelegramHook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()

	err = hook.sendMessage(msg)
	if err != nil {
		return fmt.Errorf("telegram hook: unable to send message: %s", err)
	}

	return nil
}

// Levels define on which log levels this hook would trigger
func (hook *TelegramHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
