# Docker Remote API 管理远程机器

## 创建可以管理远程机器的agent

### Dockerfile

```dockerfile
FROM ubuntu:latest
MAINTAINER alpha <alphaqiu@gmail.com>

WORKDIR /opt
RUN mkdir -p /opt/scripts
COPY prerequest.sh /opt/scripts/prerequest.sh
RUN chmod +x /opt/scripts/prerequest.sh

ENTRYPOINT ["/opt/scripts/prerequest.sh"]
```

```bash
#!/usr/bin/env bash

# step 1 create dir
mkdir -p /tmp/${SERVICE_DIR}

# step 2 link to the docker output
# mock /var/lib/docker/volumes/xxx -> /tmp/dzhyun/xxx
ln -s ${DOCKER_VOLUME_DIR} /tmp/${SERVICE_DIR}
```
### 打包成gzip压缩格式

```bash
tar zcf agent.tar.gz Dockerfile prerequest.sh
```

压缩包内根目录下必须包含Dockerfile文件，和编译所需要的全部文件。

## 创建管理程序

### 编译镜像

```go
    log.Println("-*-*-*-*-*-*-*-*-*-*-*-*-*-*-*-")
	log.Println("预备编译镜像的参数...")
	file, err := os.OpenFile("/path/to/the/agent.tar.gz", os.O_RDONLY, 0666)
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
```

`client.ImageBuild`会通过Docker Remote API向远程机器POST tar stream，在远端服务器上编译Docker 镜像。

编译完成后返回

`编译结果： {"stream":"sha256:f3b724507fd77ac4cac5a8f6bb9dd89fb9513195c6189d7b61e80ec2b4cec167\n"}`

### 执行操作

```bash
docker run --rm -d -e DOCKER_VOLUME_DIR=/var/lib/docker/volumes/app.xxx \
-e dzhyun/app.xxx -v /opt/dzhyun:/opt/dzhyun \
-v /var/lib/docker/volumes/app.xxx:/var/lib/docker/volumes/app.xxx \
--priviledged=true \
agent:latest
```

