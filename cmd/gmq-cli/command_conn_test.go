package main

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/yosssi/gmq/mqtt/client"
	"github.com/yosssi/gmq/mqtt/packet"
)

const testAddress = "iot.eclipse.org:1883"

var errTest = errors.New("test")

type packetErr struct{}

func (p packetErr) WriteTo(w io.Writer) (int64, error) {
	return 0, errTest
}

func (p packetErr) Type() (byte, error) {
	return 0x00, errTest
}

func Test_commandConn_run_err(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	if err := cmd.run(); err == nil {
		t.Error("err => nil, want => not nil")
	}
}

func Test_commandConn_run(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.network = "tcp"
	cmd.address = testAddress

	if err := cmd.run(); err != nil {
		t.Error("err => %q, want => nil", err)
	}

	if err := disconnect(cmd.ctx); err != nil {
		t.Error("err => %q, want => nil", err)
	}
}

func Test_commandConn_waitCONNACK_connack(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Add(1)
	go cmd.waitCONNACK()

	cmd.ctx.connack <- struct{}{}

	cmd.ctx.wg.Wait()
}

func Test_commandConn_waitCONNACK_timeout(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.connackTimeout = 1

	cmd.ctx.wg.Add(1)
	go cmd.waitCONNACK()

	cmd.ctx.wg.Wait()
}

func Test_commandConn_waitCONNACK_timeout_disconnDefault(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.connackTimeout = 1

	cmd.ctx.disconn <- struct{}{}

	cmd.ctx.wg.Add(1)
	go cmd.waitCONNACK()

	cmd.ctx.wg.Wait()
}

func Test_commandConn_waitCONNACK_connackEnd(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Add(1)
	go cmd.waitCONNACK()

	cmd.ctx.connackEnd <- struct{}{}

	cmd.ctx.wg.Wait()
}

func Test_commandConn_receive_ReceiveErr_disconnecting(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.disconnecting = true

	cmd.ctx.wg.Add(1)
	go cmd.receive()

	cmd.ctx.wg.Wait()
}

func Test_commandConn_receive_ReceiveErr(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Add(1)
	go cmd.receive()

	cmd.ctx.wg.Wait()
}

func Test_commandConn_receive_ReceiveErr_default(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.disconn <- struct{}{}

	cmd.ctx.wg.Add(1)
	go cmd.receive()

	cmd.ctx.wg.Wait()
}

func Test_commandConn_receive(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:1883")
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
		return
	}

	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
		}
	}()

	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	if err := ctx.cli.Connect("tcp", "localhost:1883", nil); err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Add(1)
	go cmd.receive()

	time.Sleep(1 * time.Second)

	if err := disconnect(cmd.ctx); err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Wait()
}

func Test_commandConn_handle_err(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.handle(packetErr{})
}

func Test_commandConn_handle_default(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	p, err := packet.NewCONNACKFromBytes([]byte{0x20, 0x02}, []byte{0x00, 0x00})
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.connack <- struct{}{}

	cmd.handle(p)
}

func Test_commandConn_handle(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	p, err := packet.NewCONNACKFromBytes([]byte{0x20, 0x02}, []byte{0x00, 0x00})
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.handle(p)
}

func Test_commandConn_send_send(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	if err := ctx.cli.Connect("tcp", testAddress, nil); err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	cmd.ctx.wg.Add(1)
	go cmd.send()

	cmd.ctx.send <- packet.NewPINGREQ()

	time.Sleep(1 * time.Second)

	cmd.ctx.sendEnd <- struct{}{}

	cmd.ctx.wg.Wait()

	if err := disconnect(cmd.ctx); err != nil {
		t.Error("err => %q, want => nil", err)
	}
}

func Test_commandConn_send_keepAlive(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	if err := ctx.cli.Connect("tcp", testAddress, nil); err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	var keepAlive uint = 1
	cmd.connectOpts.KeepAlive = &keepAlive

	cmd.ctx.wg.Add(1)
	go cmd.send()

	time.Sleep(2 * time.Second)

	cmd.ctx.sendEnd <- struct{}{}

	cmd.ctx.wg.Wait()

	if err := disconnect(cmd.ctx); err != nil {
		t.Error("err => %q, want => nil", err)
	}
}

func Test_commandConn_sendPacket_disconnecting(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	cmd.ctx.disconnecting = true

	cmd.sendPacket(nil)
}

func Test_commandConn_sendPacket_disconn(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	cmd.sendPacket(nil)
}

func Test_commandConn_sendPacket_default(t *testing.T) {
	ctx := newContext()

	cmd, err := newCommandConn(nil, ctx)
	if err != nil {
		t.Errorf("err => %q, want => nil", err)
	}

	ctx.cli = client.New(nil)

	cmd.ctx.disconn <- struct{}{}

	cmd.sendPacket(nil)
}

func Test_newCommandConn(t *testing.T) {
	if _, err := newCommandConn([]string{"-not-exit-flag"}, newContext()); err != errCmdArgsParse {
		errorfErr(t, err, errCmdArgsParse)
	}
}
