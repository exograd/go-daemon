package main

import (
	"github.com/exograd/go-daemon/daemon"
)

func main() {
	daemon.Run("example", "go-daemon example", NewService())
}
