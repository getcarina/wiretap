package main // import "github.com/getcarina/wiretap"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	dc "github.com/samalba/dockerclient"
	log "github.com/sirupsen/logrus"
)

type PushData struct {
	Images   []string
	PushedAt float32 `json:"pushed_at"`
	Pusher   string
}

type Repo struct {
	Name      string
	Namespace string
	RepoName  string `json:"repo_name"`
	Status    string
}

type jsonPayload struct {
	PushData PushData `json:"push_data"`
	Repo     Repo     `json:"repository"`
}

func updateContainers(j jsonPayload) error {
	tlsConfig, err := tlsConfig()
	if err != nil {
		return err
	}

	client, err := dc.NewDockerClient(os.Getenv("DOCKER_HOST"), tlsConfig)

	containers, err := client.ListContainers(true, true, "")
	if err != nil {
		return err
	}

	for _, c := range containers {
		container := newContainer(client, c)
		if container == nil {
			return fmt.Errorf("Container could not be inspected")
		}

		imageName := j.Repo.RepoName

		if container.shouldBeUpdated(imageName) {
			log.WithField("Image", imageName).Info("Pulling newer image")
			if err = client.PullImage(imageName, &dc.AuthConfig{}); err != nil {
				return err
			}
			if err = container.stop(); err != nil {
				return err
			}
			if err = container.start(imageName); err != nil {
				return err
			}
		}
	}

	return nil
}

func tokenIsValid(tokenVals []string) bool {
	validToken := os.Getenv("TOKEN")
	tokenIsValid := false
	for _, t := range tokenVals {
		if t == validToken {
			tokenIsValid = true
		}
	}
	return tokenIsValid
}

func listen(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" && tokenIsValid(r.URL.Query()["token"]) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, err.Error())
			return
		}

		jsonPayload := &jsonPayload{}
		err = json.Unmarshal(body, jsonPayload)
		if err != nil {
			writeError(w, err.Error())
			return
		}

		if err = updateContainers(*jsonPayload); err != nil {
			writeError(w, err.Error())
			return
		}

		fmt.Fprintln(w, "Okay")
	} else {
		writeError(w, "Invalid request")
	}
}

func writeError(w http.ResponseWriter, str string) {
	log.Error(str)
	w.WriteHeader(500)
	fmt.Fprintln(w, str)
}

func main() {
	http.HandleFunc("/listen", listen)
	http.ListenAndServe(":8000", nil)
}
