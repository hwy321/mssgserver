package main

import (
	"fmt"
	"mssgserver/config"
)

func main()  {
	host := config.File.MustValue("login_server","host","127.0.0.1")
	fmt.Println(host)
}
