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
* The BLOCK and SHIP server logic.
 */
package main

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon/util"
	"net"
	"strconv"
)

const BlockName = "Block"

type Block struct {
	Unknown   uint16
	BlockId   uint32
	Padding   uint16
	BlockName [36]byte
}

func handleShipLogin(sc *Client) error {
	if _, err := VerifyAccount(sc); err != nil {
		return err
	}
	sc.SendSecurity(BBLoginErrorNone, sc.guildcard, sc.teamId)
	return nil
}

func handleBlockSelection(sc *Client) error {
	var pkt MenuSelectionPacket
	util.StructFromBytes(sc.Data(), &pkt)

	// Grab the chosen block and redirect them to the selected block server.
	port, _ := strconv.ParseInt(config.ShipPort, 10, 16)
	selectedBlock := pkt.ItemId
	if int(selectedBlock) > config.NumBlocks {
		return errors.New(fmt.Sprintf("Block selection %v out of range %v", selectedBlock, config.NumBlocks))
	}
	sc.SendRedirect(uint16(uint32(port)+selectedBlock), config.HostnameBytes())
	return nil
}

func NewShipClient(conn *net.TCPConn) (*Client, error) {
	sc := NewClient(conn, BBHeaderSize)
	err := error(nil)
	if sc.SendWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + sc.IPAddr())
		sc = nil
	}
	return sc, err
}

// Ship sub-server definition.
type ShipServer struct {
	// Precomputed block packet.
	blockPkt *BlockListPacket
}

func (server ShipServer) Name() string { return "SHIP" }

func (server ShipServer) Port() string { return config.ShipPort }

func (server ShipServer) Init() {
	// Precompute the block list packet since it's not going to change.
	numBlocks := config.NumBlocks
	ship := shipList[0]

	server.blockPkt = &BlockListPacket{
		Header:  BBHeader{Type: BlockListType, Flags: uint32(numBlocks)},
		Unknown: 0x08,
		Blocks:  make([]Block, numBlocks),
	}
	shipName := fmt.Sprintf("%d:%s", ship.id, ship.name)
	copy(server.blockPkt.ShipName[:], util.ConvertToUtf16(shipName))

	for i := 0; i < numBlocks; i++ {
		b := &server.blockPkt.Blocks[i]
		b.Unknown = 0x12
		// TODO: Teth sets this to (0xEFFFFFFF - block num)?
		b.BlockId = uint32(i + 1)
		blockName := fmt.Sprintf("BLOCK %02d", i+1)
		copy(b.BlockName[:], util.ConvertToUtf16(blockName))
	}
}

func (server ShipServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
}

func (server ShipServer) Handle(c *Client) error {
	var err error = nil
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	switch hdr.Type {
	case LoginType:
		err = handleShipLogin(c)
		c.SendBlockList(server.blockPkt)
	case MenuSelectType:
		err = handleBlockSelection(c)
	default:
		log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

// Block sub-server definition.
type BlockServer struct {
	name string
	port string
}

func (server BlockServer) Name() string { return server.name }

func (server BlockServer) Port() string { return server.port }

func (server BlockServer) Init() {}

func (server BlockServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
}

func (server BlockServer) Handle(c *Client) error {
	var err error = nil
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	switch hdr.Type {
	case LoginType:
		err = handleShipLogin(c)
		// TODO: Send lobby data
	default:
		log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}