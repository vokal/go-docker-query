package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

const (
	ApiVersion        = 1.9
	DefaultUnixSocket = "/var/run/docker.sock"
	DefaultProtocol   = "unix"
	Version           = "1.1.0"
)

type Container struct {
	Id    string
	Names []string
}

type ContainerDetails struct {
	NetworkSettings struct {
		IPAddress string
	}
}

type Client struct {
	proto string
	addr  string
}

func NewClient() *Client {
	return &Client{
		proto: DefaultProtocol,
		addr:  DefaultUnixSocket,
	}
}

func (c *Client) Do(method, path string) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("/v%g%s", ApiVersion, path), nil)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{}
	req.Header.Set("User-Agent", "Docker-Client/"+Version)
	req.Header.Set("Content-Type", "application/json")
	req.Host = c.addr

	dial, err := net.Dial(c.proto, c.addr)
	if err != nil {
		return nil, err
	}

	conn := httputil.NewClientConn(dial, nil)
	resp, err := conn.Do(req)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	switch resp.StatusCode {
	case 404, 403, 401, 400:
		defer resp.Body.Close()
		return nil, errors.New("Error communicating with Docker")
	}

	return resp.Body, nil
}

func Containers() ([]Container, error) {
	c := NewClient()
	resp, err := c.Do("GET", "/containers/json")
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var result []Container
	err = json.NewDecoder(resp).Decode(&result)
	return result, err
}

func Inspect(containerId string) (*ContainerDetails, error) {
	c := NewClient()
	resp, err := c.Do("GET", fmt.Sprintf("/containers/%s/json", containerId))
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var result ContainerDetails
	err = json.NewDecoder(resp).Decode(&result)
	return &result, err
}

func FindContainerIp(name string) string {
	containers, err := Containers()
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range containers {
		for _, n := range c.Names {
			if n == name {
				container, err := Inspect(c.Id)
				if err != nil {
					log.Fatal(err)
				}

				return container.NetworkSettings.IPAddress
			}
		}
	}

	// TODO: Should return (string, error)
	return ""
}
