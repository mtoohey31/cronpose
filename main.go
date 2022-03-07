package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

func connect() *net.UnixConn {
	addr, err := net.ResolveUnixAddr("unix", "/var/run/docker.sock")
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	return conn
}

func makeReqest(conn *net.UnixConn, method string, url string, body io.Reader, v interface{}) int {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	err = req.Write(conn)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	err = json.Unmarshal(resBody, &v)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	return res.StatusCode
}

func main() {
	conn := connect()
	var containers []types.Container
	status := makeReqest(conn, http.MethodGet, "http://localhost/containers/json?all=true&filters=%7B%22label%22%3A%5B%22com.mtoohey.cronpose%22%5D%7D", nil, &containers)
	if status != http.StatusOK {
		log.Error().Msgf("response status %d while listing containers", status)
	}
	c := cron.New()
	for _, currContainer := range containers {
		// NOTE: this is required because the loop values are passed by reference
		// otherwise, and go reuses the same loop variable, and this will result in
		// the cron function running multiple times for the last container instead
		// of once for each
		container := currContainer

		schedule, ok := container.Labels["cronpose"]
		if !ok {
			schedule = container.Labels["com.mtoohey.cronpose"]
		}
		log.Info().Msgf("detected container %v with schedule %s", container.Names, schedule)
		_, err := c.AddFunc(schedule, func() {
			log.Info().Msgf("starting container %v", container.Names)
			status := makeReqest(conn, http.MethodPost, fmt.Sprintf("http://localhost/containers/%s/start", container.ID), nil, nil)
			if status != http.StatusNoContent && status != http.StatusNotModified {
				log.Error().Msgf("response status %d while starting container %v", status, container.Names)
			}
		})
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
	c.Start()
	wait := make(chan struct{})
	for {
		<-wait
	}
}
