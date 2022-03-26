package nsenter

/*
 @Author: as
 @Date: Creat in 20:48 2022/3/19
 @Description: 使用CGostens 的实现 /proc/$$/ns
*/

/*
#define _GNU_SOURCE

#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>

// 类似构造函数，只要这个包被使用，就会执行这个函数
__attribute__((constructor)) void enter_namespace(void) {
    char *copyDocker_pid;
    // 获取当前pid
    copyDocker_pid = getenv("copyDocker_pid");
    if (!copyDocker_pid) {
        return;
    }

    char *copyDocker_cmd;
    copyDocker_cmd = getenv("copyDocker_cmd");
    if (!copyDocker_cmd) {
        return;
    }
    int i;
    char nspath[1024];
    // 五种 namespace
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};

    for (i = 0; i < 5; i++) {
        // 拼接路径 /proc/$$/ns/ipc
        sprintf(nspath, "/proc/%s/ns/%s", copyDocker_pid, namespaces[i]);
        int fd = open(nspath, O_RDONLY);
        // 真正调用 setns 系统调用进入对应的 Namespace
        if (setns(fd, 0) != -1) {
            //fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
        }
        close(fd);
    }
    // 在进入的 Namespace 中执行指定的命令
    int res = system(copyDocker_cmd);

    exit(0);
    return;
}
*/
import "C"
