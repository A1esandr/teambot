package main

import "teambot/app"

func main() {
	srv := app.NewApp()
	srv.Start()
}
