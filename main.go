package main

/*
 @Author: as
 @Date: Creat in 22:59 2022/3/13
 @Description: copyDocker
*/

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = "This is a simple docker. The purpose to learn from 'mydocker'"

func main() {
	// 实例化一个命令行程序
	app := cli.NewApp()
	app.Usage = usage       // 设置程序用途的描述
	app.Version = "1.0.0"   // 版本号
	app.Name = "copyDocker" // 程序名

	// 设置 commands
	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commieCommand,
		listCommand,
		logCommand,
		stopCommand,
		execCommand,
		removeCommand,
	}

	app.Before = func(ctx *cli.Context) error {
		logrus.SetFormatter(&logrus.JSONFormatter{})

		logrus.SetOutput(os.Stdout)
		return nil
	}

	// 进行执行
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
