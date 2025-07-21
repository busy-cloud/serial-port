package internal

import (
	"fmt"

	"github.com/busy-cloud/boat/db"
	"github.com/busy-cloud/boat/lib"
	"github.com/busy-cloud/boat/log"
)

var ports lib.Map[SerialPortImpl]

func StartPorts() {
	//加载连接器
	var ps []*SerialPort
	err := db.Engine().Find(&ps)
	if err != nil {
		log.Error(err)
		return
	}
	for _, p := range ps {
		if p.Disabled {
			log.Info("port %s is disabled", p.Id)
			continue
		}
		err := FromPort(p)
		if err != nil {
			log.Error(err)
		}
	}
}

func StopPorts() {
	ports.Range(func(name string, client *SerialPortImpl) bool {
		_ = client.Close()
		return true
	})
}

func FromPort(m *SerialPort) error {
	port := NewSerialPort(m)

	//保存
	val := ports.LoadAndStore(port.Id, port)
	if val != nil {
		err := val.Close()
		if err != nil {
			log.Error(err)
		}
	}

	//启动
	err := port.Open()
	if err != nil {
		return err
	}

	return nil
}

func LoadPort(id string) error {
	var l SerialPort
	has, err := db.Engine().ID(id).Get(&l)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("serial port %s not found", id)
	}

	return FromPort(&l)
}

func UnloadPort(id string) error {
	val := ports.LoadAndDelete(id)
	if val != nil {
		return val.Close()
	}
	return nil
}
