package main

import (
	"github.com/Opafanls/hylan/server/srv"
)

func main() {
	sv := srv.NewHylanServer()
	sv.Start()
}
