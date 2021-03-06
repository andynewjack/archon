/*
* Archon PSO Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
*
* Handles the connection initialization and management for connected
* ships. This module handles all of its own connection logic since the
* shipgate protocol differs from the way game clients are processed.
 */
package main

import (
	// 	"crypto/tls"
	// 	"errors"
	// "fmt"
	// 	"io"
	"net"
	// 	"os"
	// 	"runtime/debug"
	// 	"strings"
	// 	"sync"
	// 	"time"
	"strconv"

	"github.com/dcrodman/archon/util"
)

type Ship struct {
	name [23]byte
	id   uint32

	ipAddr [4]byte
	port   uint16

	// conn   net.Conn
	// recvSize   int
	// packetSize uint16
	// buffer     []byte
}

// func (s *Ship) Client() Client { return s }
// func (s *Ship) IPAddr() string { return s.ipAddr }
// func (s *Ship) Data() []byte   { return s.buffer[:s.packetSize] }
// func (s *Ship) Close()         { s.conn.Close() }

// // Encryption/decryption is handled by the TLS connection.
// func (s *Ship) Encrypt(data []byte, size uint32) {}
// func (s *Ship) Decrypt(data []byte, size uint32) {}

// func (s *Ship) Send(data []byte) error {
// 	_, err := s.conn.Write(data)
// 	return err
// }

// func (s *Ship) Process() error {
// 	s.recvSize = 0
// 	s.packetSize = 0

// 	// Wait for the packet header.
// 	for s.recvSize < ShipgateHeaderSize {
// 		bytes, err := s.conn.Read(s.buffer[s.recvSize:ShipgateHeaderSize])
// 		if bytes == 0 || err == io.EOF {
// 			// The client disconnected, we're done.
// 			return err
// 		} else if err != nil {
// 			fmt.Println("Sockt error")
// 			// Socket error, nothing we can do now
// 			return errors.New("Socket Error (" + s.ipAddr + ") " + err.Error())
// 		}
// 		s.recvSize += bytes
// 		s.packetSize, _ = util.GetPacketSize(s.buffer[:2])
// 	}
// 	pktSize := int(s.packetSize)

// 	// Grow the client's receive buffer if they send us a packet bigger
// 	// than its current capacity.
// 	if pktSize > cap(s.buffer) {
// 		newSize := pktSize + len(s.buffer)
// 		newBuf := make([]byte, newSize)
// 		copy(newBuf, s.buffer)
// 		s.buffer = newBuf
// 	}

// 	// Read in the rest of the packet.
// 	for s.recvSize < pktSize {
// 		remaining := pktSize - s.recvSize
// 		bytes, err := s.conn.Read(s.buffer[s.recvSize : s.recvSize+remaining])
// 		if err != nil {
// 			return errors.New("Socket Error (" + s.ipAddr + ") " + err.Error())
// 		}
// 		s.recvSize += bytes
// 	}
// 	return nil
// }

// // Wraps Process() in a channel that can be used for timeouts.
// func (s *Ship) Read() <-chan error {
// 	c := make(chan error)
// 	go func() {
// 		c <- s.Process()
// 	}()
// 	return c
// }

// Packet types for the shipgate. These can overlap since they aren't
// processed by the same set of handlers as the client ones.
// const (
// 	ShipgateHeaderSize  = 8
// 	ShipgateAuthType    = 0x01
// 	ShipgateAuthAckType = 0x02
// 	ShipgatePingType    = 0x03
// )

// type ShipgateHeader struct {
// 	Size uint16
// 	Type uint16
// 	// Used to distinguish between requests.
// 	Id uint32
// }

// // Initial auth request sent to the shipgate.
// type ShipgateAuthPkt struct {
// 	Header ShipgateHeader
// 	Name   [24]byte
// }

// // Send the packet serialized (or otherwise contained) in pkt to a ship.
// func SendShipPacket(ship *Ship, pkt []byte, length uint16) int {
// 	if err := ship.Send(pkt[:length]); err != nil {
// 		log.Warn("Error sending to ship %v: %s", ship.IPAddr(), err.Error())
// 		return -1
// 	}
// 	return 0
// }

// // Ship name acknowledgement.
// func SendAuthAck(ship *Ship) int {
// 	pkt := &ShipgateHeader{
// 		Size: ShipgateHeaderSize,
// 		Type: ShipgateAuthAckType,
// 		Id:   0,
// 	}
// 	data, size := util.BytesFromStruct(pkt)
// 	if config.DebugMode {
// 		fmt.Println("Sending Auth Ack")
// 		util.PrintPayload(data, size)
// 		fmt.Println()
// 	}
// 	return SendShipPacket(ship, data, uint16(size))
// }

// // Liveliness check.
// func SendPing(ship *Ship) int {
// 	pkt := &ShipgateHeader{
// 		Size: ShipgateHeaderSize,
// 		Type: ShipgatePingType,
// 		Id:   0,
// 	}
// 	data, size := util.BytesFromStruct(pkt)
// 	if config.DebugMode {
// 		fmt.Println("Sending Ping")
// 		util.PrintPayload(data, size)
// 		fmt.Println()
// 	}
// 	return SendShipPacket(ship, data, uint16(size))
// }

