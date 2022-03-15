package main

import (
	"copyDocker/cgroups/subsystems"
	"copyDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

/*
 @Author: as
 @Date: Creat in 23:14 2022/3/13
 @Description: copyDocker
*/

var runCommand = cli.Command{
	Name:  "run", // 命令名
	Usage: "Create a container with Namespace and Cgroups (run -ti [command])",
	// 定义run时 Command 参数
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "use ti",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
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
		Run(tty, cmdArray, &subsystems.ResourceConfig{
			MemoryLimit: ctx.String("m"),
		})
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
		cmd := ctx.Args().Get(0)
		logrus.Infof("command %s", cmd)

		return container.RunContainerInitProcess()
	},
}
