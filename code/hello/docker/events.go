package main

import (
	dockerClient "github.com/docker/docker/client"
	"log"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/events"
	"os"
	"os/signal"
	"fmt"
	"time"
	"io"
)

const (
	EventEventType = "event"
	DateTimeLayout = "2006/01/02 15:04:05"
)

func main() {
	log.Println("执行初始化操作...")
	client, err := dockerClient.NewClient("tcp://192.168.0.104:2376", "1.25", nil, nil)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	filters := filters.NewArgs()
	filters.Add(EventEventType, "create")
	filters.Add(EventEventType, "destroy")
	filters.Add(EventEventType, "start")
	filters.Add(EventEventType, "stop")
	filters.Add(events.ContainerEventType, "test")

	msgCh, errCh := client.Events(context.Background(),
		types.EventsOptions{
			Filters:filters,
		})

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Kill, os.Interrupt)

	go func() {
		log.Println("系统启动，开始监控docker事件流.")
		for {
			select {
			case message := <-msgCh:
				printMessage(message)
			case err := <-errCh:
				if err == io.ErrUnexpectedEOF {
					log.Println("连接被关闭，退出.")
				} else {
					log.Printf("Error found when listening events on docker. cause: %v\n", err)
				}
				close(sigCh)
				return
			case <-sigCh:
				return
			}
		}
	}()

	<-sigCh
	log.Println("任务已执行完毕，程序退出.")
}

func printMessage(message events.Message) {
	timestamp := time.Unix(message.Time, 0)
	fmt.Printf(`-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-
 Action: %s, ID: %s,
 Type: %s, Status: %s
 From: %s, Time: %s
`,
		message.Action, message.ID, message.Type, message.Status,
		message.From, timestamp.Format(DateTimeLayout))

	fmt.Printf("  Actor ID: %s\n", message.Actor.ID)
	if len(message.Actor.Attributes) > 0 {
		for k, v := range message.Actor.Attributes {
			fmt.Printf("   %s = %s, ", k, v)
		}
		fmt.Println()
	}
	fmt.Println("-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-")
	fmt.Println()
}