package main

import (
	"copyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"text/tabwriter"
)

/*
 @Author: as
 @Date: Creat in 18:18 2022/3/19
 @Description: docker ps的实现
*/

func ListContainer() {
	// 找到存储容器信息的路径 /var/run/copyDocker/${}/
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL)-1]
	// 读取该文件下的所有文件
	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		logrus.Errorf("Read dir %s error %v.", dirURL, err)
		return
	}
	var containers []*container.ContainerInfo
	for _, v := range files {
		conInfo,err:=getContainerInfo(v)
		if err!=nil{
			logrus.Errorf("Get containerInfo error:",err)
			continue
		}
		containers = append(containers, conInfo)
	}
	w:=tabwriter.NewWriter(os.Stdout,12,1,3,' ',0)
	// 直接在控制台出信息
	fmt.Fprint(w,"ID\tNAME\tPID\tStatus\tCommand\tCreated\n")
	for _,itme:=range containers{
		// 打印出来
		fmt.Fprintf(w,"%s\t%s\t%s\t%s\t%s\t%s\n",
			itme.ID,
			itme.Name,
			itme.Pid,
			itme.Status,
			itme.Command,
			itme.CreatedTime,
		)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error:%v",err)
		return
	}
}

// 从文件获取存储的容器信息
func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	// Name
	containerName := file.Name()
	// 绝对路径
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName

	// 读取信息
	ctx, err := ioutil.ReadFile(configFileDir)
	if err!=nil{
		logrus.Errorf("Read file %s error: %v",configFileDir,err)
		return nil, err
	}
	var info container.ContainerInfo
	if err := json.Unmarshal(ctx, &info); err != nil {
		logrus.Errorf("Json unMarshal error:",err)
		return nil,err
	}
	return &info,nil
}
