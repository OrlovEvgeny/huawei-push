package push

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nburmi/huawei-push/common"
	"github.com/nburmi/huawei-push/token"
)

const messagingEndpoint = "https://push-api.cloud.huawei.com/v1/%s/messages:send"

type Response struct {
	StatusCode    int    `json:"-"`
	Code          Code   `json:"code"`
	Message       string `json:"msg"`
	RequestID     string `json:"requestId"`
	IllegalTokens string `json:"illegal_token,omitempty"`
}

type Pusher interface {
	Push(*Message) (*Response, error)
	PushValidate(*Message) (*Response, error)
}

func New(appID string, t token.Tokener, d common.HTTPDoer) Pusher {
	return &pusher{endpoint: fmt.Sprintf(messagingEndpoint, appID), Tokener: t, HTTPDoer: d}
}

type pusher struct {
	token.Tokener
	common.HTTPDoer

	endpoint string
}

func (p *pusher) Push(m *Message) (*Response, error) {
	return p.push(&dataPush{M: m})
}

func (p *pusher) PushValidate(m *Message) (*Response, error) {
	return p.push(&dataPush{M: m, ValidateOnly: true})
}

func checkAndSetDefaultVals(m *Message) {
	/*
		FastAppTarget
			1: development state.
			2: production state (default value).
	*/
	if m.Android != nil && m.Android.FastAppTarget == 0 {
		m.Android.FastAppTarget = 2
	}
}

func (p *pusher) push(d *dataPush) (*Response, error) {
	checkAndSetDefaultVals(d.M)

	tok, err := p.Tokener.Get()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal(d) %v", err)
	}

	r, err := http.NewRequest("POST", p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", fmt.Sprintf("%s %s", tok.TokenType, tok.AccessToken))
	r.Header.Add("Content-Type", "application/json")

	resp, err := p.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	rs := &Response{StatusCode: resp.StatusCode}

	err = json.NewDecoder(resp.Body).Decode(&rs)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return rs, nil
}
