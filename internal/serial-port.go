package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/busy-cloud/boat/db"
	"github.com/busy-cloud/boat/log"
	"github.com/busy-cloud/boat/mqtt"
	"github.com/god-jason/iot-master/link"
	"go.bug.st/serial"
)

func init() {
	db.Register(&SerialPort{})
}

type SerialPort struct {
	Id              string         `json:"id,omitempty" xorm:"pk"`
	Name            string         `json:"name,omitempty"`
	Description     string         `json:"description,omitempty"`
	PortName        string         `json:"port_name,omitempty"`                       //port, e.g. COM1 "/dev/ttySerial1".
	BaudRate        int            `json:"baud_rate,omitempty"`                       //9600 115200
	DataBits        int            `json:"data_bits"`                                 //5 6 7 8
	StopBits        int            `json:"stop_bits"`                                 //1 2
	ParityMode      int            `json:"parity_mode"`                               //0 1 2 NONE ODD EVEN
	PackDelay       int            `json:"pack_delay"`                                //打包延迟 ms
	Protocol        string         `json:"protocol,omitempty"`                        //通讯协议
	ProtocolOptions map[string]any `json:"protocol_options,omitempty" xorm:"json"`    //通讯协议参数
	Disabled        bool           `json:"disabled,omitempty"`                        //禁用
	Created         time.Time      `json:"created,omitempty,omitzero" xorm:"created"` //创建时间

	link.Status `xorm:"-"`
}

type SerialPortImpl struct {
	*SerialPort

	serial.Port

	buf    []byte
	opened bool
}

func NewSerialPort(l *SerialPort) *SerialPortImpl {
	c := &SerialPortImpl{
		SerialPort: l,
		buf:        make([]byte, 4096),
	}
	return c
}

func (c *SerialPortImpl) connect() (err error) {
	if c.Port != nil {
		_ = c.Port.Close()
	}

	//连接
	opts := serial.Mode{
		BaudRate: c.BaudRate,
		DataBits: c.DataBits,
		StopBits: serial.StopBits(c.StopBits),
		Parity:   serial.Parity(c.ParityMode),
	}

	log.Trace("create serial ", c.PortName, opts)
	c.Port, err = serial.Open(c.PortName, &opts)
	if err != nil {
		return err
	}

	c.Running = true

	go c.receive(c.Port)

	return
}

func (c *SerialPortImpl) Open() (err error) {
	if c.opened {
		return errors.New("already open")
	}
	c.opened = true

	//保持连接
	go c.keep()

	return c.connect()
}

func (c *SerialPortImpl) keep() {
	for c.opened {
		time.Sleep(time.Minute)

		if c.Port == nil {
			err := c.connect()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (c *SerialPortImpl) Close() error {
	c.opened = false

	//停止监听
	if c.Port != nil {
		err := c.Port.Close()
		c.Port = nil
		return err
	}

	return nil
}

func (c *SerialPortImpl) receive(conn serial.Port) {
	//从数据库中查询
	var l link.Link
	//xorm.ErrNotExist //db.Engine.Exist()
	//.Where("linker=", "serial-port").And("id=", id)
	has, err := db.Engine().ID(c.Id).Get(&l)
	if err != nil {
		_, _ = conn.Write([]byte(err.Error()))
		_ = conn.Close()
		return
	}

	//查不到
	if !has {
		l.Id = c.Id
		l.Linker = "serial-port"
		l.Protocol = c.Protocol //继承协议
		l.ProtocolOptions = c.ProtocolOptions
		_, err = db.Engine().InsertOne(&l)
		if err != nil {
			_, _ = conn.Write([]byte(err.Error()))
			_ = conn.Close()
			return
		}
	} else {
		if l.Disabled {
			_, _ = conn.Write([]byte("disabled"))
			_ = conn.Close()
			return
		}
	}

	//连接
	topicOpen := fmt.Sprintf("link/serial-port/%s/open", c.Id)
	mqtt.Publish(topicOpen, nil)
	if c.Protocol != "" {
		topicOpen = fmt.Sprintf("protocol/%s/link/serial-port/%s/open", c.Protocol, c.Id)
		mqtt.Publish(topicOpen, c.ProtocolOptions)
	}

	topicUp := fmt.Sprintf("link/serial-port/%s/up", c.Id)
	topicUpProtocol := fmt.Sprintf("protocol/%s/link/serial-port/%s/up", c.Protocol, c.Id)

	var cursor int //定位
	var n int
	var e error
	buf := make([]byte, 4096)
	var data []byte

	var delay *time.Timer

	if c.PackDelay > 5000 {
		c.PackDelay = 50
	}
	var delayMS = time.Duration(c.PackDelay) * time.Millisecond

	for {
		n, e = conn.Read(buf[cursor:])
		if e != nil {
			_ = conn.Close()
			break
		}
		data = buf[:cursor+n]

		//无延迟打包，直接上传
		if delayMS <= 0 {
			//转发
			mqtt.Publish(topicUp, data)
			if c.Protocol != "" {
				mqtt.Publish(topicUpProtocol, data)
			}
		}

		//以下为延时逻辑

		//移动指针
		cursor = cursor + n

		//首包延迟
		if delay == nil {
			delay = time.AfterFunc(delayMS, func() {
				//清空
				cursor = 0
				delay = nil

				mqtt.Publish(topicUp, data)
				if c.Protocol != "" {
					mqtt.Publish(topicUpProtocol, data)
				}
			})
		} else {
			//次包重置
			delay.Reset(delayMS)
		}
	}

	//下线
	topicClose := fmt.Sprintf("link/serial-port/%s/close", c.Id)
	mqtt.Publish(topicClose, e.Error())
	if c.Protocol != "" {
		topic := fmt.Sprintf("protocol/%s/link/serial-port/%s/close", c.Protocol, c.Id)
		mqtt.Publish(topic, e.Error())
	}

	c.Running = false

	c.Port = nil
}
