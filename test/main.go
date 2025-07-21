package main

import (
	"os"
	"os/signal"
	"syscall"

	_ "github.com/busy-cloud/boat/apis" //boat的基本接口
	"github.com/busy-cloud/boat/apps"
	"github.com/busy-cloud/boat/boot"
	_ "github.com/busy-cloud/boat/broker"
	"github.com/busy-cloud/boat/log"
	"github.com/busy-cloud/boat/web"
	_ "github.com/busy-cloud/modbus" //测试一个协议
	_ "github.com/busy-cloud/serial-port"
	_ "github.com/god-jason/iot-master"
	"github.com/spf13/viper"
)

func init() {
	//测试
	apps.Pages().AddDir("pages")
}

func main() {
	viper.SetConfigName("serial-port")
	//e := viper.SafeWriteConfig()
	////e := viper.WriteConfig()
	//if e != nil {
	//	log.Error(e)
	//}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs

		//关闭web，出发
		_ = web.Shutdown()
	}()

	//安全退出
	defer boot.Shutdown()

	err := boot.Startup()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = web.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
