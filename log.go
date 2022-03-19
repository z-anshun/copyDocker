package main

import (
	"copyDocker/container"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

/*
 @Author: as
 @Date: Creat in 20:29 2022/3/19
 @Description: 查看容器 log 的具体实现
*/

func logContainer(containerName string) {
	// 对应文件夹的位置
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// /var/run/copyDocker/${}/container.log
	logFileLocation := dirURL + container.ContainerLogFile
	// 日志文件的打开
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		logrus.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}

	// read
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}

	fmt.Fprint(os.Stdout, string(content))

}
