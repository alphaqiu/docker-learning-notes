package main

import (
	"log"
	dockerClient "github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/mount"
	"flag"
	"github.com/docker/docker/api/types/filters"
	"encoding/json"
	"time"
	"os"
	"io/ioutil"
)

var (
	swarmMangerURL string
	dockerAPIVersion string
	action string
	image string
)

func main() {
	flag.StringVar(&swarmMangerURL, "manager-url", "tcp://192.168.0.104:2376", "docker swarm manager address")
	flag.StringVar(&dockerAPIVersion, "docker-api-version", "1.25", "docker api version")
	flag.StringVar(&action, "action", "create", "action")
	flag.StringVar(&image, "image", "server:0.2", "image with tag")
	flag.Parse()

	log.Println("启动管理程序...")
	client, err := dockerClient.NewClient(swarmMangerURL, dockerAPIVersion, nil, nil)
	if err != nil {
		log.Printf("连接远程swarm manager失败, %v", err)
		return
	}

	defer client.Close()

	switch action {
	case "create":
		createService(client, 3, 2)
	case "find":
		findTasks(client)
	case "node":
		queryNode(client)
	default:
		log.Println("错误的执行命令")

	}

}

func createService(client *dockerClient.Client, maxAttempts, replicas uint64) (serviceID string, err error) {
	log.Println("创建服务规格...")
	service := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "server",
			Labels: map[string]string{
				"dzhyun.app": "server",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image: image,
				Labels: map[string]string{
					"dzhyun.task":"server",
				},
				Env: []string{"SWARM_SERVICE_ID=first"},
				Mounts: []mount.Mount{
					{
						Type: mount.TypeBind,
						Source: "/opt/go-program/src/alpha.co/server/docker-host",
						Target: "/etc/docker-host",
						ReadOnly: true,
					},
				},
			},
			// 限制重启次数，解决多次重启造成大量的任务被反复创建
			// 需要创建一个失败的app，让它反复重启看看 swarm service 停止后是怎样的表现
			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionOnFailure,
				MaxAttempts: &maxAttempts,
			},
		},
		Mode: swarm.ServiceMode{
			// 这个副本不需要
			Replicated: &swarm.ReplicatedService{
				Replicas: &replicas,
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Mode:swarm.ResolutionModeVIP,
			Ports:[]swarm.PortConfig{
				{
					// 命名也可以不需要
					Name: "http web port",
					Protocol:swarm.PortConfigProtocolTCP,
					TargetPort: 1533,
					PublishedPort: 1533,
				},
			},
		},
	}
	options := types.ServiceCreateOptions{}

	now := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Minute))
	defer cancel()

	var response types.ServiceCreateResponse
	response, err = client.ServiceCreate(ctx, service, options)
	if err != nil {
		log.Printf("创建服务失败, %v", err)
		return "", err
	}

	serviceID = response.ID
	log.Printf("服务ID：%s", serviceID)
	return serviceID, nil
}

// 在每台服务器上放置宿主机ip的文件，并被映射进容器，就不需要查询任务列表查找它所在节点并查找节点的ip
// 安装好服务以后，拿到服务id，查询服务对应的task，拿到task去拿对应的node信息中的ip地址，然后执行agent
func findTasks(client *dockerClient.Client) {
	filter := filters.NewArgs()
	filter.Add("service", "server")
	options := types.TaskListOptions{
		Filter:filter,
	}
	tasks, err := client.TaskList(context.Background(), options)
	if err != nil {
		log.Printf("获取任务列表失败, %v", err)
		return
	}

	log.Println("--*--*--*--*--*--*--*--*--*--*--*--*--")
	for _, task := range tasks {
		log.Printf("Task ID: %s, Task Slot: %d(Task State: %s)", task.ID, task.Slot, task.DesiredState)
		log.Printf("Node ID: %s, Service ID: %s", task.NodeID, task.ServiceID)
		log.Printf("Task Labels: %v", task.Labels)
		log.Println("--*--*--*--*--*--*--*--*--*--*--*--*--")
	}

}

type node struct {
	Status nodeStatus
}

type nodeStatus struct {
	State swarm.NodeState
	Addr string
}

func queryNode(client *dockerClient.Client) {
	_, raw, err := client.NodeInspectWithRaw(context.Background(), "do0nmebjvau8sc4wiwsmr5z06")
	if err != nil {
		log.Printf("查询节点信息失败, %v", err)
		return
	}

	//fmt.Printf("%s", raw)
	var n node
	if err = json.Unmarshal(raw, &n); err != nil {
		log.Printf("JSON解码失败, %v", err)
		return
	}

	log.Printf("节点IP地址: %s", n.Status.Addr)

	log.Println("-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-")
	log.Println("预备编译镜像的参数...")
	file, err := os.OpenFile("/opt/go-program/src/alpha.co/agent/agent.tar.gz", os.O_RDONLY, 0666)
	if err != nil {
		log.Printf("读取agent.tar.gz失败, %v", err)
		return
	}

	defer file.Close()

	options := types.ImageBuildOptions{
		Tags: []string{"agent:latest"},
		Remove: true,
		ForceRemove: true,
		PullParent: false,
		SuppressOutput: true,
		Labels: map[string]string{
			"dzhyun.app": "agent",
		},
	}

	log.Println("开始编译镜像...")
	response, err := client.ImageBuild(context.Background(), file, options)
	if err != nil {
		log.Printf("编译镜像失败, %v", err)
		return
	}

	res, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("读取返回的结果失败, %v", err)
		return
	}
	defer response.Body.Close()

	log.Printf("编译结果： %s", res)
}