// Loop for the life of the server, pinging the shipgate every 30
// seconds to update the list of available ships.
// func fetchShipList() {
// 	config := GetConfig()
// 	errorInterval, pingInterval := time.Second*5, time.Second*60
// 	shipgateUrl := fmt.Sprintf("http://%s:%s/list", config.ShipgateHost, config.ShipgatePort)
// 	for {
// 		resp, err := http.Get(shipgateUrl)
// 		if err != nil {
// 			log.Error("Failed to connect to shipgate: "+err.Error(), logger.CriticalPriority)
// 			// Sleep for a shorter interval since we want to know as soon
// 			// as the shipgate is back online.
// 			time.Sleep(errorInterval)
// 		} else {
// 			ships := make([]ShipgateListEntry, 1)
// 			// Extract the Http response and convert it from JSON.
// 			shipData := make([]byte, 100)
// 			resp.Body.Read(shipData)
// 			if err = json.Unmarshal(util.StripPadding(shipData), &ships); err != nil {
// 				log.Error("Error parsing JSON response from shipgate: "+err.Error(),
// 					logger.MediumPriority)
// 				time.Sleep(errorInterval)
// 				continue
// 			}

// 			// Taking the easy way out and just reallocating the entire slice
// 			// to make the GC do the hard part. If this becomes an issue for
// 			// memory footprint then the list should be overwritten in-place.
// 			shipListMutex.Lock()
// 			if len(ships) < 1 {
// 				shipList = []ShipEntry{defaultShip}
// 			} else {
// 				shipList = make([]ShipEntry, len(shipList))
// 				for i := range ships {
// 					ship := shipList[i]
// 					ship.Unknown = 0x12
// 					// TODO: Does this have any actual significance? Will the possibility
// 					// of a ship id changing for the same ship break things?
// 					ship.Id = uint32(i)
// 					ship.Shipname = ships[i].Shipname
// 				}
// 			}
// 			shipListMutex.Unlock()
// 			log.Info("Updated ship list", logger.LowPriority)
// 			time.Sleep(pingInterval)
// 		}
// 	}
// }

// func processShipgatePacket(ship *Ship) {
// 	var hdr ShipgateHeader
// 	util.StructFromBytes(ship.Data()[:ShipgateHeaderSize], &hdr)

// 	var err error
// 	switch hdr.Type {
// 	case ShipgateAuthType:
// 		var pkt ShipgateAuthPkt
// 		util.StructFromBytes(ship.Data(), &pkt)
// 		ship.name = string(pkt.Name[:])
// 		SendAuthAck(ship)
// 		log.Info("Registered ship: %v", ship.name)
// 	default:
// 		log.Info("Received unknown packet %x from %s", hdr.Type, ship.IPAddr())
// 	}

// 	// Just log the error and let the handlers above do any cleanup. We don't
// 	// want to close the connection here like we would for a game client
// 	// in order to prevent one packet error from causing a reconnect.
// 	if err != nil {
// 		log.Warn(err.Error())
// 	}
// }

// // Per-ship connection loop. Unlike the other servers where each client
// // gets their own goroutine, each individual packet from the shipgate gets
// // its own goroutine and the ship handles mapping the responses to the
// // initiating client.
// func handleShipConnection(conn net.Conn) {
// 	addr := strings.Split(conn.RemoteAddr().String(), ":")
// 	ship := &Ship{
// 		conn:   conn,
// 		ipAddr: addr[0],
// 		port:   addr[1],
// 		buffer: make([]byte, 512),
// 	}
// 	// shipConnections.AddClient(ship)
// 	log.Info("Accepted ship connection from %v", ship.IPAddr())

// 	defer func() {
// 		if err := recover(); err != nil {
// 			log.Error("Error in ship communication: %s: %s\n%s\n",
// 				ship.IPAddr(), err, debug.Stack())
// 		}
// 		conn.Close()
// 		log.Info("Disconnected ship: %s", ship.name)
// 		// shipConnections.RemoveClient(ship)
// 	}()

// 	var err error
// 	for {
// 		select {
// 		// If we don't hear from a ship for 60 seconds, ping it to
// 		// make sure it's still alive.
// 		case <-time.After(time.Second * 60):
// 			// TODO: Ping the ship
// 			continue
// 		case err = <-ship.Read():
// 		}

// 		if err == io.EOF {
// 			break
// 		} else if err != nil {
// 			// Error communicating with the client.
// 			log.Warn(err.Error())
// 			break
// 		}
// 		go processShipgatePacket(ship)
// 	}
// }

// func startShipgate(wg *sync.WaitGroup) {
// 	// Load our certificate file ship auth.
// 	cert, err := tls.LoadX509KeyPair(CertificateFile, KeyFile)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		os.Exit(-1)
// 	}
// 	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

// 	socket, err := tls.Listen("tcp", config.Hostname+":"+config.ShipgatePort, tlsCfg)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		os.Exit(-1)
// 	}

// 	fmt.Printf("Waiting for SHIPGATE connections on %s:%s...\n",
// 		config.Hostname, config.ShipgatePort)

// 	// Wait for ship connections and spin off goroutines to handle them.
// 	for {
// 		conn, err := socket.Accept()
// 		if err != nil {
// 			log.Warn("Failed to accept connection: %s", err.Error())
// 			continue
// 		}
// 		go handleShipConnection(conn)
// 	}

// 	wg.Done()
// }
//
// Shipgate sub-server definition.
type ShipgateServer struct{}

func (server ShipgateServer) Name() string { return "SHIPGATE" }

func (server ShipgateServer) Port() string { return config.ShipgatePort }

func (server *ShipgateServer) Init() error {
	// Create our ship entry for the built-in ship server. Any other connected
	// ships will be added to this list by the shipgate, if it's enabled.
	s := &shipList[0]
	s.id = 1
	s.ipAddr = config.BroadcastIP()
	port, _ := strconv.ParseUint(config.ShipPort, 10, 16)
	s.port = uint16(port)
	copy(s.name[:], config.ShipName)
	return nil
}

func (server ShipgateServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewLoginClient(conn)
}

// Basically a no-op at this point since we only have one ship.
func (server ShipgateServer) Handle(c *Client) error {
	var err error
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	switch hdr.Type {
	default:
		log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}
