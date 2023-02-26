package main

import (
	"fmt"

	"github.com/ivolkoff/tcp-pow-go/internal/client"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/config"
)

func main() {
	fmt.Println("start client")

	// loading config from file and env
	configInst, err := config.Load("config/config.json")
	if err != nil {
		fmt.Println("error load config:", err)
		return
	}

	// run client
	cli := client.NewClient(&client.Dependency{
		Config: configInst,
	})
	address := fmt.Sprintf("%s:%d", configInst.ServerHost, configInst.ServerPort)
	if err := cli.Run(address); err != nil {
		fmt.Println("client error:", err)
	}
}
