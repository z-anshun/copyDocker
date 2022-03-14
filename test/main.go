package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
)

// "fork/exec /proc/self/exe: no such file or directory"
func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		cli.Command{
			Name: "test",
			Action: func(ctx *cli.Context) error {
				cmd:=exec.Command("/proc/self/exe", "try")
				cmd.Stdout=os.Stdout
				cmd.Stdin=os.Stdin
				cmd.Stderr=os.Stderr
				if err := cmd.Start(); err != nil {
					panic(err)
				}
				fmt.Fprintln(os.Stdout,"Father:",os.Getpid())

				cmd.Wait()
				return nil
			},
		},
		cli.Command{
			Name: "try",
			Action: func(ctx *cli.Context) error {
				fmt.Println("this is children ",os.Getpid())
				return nil
			},
		},
	}

	// 进行执行
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func fork(){

}