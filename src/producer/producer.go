/*
MIT License

Copyright (c) 2026 gounix

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package producer

import (
	"docker-rate-limit/data"
	"docker-rate-limit/logger"
	"docker-rate-limit/environ"
	"docker-rate-limit/secret"
	"net/http"
	"errors"
	"encoding/json"
	"io"
	"time"
	"fmt"
	"strings"
)

const (
        tokenUrlPattern    = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
        manifestUrlPattern = "https://registry-1.docker.io/v2/%s/manifests/%s"
	repo               = "ratelimitpreview/test"
	tag                = "latest"
)

type (
        TokenRespT struct {
                Token       string    `json:"token"`
                AccessToken string    `json:"access_token"`
                ExpiresIn   int64     `json:"expires_in"`
                IssuedAt    time.Time `json:"issued_at"`
        }
)

func getToken(repo string, user string, passwd string) string {
        var dat TokenRespT

	url := fmt.Sprintf(tokenUrlPattern, repo)
        logger.Info("producer.getToken", "url", url)

        client := &http.Client{ }
        req, err := http.NewRequest("GET", url, nil)
	if environ.Env.ImagePullSecret != "" {
		logger.Info("producer,getToken", "user", user)
		req.SetBasicAuth(user, passwd)
	}

        resp, err := client.Do(req)
        if err != nil {
                logger.Error("producer.getToken", "http.Get error", err)
                return ""
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                logger.Error("producer.getToken", "status", resp.Status)
                return ""
        }

        body, err := io.ReadAll(resp.Body)
        err = json.Unmarshal(body, &dat)
        if err != nil {
                logger.Error("producer.getToken", "unmarshal error", err)
                logger.Error("producer.getToken", "body", body)
                return ""
        }
        logger.Info("producer.getToken", "token", dat.Token[:10], "expires_in", dat.ExpiresIn, "issued_at", dat.IssuedAt)

        return dat.Token
}

func getRateLimit(token string, repo string, tag string) (string, string, string, error) {

        url := fmt.Sprintf(manifestUrlPattern, repo, tag)
        logger.Info("producer.getRateLimit", "url", url)

        client := &http.Client{ }
        req, err := http.NewRequest("HEAD", url, nil)
        req.Header.Add("Authorization", "Bearer " + token)
        resp, err := client.Do(req)
        if err != nil {
                logger.Error("producer.getRateLimit", "client.Do error", err)
                return "", "", "", err
        }
        if resp.StatusCode != 200 {
                logger.Error("producer.getRateLimit", "status", resp.Status)
                return "", "", "", errors.New(resp.Status)
        }

	// [100;w=21600] [72;w=21600] [84.107.241.9]
	limit := resp.Header.Get("Ratelimit-Limit")
	remaining := resp.Header.Get("Ratelimit-Remaining")
	source := resp.Header.Get("Docker-Ratelimit-Source")

	endposLimit := strings.IndexAny(limit, ";")
	endposRemaining := strings.IndexAny(remaining, ";")

	logger.Info("producer.getRateLimit", "ratelimit-limit", limit[:endposLimit])
	logger.Info("producer.getRateLimit", "ratelimit-remaining", remaining[:endposRemaining])
        logger.Info("producer.getRateLimit", "docker-ratelimit-source", source)
	
	return limit[:endposLimit], remaining[:endposRemaining], source, nil
}



func Put(interval int64) {
	var user, passwd string

	if environ.Env.ImagePullSecret != "" {
		user, passwd, _ = secret.GetSecret(environ.Env.Namespace, environ.Env.ImagePullSecret)
		logger.Info("producer.Put", "user", user)
	}
	for {
		token := getToken(repo, user, passwd)
		limit, remaining, source, err := getRateLimit(token, repo, tag)
		if err == nil {
			if environ.Env.ImagePullSecret != "" {
				// otherwise you get a meaningless uuid
				source = user
				logger.Info("producer.Put", "changed source", source)
			}
			data.Put(limit, remaining, source)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
