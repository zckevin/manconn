package main

import (
    "github.com/zckevin/manconn"
)

func main() {
    _ = manconn.NewDispatcher()
    <-make(chan struct{})
}
