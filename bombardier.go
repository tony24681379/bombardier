package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/tony24681379/bombardier/lib"
)

func main() {
	cfg, err := lib.Parser.Parse(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(lib.ExitFailure)
	}
	bombardier, err := lib.NewBombardier(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(lib.ExitFailure)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		bombardier.Barrier.Cancel()
	}()
	bombardier.Bombard()
	if bombardier.Conf.PrintResult {
		bombardier.PrintStats()
	}
}
