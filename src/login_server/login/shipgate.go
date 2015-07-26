/*
* Archon Login Server
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
package login

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"libarchon/server"
	"libarchon/util"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

type Ship struct {
	// We aren't communicating with a PSO client, but the
	// basic connection logic is the same for ship communication.
	conn net.Conn
	name string

	ipAddr string
	port   string

	recvSize   int
	packetSize uint16
	buffer     []byte
}

func (s *Ship) Client() server.Client { return s }
func (s *Ship) IPAddr() string        { return s.ipAddr }
func (s *Ship) Data() []byte          { return s.buffer[:s.packetSize] }
func (s *Ship) Close()                { s.conn.Close() }

// Encryption/decryption is handled by the TLS connection.
func (s *Ship) Encrypt(data []byte, size uint32) {}
func (s *Ship) Decrypt(data []byte, size uint32) {}

func (s *Ship) Send(data []byte) error {
	return nil
}

func (s *Ship) Process() error {
	s.recvSize = 0
	s.packetSize = 0

	// Wait for the packet header.
	for s.recvSize < ShipgateHeaderSize {
		bytes, err := s.conn.Read(s.buffer[s.recvSize:ShipgateHeaderSize])
		if bytes == 0 || err == io.EOF {
			// The client disconnected, we're done.
			return err
		} else if err != nil {
			fmt.Println("Sockt error")
			// Socket error, nothing we can do now
			return errors.New("Socket Error (" + s.ipAddr + ") " + err.Error())
		}
		s.recvSize += bytes
		s.packetSize, _ = util.GetPacketSize(s.buffer[:2])
	}
	pktSize := int(s.packetSize)

	// Grow the client's receive buffer if they send us a packet bigger
	// than its current capacity.
	if pktSize > cap(s.buffer) {
		newSize := pktSize + len(s.buffer)
		newBuf := make([]byte, newSize)
		copy(newBuf, s.buffer)
		s.buffer = newBuf
	}

	// Read in the rest of the packet.
	for s.recvSize < pktSize {
		remaining := pktSize - s.recvSize
		bytes, err := s.conn.Read(s.buffer[s.recvSize : s.recvSize+remaining])
		if err != nil {
			return errors.New("Socket Error (" + s.ipAddr + ") " + err.Error())
		}
		s.recvSize += bytes
	}
	return nil
}

// Loop for the life of the server, pinging the shipgate every 30
// seconds to update the list of available ships.
func fetchShipList() {
	// config := GetConfig()
	// errorInterval, pingInterval := time.Second*5, time.Second*60
	// shipgateUrl := fmt.Sprintf("http://%s:%s/list", config.ShipgateHost, config.ShipgatePort)
	// for {
	// 	resp, err := http.Get(shipgateUrl)
	// 	if err != nil {
	// 		log.Error("Failed to connect to shipgate: "+err.Error(), logger.CriticalPriority)
	// 		// Sleep for a shorter interval since we want to know as soon
	// 		// as the shipgate is back online.
	// 		time.Sleep(errorInterval)
	// 	} else {
	// 		ships := make([]ShipgateListEntry, 1)
	// 		// Extract the Http response and convert it from JSON.
	// 		shipData := make([]byte, 100)
	// 		resp.Body.Read(shipData)
	// 		if err = json.Unmarshal(util.StripPadding(shipData), &ships); err != nil {
	// 			log.Error("Error parsing JSON response from shipgate: "+err.Error(),
	// 				logger.MediumPriority)
	// 			time.Sleep(errorInterval)
	// 			continue
	// 		}

	// 		// Taking the easy way out and just reallocating the entire slice
	// 		// to make the GC do the hard part. If this becomes an issue for
	// 		// memory footprint then the list should be overwritten in-place.
	// 		shipListMutex.Lock()
	// 		if len(ships) < 1 {
	// 			shipList = []ShipEntry{defaultShip}
	// 		} else {
	// 			shipList = make([]ShipEntry, len(shipList))
	// 			for i := range ships {
	// 				ship := shipList[i]
	// 				ship.Unknown = 0x12
	// 				// TODO: Does this have any actual significance? Will the possibility
	// 				// of a ship id changing for the same ship break things?
	// 				ship.Id = uint32(i)
	// 				ship.Shipname = ships[i].Shipname
	// 			}
	// 		}
	// 		shipListMutex.Unlock()
	// 		log.Info("Updated ship list", logger.LowPriority)
	// 		time.Sleep(pingInterval)
	// 	}
	// }
}

func processShipgatePacket(ship *Ship) error {
	var hdr ShipgateHeader
	util.StructFromBytes(ship.Data()[:ShipgateHeaderSize], &hdr)

	var err error = nil
	switch hdr.Type {
	case ShipgateAuthType:
		var pkt ShipgateAuthPkt
		util.StructFromBytes(ship.Data(), &pkt)
		ship.name = string(pkt.Name[:])
		log.Info("Registered ship: %v", ship.name)

	default:
		log.Info("Received unknown packet %x from %s", hdr.Type, ship.IPAddr())
	}

	return err
}

// Per-ship connection loop.
func handleShipConnection(conn net.Conn) {
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	ship := &Ship{
		conn:   conn,
		ipAddr: addr[0],
		port:   addr[1],
		buffer: make([]byte, 512),
	}
	shipConnections.AddClient(ship)
	log.Info("Accepted ship connection from %v", ship.IPAddr())

	defer func() {
		if err := recover(); err != nil {
			log.Error("Error in ship communication: %s: %s\n%s\n",
				ship.IPAddr(), err, debug.Stack())
		}
		conn.Close()
		log.Info("Disconnected ship: %s", ship.name)
		shipConnections.RemoveClient(ship)
	}()

	for {
		err := ship.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		if err = processShipgatePacket(ship); err != nil {
			log.Warn(err.Error())
			break
		}
	}
}

func startShipgate(wg *sync.WaitGroup) {
	// Load our certificate file ship auth.
	cert, err := tls.LoadX509KeyPair(CertificateFile, KeyFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	socket, err := tls.Listen("tcp", config.Hostname+":"+config.ShipgatePort, tlsCfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	fmt.Printf("Waiting for SHIPGATE connections on %s:%s...\n",
		config.Hostname, config.ShipgatePort)

	// Wait for ship connections and spin off goroutines to handle them.
	for {
		conn, err := socket.Accept()
		if err != nil {
			log.Warn("Failed to accept connection: %s", err.Error())
			continue
		}
		go handleShipConnection(conn)
	}

	wg.Done()
}