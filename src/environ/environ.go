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

package environ

import (
        "go-simpler.org/env"
	"docker-rate-limit/logger"
	"os"
)

type EnvT struct {
	Standalone      bool   `env:"STANDALONE" default:"false"`
	RefreshSeconds  int64  `env:"REFRESH_SECONDS" default:"300"`
	PortNumber      int64  `env:"PORT_NUMBER" default:"9900"`
        ImagePullSecret string `env:"IMAGEPULL_SECRET"`
        Namespace       string `env:"NAMESPACE"`
}

var Env EnvT

func Load() error {
	if err := env.Load(&Env, nil); err != nil {
                logger.Error("environ.Load", "err", err)
		os.Exit(1)
        }
	logger.Info("environ.Load loaded environment", "STANDALONE", Env.Standalone, "REFRESH_SECONDS", Env.RefreshSeconds, "PORT_NUMBER", Env.PortNumber, "IMAGEPULL_SECRET", Env.ImagePullSecret, "NAMESPACE", Env.Namespace)
	return nil
}
