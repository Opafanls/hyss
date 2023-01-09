package main

import "github.com/Opafanls/hylan/server"

func main() {
	sv := server.NewHylanServer()

	sv.Start()
}
