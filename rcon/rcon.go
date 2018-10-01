/*
Copyright 2018 Google LLC
Copyright 2016 Kane York

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/*
Copyright (c) 2014 Alexander Harkness

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

// Package rcon is a RCON Client written in Go.
// It is designed to be easy to use and integrate into your own applications.
// Adapted from github.com/bearbin/mcgorcon
package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type packetType int32

// Client represents an active RCON connection.
//
// It is NOT safe for multithreaded SendCommand calls.
type Client struct {
	connection net.Conn
}

// header is the header of a Minecraft RCON packet.
type header struct {
	Size       int32
	RequestID  int32
	PacketType packetType
}

const packetTypeCommand packetType = 2
const packetTypeAuth packetType = 3
const requestIDBadLogin int32 = -1

// Dial connects to a RCON server and attempts to authenticate if 'pass' is not the empty string.
func Dial(host string, port int, pass string) (Client, error) {
	return DialTimeout(host, port, pass, 10*time.Second)
}

func DialTimeout(host string, port int, pass string, timeout time.Duration) (Client, error) {
	address := net.JoinHostPort(host, fmt.Sprint(port))
	conn, err := net.DialTimeout("tcp4", address, timeout)
	if err != nil {
		return Client{}, err
	}
	c := Client{connection: conn}
	if pass != "" {
		err = c.authenticate(pass)
		if err != nil {
			c.Close()
			return Client{}, err
		}
	}
	return c, nil
}

// Command runs command on the server and returns the result as a string.
// This function is not thread-safe (even though the protocol itself is).
func (c *Client) Command(command string) (string, error) {
	// Send the packet.
	head, payload, err := c.sendPacket(packetTypeCommand, []byte(command))
	if err != nil {
		return "", err
	}

	// Auth was bad, throw error.
	if head.RequestID == requestIDBadLogin {
		return "", errors.New("Bad auth, could not send command.")
	}
	return string(payload), nil
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.connection.Close()
}

// authenticate authenticates the user with the server.
func (c *Client) authenticate(password string) error {
	// Send the packet.
	head, _, err := c.sendPacket(packetTypeAuth, []byte(password))
	if err != nil {
		return err
	}

	// If the credentials were bad, throw error.
	if head.RequestID == requestIDBadLogin {
		return errors.New("Bad auth, could not authenticate.")
	}

	return nil
}

// sendPacket sends the binary packet representation to the server and returns the response.
func (c *Client) sendPacket(t packetType, p []byte) (header, []byte, error) {
	// Generate the binary packet.
	packet, err := packetise(t, p)
	if err != nil {
		return header{}, nil, err
	}

	// Send the packet over the wire.
	_, err = c.connection.Write(packet)
	if err != nil {
		return header{}, nil, err
	}
	// Receive and decode the response.
	head, payload, err := depacketise(c.connection)
	if err != nil {
		return header{}, nil, err
	}

	return head, payload, nil
}

// packetise encodes the packet type and payload into a binary representation to send over the wire.
func packetise(t packetType, p []byte) ([]byte, error) {
	// Generate a random request ID.
	pad := [2]byte{}
	length := int32(len(p) + 10)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, length)
	binary.Write(&buf, binary.LittleEndian, int32(0))
	binary.Write(&buf, binary.LittleEndian, t)
	binary.Write(&buf, binary.LittleEndian, p)
	binary.Write(&buf, binary.LittleEndian, pad)
	// Notchian server doesn't like big packets :(
	if buf.Len() >= 1460 {
		return nil, errors.New("Packet too big when packetising.")
	}
	// Return the bytes.
	return buf.Bytes(), nil
}

// depacketise decodes the binary packet into a native Go struct.
func depacketise(r io.Reader) (header, []byte, error) {
	head := header{}
	err := binary.Read(r, binary.LittleEndian, &head)
	if err != nil {
		return header{}, nil, err
	}
	payload := make([]byte, head.Size-8)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return header{}, nil, err
	}
	return head, payload[:len(payload)-2], nil
}
