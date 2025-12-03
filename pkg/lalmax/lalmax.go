package lalmax

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Config struct {
	URL    string
	Secret string
}

type Engine struct {
	cfg Config
	cli *http.Client
}

func NewEngine() Engine {
	return Engine{
		cli: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        30,
				MaxIdleConnsPerHost: 30,
				MaxConnsPerHost:     100,
			},
		},
	}
}

func (e Engine) SetConfig(cfg Config) Engine {
	e.cfg = cfg
	return e
}

func (e *Engine) post(ctx context.Context, path string, data map[string]any, out any) error {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.URL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
