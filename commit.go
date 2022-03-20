package main

import (
	"copyDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
)

/*
 @Author: as
 @Date: Creat in 19:51 2022/3/18
 @Description: 镜像打包的实现
*/

// 打包函数具体方法的实现
func commitContainer(containerName, imageName string) {
	mntUrl := fmt.Sprintf(container.MntURL, containerName) + "/"
	imageTar := container.RootURL + "/" + imageName + ".tar"
	logrus.Infof("tar image: %s", imageTar)
	// tar -czf /root/${}.tar -C .
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntUrl, ".").
		CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder %s error:%v", container.MntURL, err)
	}
}
