package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	dc "github.com/samalba/dockerclient"
	log "github.com/sirupsen/logrus"
)

type Container struct {
	dc.Container
	Client *dc.DockerClient
	Info   *dc.ContainerInfo
}

func (c *Container) name() string {
	strs := strings.SplitAfter(c.Names[0], "/")
	return strs[len(strs)-1]
}

func (c *Container) runtimeConfig() (*dc.ContainerConfig, error) {
	image, err := c.Client.InspectImage(c.Image)
	if err != nil {
		return nil, err
	}

	config := c.Info.Config
	imageConfig := image.Config

	if config.WorkingDir == imageConfig.WorkingDir {
		config.WorkingDir = ""
	}

	if config.User == imageConfig.User {
		config.User = ""
	}

	if reflect.DeepEqual(config.Cmd, imageConfig.Cmd) {
		config.Cmd = nil
	}

	if reflect.DeepEqual(config.Entrypoint, imageConfig.Entrypoint) {
		config.Entrypoint = nil
	}

	config.Env = sliceSubtract(config.Env, imageConfig.Env)
	config.Labels = stringMapSubtract(config.Labels, imageConfig.Labels)
	config.Volumes = structMapSubtract(config.Volumes, imageConfig.Volumes)

	config.ExposedPorts = structMapSubtract(config.ExposedPorts, imageConfig.ExposedPorts)
	for p := range c.Info.HostConfig.PortBindings {
		config.ExposedPorts[p] = struct{}{}
	}

	return config, nil
}

func (c Container) hostConfig() *dc.HostConfig {
	hostConfig := c.Info.HostConfig

	for i, link := range hostConfig.Links {
		name := link[0:strings.Index(link, ":")]
		alias := link[strings.LastIndex(link, "/"):]
		hostConfig.Links[i] = fmt.Sprintf("%s:%s", name, alias)
	}

	return hostConfig
}

func (c *Container) start(image string) error {
	config, err := c.runtimeConfig()
	if err != nil {
		return err
	}

	config.HostConfig = *c.hostConfig()
	config.Image = image

	newContainerID, err := c.Client.CreateContainer(config, c.name(), nil)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"ID":    newContainerID,
		"Name":  c.name(),
		"Image": image,
	}).Info("New container created")

	return c.Client.StartContainer(newContainerID, nil)
}

func (c *Container) stop() error {
	signal := "SIGTERM"

	log.WithFields(log.Fields{
		"Name":   c.name(),
		"ID":     c.Id,
		"Signal": signal,
	}).Info("Killing container")

	if err := c.Client.KillContainer(c.Id, signal); err != nil {
		return err
	}

	c.waitForStop(30)

	log.WithField("ID", c.Id).Info("Removing container")

	if err := c.Client.RemoveContainer(c.Id, true, false); err != nil {
		return err
	}

	if err := c.waitForStop(30); err == nil {
		return fmt.Errorf("Container [%s] (%s) could not be removed", c.name(), c.Id)
	}

	return nil
}

func (c *Container) waitForStop(waitTime time.Duration) error {
	timeout := time.After(waitTime)

	for {
		select {
		case <-timeout:
			return nil
		default:
			if ci, err := c.Client.InspectContainer(c.Id); err != nil {
				return err
			} else if !ci.State.Running {
				return nil
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func (c *Container) shouldBeUpdated(repoName string) bool {
	return c.Info.Config.Image == repoName
}

func newContainer(client *dc.DockerClient, parent dc.Container) *Container {
	info, err := client.InspectContainer(parent.Id)
	if err != nil {
		return nil
	}

	return &Container{
		Client:    client,
		Container: parent,
		Info:      info,
	}
}
