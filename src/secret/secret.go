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

package secret

import (
	"fmt"
	"context"
	"errors"
	"encoding/json"
	"path/filepath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	"docker-rate-limit/environ"
	"docker-rate-limit/logger"
)

type AuthsT struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

type DockerT struct {
	Auths AuthsT `json:"https://index.docker.io/v1/"`
}

type SecretT struct {
	Docker DockerT `json:"auths"`
}

func newClientSet() (*kubernetes.Clientset, error) {
        var config *rest.Config
        var err error
        var kubeconfig string

        if environ.Env.Standalone {
                if home := homedir.HomeDir(); home != "" {
                        kubeconfig = filepath.Join(home, ".kube", "config")
                } else {
                        logger.Error("secret.newClientSet could not find kubeconfig")
                        return nil, fmt.Errorf("could not find kubeconfig")
                }

                config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
                if err != nil {
                        logger.Error("secret.newClientSet", "clientcmd.BuildConfigFromFlags", err)
                        return nil, fmt.Errorf("error loading given file: %w", err)
                }
        } else {
                config, err = rest.InClusterConfig()
                if err != nil {
                        logger.Error("secret.newClientSet", "rest.InClusterConfig", err)
			return nil, err
                }
        }

        return kubernetes.NewForConfig(config)
}

func GetSecret(namespace string, name string) (string, string, error) {
	var dat SecretT

	clientset, _ := newClientSet()
	secretsClient := clientset.CoreV1().Secrets(namespace)
	secret, err := secretsClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		logger.Error("secret.GetSecret", "namespace", namespace, "name", name, "err", err)
		return "", "", err
	}

	value, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		logger.Error("secret.GetSecret key .dockerconfigjson not found in secret")
		return "", "", errors.New("key .dockerconfigjson not found in secret")
	}

	err = json.Unmarshal(value, &dat)
	if err != nil {
		logger.Error("secret.GetSecret unmarshal", "error", err)
		return "", "", err
	} else {
		return dat.Docker.Auths.Username, dat.Docker.Auths.Password, nil
	}
	return "", "", nil
}
