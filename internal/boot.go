package internal

import (
	"time"

	"github.com/busy-cloud/boat/boot"
)

func init() {
	boot.Register("serial-port", &boot.Task{
		Startup:  Startup,
		Shutdown: Shutdown,
		Depends:  []string{"log", "mqtt", "database"},
	})
}

func Startup() error {

	//订阅通知
	subscribe()

	//5秒后再启动，先让其他准备好
	time.AfterFunc(time.Second*5, StartPorts)

	return nil
}

func Shutdown() error {

	StopPorts()

	return nil
}
