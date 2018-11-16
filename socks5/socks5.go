/*
MIT License

Copyright (c) 2018 KIM KeepInMind Gmbh/srl

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Package socks5 provides a SOCKS5 server implementation. See RFC 1928
// for protocol specification.
package socks5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/booster-proj/proxy/dialer"
	"github.com/booster-proj/proxy/transmit"
	"upspin.io/log"
)

const (
	socks5Version = uint8(5)
)

// Possible METHOD field values
const (
	socks5MethodNoAuth              = uint8(0)
	socks5MethodGSSAPI              = uint8(1)
	socks5MethodUsernamePassword    = uint8(2)
	socks5MethodNoAcceptableMethods = uint8(0xff)
)

// Possible CMD field values
const (
	socks5CmdConnect   = uint8(1)
	socks5CmdBind      = uint8(2)
	socks5CmdAssociate = uint8(3)
)

// Possible REP field values
const (
	socks5RespSuccess                 = uint8(0)
	socks5RespGeneralServerFailure    = uint8(1)
	socks5RespConnectionNotAllowed    = uint8(2)
	socks5RespNetworkUnreachable      = uint8(3)
	socks5RespHostUnreachable         = uint8(4)
	socks5RespConnectionRefused       = uint8(5)
	socks5RespTTLExpired              = uint8(6)
	socks5RespCommandNotSupported     = uint8(7)
	socks5RespAddressTypeNotSupported = uint8(8)
	socks5RespUnassigned              = uint8(9)
)

const (
	// FieldReserved should be used to fill fields marked as reserved.
	socks5FieldReserved = uint8(0x00)
)

const (
	// AddrTypeIPV4 is a version-4 IP address, with a length of 4 octets
	socks5IP4 = uint8(1)

	// AddrTypeFQDN field contains a fully-qualified domain name. The first
	// octet of the address field contains the number of octets of name that
	// follow, there is no terminating NUL octet.
	socks5FQDN = uint8(3)

	// AddrTypeIPV6 is a version-6 IP address, with a length of 16 octets.
	socks5IP6 = uint8(4)
)

var (
	supportedMethods = []uint8{socks5MethodNoAuth}
)

// Proxy represents a SOCKS5 proxy server implementation.
type Proxy struct {
	dialer.Dialer
	port int
}

// New returns a new Proxy instance.
func New() *Proxy {
	return &Proxy{
		Dialer: dialer.Default,
	}
}

// DialWith make the receiver use d for dialing TCP connections, if d != nil.
func (s *Proxy) DialWith(d dialer.Dialer) {
	if d != nil {
		s.Dialer = d
	}
}

// ListenAndServe accepts and handles TCP connections
// using the SOCKS5 protocol.
func (s *Proxy) ListenAndServe(ctx context.Context, port int) error {
	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}
	defer ln.Close()

	errc := make(chan error)
	defer close(errc)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errc <- fmt.Errorf("ListenAndServe: cannot accept conn: %v", err)
				return
			}

			go func() {
				if err = s.Handle(ctx, conn); err != nil {
					log.Error.Println(err)
				}
			}()
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		ln.Close()
		<-errc // wait for listener to return
		return ctx.Err()
	}
}

func (s *Proxy) Protocol() string {
	return "socks5"
}

// Handle performs the steps required to be SOCKS5 compliant.
// See RFC 1928 for details.
//
// Should run in its own go routine, closes the connection
// when returning.
func (s *Proxy) Handle(ctx context.Context, conn net.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	// method sub-negotiation phase
	if err := s.Negotiate(conn); err != nil {
		return err
	}

	// request details

	// len is just an estimation
	buf := make([]byte, 6+net.IPv4len)

	if _, err := io.ReadFull(conn, buf[:3]); err != nil {
		return errors.New("Handle: unable to read request: " + err.Error())
	}

	v := buf[0]   // protocol version
	cmd := buf[1] // command to execute
	_ = buf[2]    // reserved field

	// Check version number
	if v != socks5Version {
		return errors.New("Handle: unsupported version: " + string(v))
	}

	target, err := ReadAddress(conn)
	if err != nil {
		return err
	}

	log.Debug.Printf("Handle: performing [%v] to: %v", prettyCmd(cmd), target)

	var tconn net.Conn
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch cmd {
	case socks5CmdConnect:
		tconn, err = s.Connect(_ctx, conn, target)
	case socks5CmdAssociate:
		tconn, err = s.Associate(_ctx, conn, target)
	case socks5CmdBind:
		tconn, err = s.Bind(_ctx, conn, target)
	default:
		return errors.New("Handle: unexpected CMD(" + strconv.Itoa(int(cmd)) + ")")
	}
	if err != nil {
		return err
	}
	defer tconn.Close()

	// start proxying
	start := time.Now()
	ptp := fmt.Sprintf("%v <->  %v (~> %v)", conn.LocalAddr(), tconn.RemoteAddr(), target)

	log.Debug.Printf("Handle: %v => data transmission [BEGIN]", ptp)
	defer func() {
		end := time.Now()
		d := end.Sub(start)
		log.Debug.Printf("Handle: %v => data transmission [END] d(%v)", ptp, d)
	}()

	ctx = transmit.NewContext(ctx, time.Minute*10, 1500)
	if err = transmit.Data(ctx, conn, tconn); err != nil {
		return errors.New(ptp + ": " + err.Error())
	}
	return nil
}

func prettyCmd(cmd uint8) string {
	switch cmd {
	case socks5CmdConnect:
		return "Connect"
	case socks5CmdAssociate:
		return "Associate"
	case socks5CmdBind:
		return "Bind"
	default:
		return "Undefined"
	}
}

// ReadAddress reads hostname and port and converts them
// into its string format, properly formatted.
//
// r expects to read one byte that specifies the address
// format (1/3/4), followed by the address itself and a
// 16 bit port number.
//
// addr == "" only when err != nil.
func ReadAddress(r io.Reader) (addr string, err error) {
	host, err := ReadHost(r)
	if err != nil {
		return "", err
	}

	port, err := ReadPort(r)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, port), nil
}

// ReadHost deals with the host part of ReadAddress.
func ReadHost(r io.Reader) (string, error) {
	// cap is just an estimantion
	buf := make([]byte, 0, 1+net.IPv6len)
	buf = buf[:1]

	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("ReadHost: unable to read address type: " + err.Error())
	}

	atype := buf[0] // address type

	bytesToRead := 0
	switch atype {
	case socks5IP4:
		bytesToRead = net.IPv4len
	case socks5IP6:
		bytesToRead = net.IPv6len
	case socks5FQDN:
		_, err := io.ReadFull(r, buf[:1])
		if err != nil {
			return "", errors.New("ReadHost: failed to read domain length: " + err.Error())
		}
		bytesToRead = int(buf[0])
	default:
		return "", errors.New("ReadHost: got unknown address type " + strconv.Itoa(int(atype)))
	}

	if cap(buf) < bytesToRead {
		buf = make([]byte, bytesToRead)
	} else {
		buf = buf[:bytesToRead]
	}
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("ReadHost: failed to read address: " + err.Error())
	}

	var host string
	if atype == socks5FQDN {
		host = string(buf)
	} else {
		host = net.IP(buf).String()
	}

	return host, nil
}

// ReadPort deals with the port part of ReadAddress.
func ReadPort(r io.Reader) (string, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", errors.New("ReadPort: " + err.Error())
	}

	port := int(buf[0])<<8 | int(buf[1])
	return strconv.Itoa(port), nil
}

// EncodeAddressBinary expects as input a canonical host:port address and
// returns the binary representation as speccified in the socks5 protocol (RFC 1928).
// Booster uses the same encoding.
func EncodeAddressBinary(addr string) ([]byte, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("EncodeAddressBinary: unrecognised address format : " + addr + " : " + err.Error())
	}

	hbuf, err := EncodeHostBinary(host)
	if err != nil {
		return nil, err
	}

	pbuf, err := EncodePortBinary(port)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 0, len(hbuf)+len(pbuf))
	buf = append(buf, hbuf...)
	buf = append(buf, pbuf...)

	return buf, nil
}

// EncodeHostBinary encodes a canonical host (IPv4, IPv6, FQDN) into a
// byte slice. Format follows RFC 1928.
func EncodeHostBinary(host string) ([]byte, error) {
	buf := make([]byte, 0, 1+len(host)) // 1 if fqdn (address size)

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, socks5IP4)
			ip = ip4
		} else {
			buf = append(buf, socks5IP6)
		}
		buf = append(buf, ip...)
	} else {
		if len(host) > 255 {
			return nil, errors.New("EncodeHostBinary: destination host name too long: " + host)
		}
		buf = append(buf, socks5FQDN)
		buf = append(buf, byte(len(host)))
		buf = append(buf, host...)
	}

	return buf, nil
}

// EncodePortBinary encodes a canonical port into 2 bytes.
// Format follows RFC 1928.
func EncodePortBinary(port string) ([]byte, error) {
	buf := make([]byte, 0, 2)
	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("EncodePortBinary: failed to parse port number: " + port)
	}
	if p < 1 || p > 0xffff {
		return nil, errors.New("EncodePortBinary: port number out of range: " + port)
	}

	buf = append(buf, byte(p>>8), byte(p))
	return buf, nil
}
