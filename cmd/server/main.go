package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ivolkoff/tcp-pow-go/internal/pkg/cache"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/clock"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/config"
	"github.com/ivolkoff/tcp-pow-go/internal/server"
)

func main() {
	fmt.Println("start server")

	// loading config from file and env
	configInst, err := config.Load("config/config.json")
	if err != nil {
		fmt.Println("error load config:", err)
		return
	}

	// init context to pass config down
	ctx := context.Background()

	cacheInst, err := cache.InitRedisCache(ctx, configInst.CacheHost, configInst.CachePort)
	if err != nil {
		fmt.Println("error init cache:", err)
		return
	}

	// run server
	ser := server.NewServer(&server.Dependency{
		Config: configInst,
		Clock:  new(clock.SystemClock),
		Cache:  cacheInst,
		Rand:   rand.New(rand.NewSource(0)),
	})
	address := fmt.Sprintf("%s:%d", configInst.ServerHost, configInst.ServerPort)
	if err := ser.Run(address); err != nil {
		fmt.Println("server error:", err)
	}
}
