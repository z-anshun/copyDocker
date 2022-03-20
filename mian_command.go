package main

import (
	"copyDocker/cgroups/subsystems"
	"copyDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

/*
 @Author: as
 @Date: Creat in 23:14 2022/3/13
 @Description: copyDocker
*/
// docker run imageName  -ti -name containerName
var runCommand = cli.Command{
	Name:  "run", // 命令名
	Usage: "Create a container with Namespace and Cgroups (run -ti [command])",
	// 定义run时 Command 参数
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "use ti",
		},
		// -d 标签，detach 表示后台运行
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		// 添加 -v 的标签
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		// -name 提供容器 name
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringSliceFlag{
			Name: "e",
			Usage: "set env",
		},
	},
	// 正在 run 的函数
	// 1. 判断用户是否包含 command
	// 2. 获取用户指定的 command
	// 3. 调用 run function 去启动容器
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		var cmdArray []string
		for _, arg := range ctx.Args() {
			cmdArray = append(cmdArray, arg)
		}

		tty := ctx.Bool("ti")
		detach := ctx.Bool("d")

		// terminal 和 detach 不能共存
		if tty && detach {
			return fmt.Errorf("ti and d paramter can not both provited")
		}
		logrus.Infof("CreateTry %v", tty)
		volume := ctx.String("v")

		// 将容器名传递下去
		containerName := ctx.String("name")

		imageName:=cmdArray[0]
		cmdArray=cmdArray[1:]

		envSlice:=ctx.StringSlice("e")

		Run(tty, cmdArray, volume, &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("m"),
			CpuShare:    ctx.String("cpuset"),
			CpuSet:      ctx.String("cpushare"),
		}, containerName,imageName,envSlice)
		return nil
	},
}

// 定义 initCommand 的具体操作，只限于内部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's proc in container. Do not call it outside.",
	/*
		1. 获取传递过来的 command 参数
		2. 执行容器初始化操作
	*/
	Action: func(ctx *cli.Context) error {
		logrus.Infof("init come on")
		logrus.Infof("send in command %s", ctx.Args())
		return container.RunContainerInitProcess()
	},
}

// 定义打包镜像的 commitCommand ,传入对应的镜像名
// docker commit containerName imgaeName
var commieCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("Missing container name and image name")
		}
		containerName:=ctx.Args().Get(0)
		imageName := ctx.Args().Get(1)
		commitContainer(containerName,imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(ctx *cli.Context) error {
		// 列出所有的 containerInfo
		ListContainer()
		return nil
	},
}

// docker logs
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("Please input container name")
		}
		containerName := ctx.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

// docker exec 进入容器
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(ctx *cli.Context) error {
		// 环境变量 copyDocker_pid 的值
		if os.Getenv(ENV_EXEC_PID) != "" {
			logrus.Infof("pid callback pid %s", os.Getpid())
			return nil
		}
		// 命令格式 copyDocker exec containerName cmd
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("Missing container name or command.")
		}
		containerName := ctx.Args().Get(0)
		var commandArray []string

		// 除了容器名之外，其它参数都当作需要执行的命令执行
		for _, arg := range ctx.Args().Tail() {
			commandArray = append(commandArray, arg)
		}
		// 执行命令
		ExecContainer(containerName, commandArray)
		return nil
	},
}

// docker stop
var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "Stop a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := ctx.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := ctx.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}
