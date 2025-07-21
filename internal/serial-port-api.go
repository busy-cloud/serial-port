package internal

import (
	"github.com/busy-cloud/boat/api"
	"github.com/busy-cloud/boat/curd"
	"github.com/gin-gonic/gin"
	"go.bug.st/serial"
)

func init() {
	api.Register("GET", "serial-port/list", curd.ApiListHook[SerialPort](getPortsInfo))
	api.Register("POST", "serial-port/search", curd.ApiSearchHook[SerialPort](getPortsInfo))
	api.Register("POST", "serial-port/create", curd.ApiCreateHook[SerialPort](nil, func(m *SerialPort) error {
		_ = FromPort(m)
		return nil
	}))
	api.Register("GET", "serial-port/:id", curd.ApiGetHook[SerialPort](getPortInfo))

	api.Register("POST", "serial-port/:id", curd.ApiUpdateHook[SerialPort](nil, func(m *SerialPort) error {
		_ = FromPort(m)
		return nil
	}, "id", "name", "description", "port_name", "baud_rate", "data_bits", "stop_bits", "parity_mode", "pack_delay", "disabled", "protocol", "protocol_options"))

	api.Register("GET", "serial-port/:id/delete", curd.ApiDeleteHook[SerialPort](nil, func(m *SerialPort) error {
		_ = UnloadPort(m.Id)
		return nil
	}))

	api.Register("GET", "serial-port/:id/enable", curd.ApiDisableHook[SerialPort](false, nil, func(id any) error {
		_ = LoadPort(id.(string))
		return nil
	}))

	api.Register("GET", "serial-port/:id/disable", curd.ApiDisableHook[SerialPort](true, nil, func(id any) error {
		_ = UnloadPort(id.(string))
		return nil
	}))

	api.Register("GET", "serial-port/:id/open", portOpen)
	api.Register("GET", "serial-port/:id/close", portClose)

	api.Register("GET", "serial-port/:id/status", portStatus)

	api.Register("GET", "serial-port/ports", getPorts)
}

func getPortsInfo(ds []*SerialPort) error {
	for _, d := range ds {
		_ = getPortInfo(d)
	}
	return nil
}

func getPortInfo(d *SerialPort) error {
	l := ports.Load(d.Id)
	if l != nil {
		d.Status = l.Status
	}
	return nil
}

func portClose(ctx *gin.Context) {
	l := ports.Load(ctx.Param("id"))
	if l == nil {
		api.Fail(ctx, "找不到串口")
		return
	}

	err := l.Close()
	if err != nil {
		api.Error(ctx, err)
		return
	}

	api.OK(ctx, nil)
}

func portOpen(ctx *gin.Context) {
	l := ports.Load(ctx.Param("id"))
	if l != nil {
		err := l.Open()
		if err != nil {
			api.Error(ctx, err)
			return
		}
		api.OK(ctx, nil)
		return
	}

	err := LoadPort(ctx.Param("id"))
	if err != nil {
		api.Error(ctx, err)
		return
	}

	api.OK(ctx, nil)
}

func portStatus(ctx *gin.Context) {
	l := ports.Load(ctx.Param("id"))
	if l == nil {
		api.Fail(ctx, "找不到串口")
		return
	}

	api.OK(ctx, l.Status)
}

func getPorts(ctx *gin.Context) {
	ss, err := serial.GetPortsList()
	if err != nil {
		api.Error(ctx, err)
		return
	}

	api.OK(ctx, ss)
}
