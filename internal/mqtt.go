package internal

import (
	"strings"

	"github.com/busy-cloud/boat/mqtt"
)

func subscribe() {

	//订阅数据变化
	mqtt.Subscribe("link/serial-port/+/down", func(topic string, payload []byte) {
		ss := strings.Split(topic, "/")
		conn := ports.Load(ss[2])
		if conn != nil {
			_, _ = conn.Write(payload)
		}
	})

	//关闭连接
	//mqtt.Subscribe("link/serial-port/+/kill", func(topic string, payload []byte) {
	//	ss := strings.Split(topic, "/")
	//	conn := links.Load(ss[2])
	//	if conn != nil {
	//		_ = conn.Close()
	//	}
	//})
}
