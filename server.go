package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const (
	// CommandVersion version
	CommandVersion = "version"

	// CommandBlock block
	CommandBlock = "block"

	// CommandAddr address
	CommandAddr = "addr"

	// CommandInv invite
	CommandInv = "inv"

	// CommandGetBlocks get blocks
	CommandGetBlocks = "getblcoks"

	// CommandGetData get data
	CommandGetData = "getdata"

	// CommandTx transaction
	CommandTx = "tx"
)

const (
	CommandGetDataTypeBlock = "block"

	CommandGetDataTypeData = "data"
)

const (
	// protocol is server protocol
	protocol = "tcp"

	nodeVersion = 1

	// commandLength is the length for command
	commandLength = 12
)

var (
	// nodeAddress is the address of node
	nodeAddress string

	// miningAddress is the address for mining
	miningAddress string

	// knownNodes is a list of known nodes
	knownNodes = []string{"localhost:3000"}

	// blocksInTransit stores block data in transit
	blocksInTransit = [][]byte{}
)

type addrData struct {
	AddrList []string
}

type blockData struct {
	AddrFrom string
	Block    []byte
}

type versionData struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

type getBlocksData struct {
	AddrFrom string
}

type getDataData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

// commandToBytes converts command string to bytes
func commandToBytes(command string) []byte {
	var byteArr [commandLength]byte

	for i, c := range command {
		byteArr[i] = byte(c)
	}

	return byteArr[:]
}

// bytesToCommand converts bytes to command string
func bytesToCommand(bytes []byte) string {
	command := make([]byte, 0, commandLength)

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return string(command)
}

func sendCommandAndPayload(addr, command string, data interface{}) {
	payload := gobEncode(data)
	request := append(commandToBytes(command), payload...)

	sendData(addr, request)
}

func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		log.Printf("%s is not avaliable\n", addr)

		var newKnownNodes []string
		for _, node := range knownNodes {
			if node != addr {
				newKnownNodes = append(newKnownNodes, node)
			}
		}

		knownNodes = newKnownNodes
		return
	}

	defer func() {
		if err = conn.Close(); err != nil {
			log.Println(err)
		}
	}()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

// sendVersion sends the current height of blockchain to other node
func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()

	v := versionData{
		Version:    nodeVersion,
		BestHeight: bestHeight,
		AddrFrom:   nodeAddress,
	}

	sendCommandAndPayload(addr, CommandVersion, v)
}

func requestBlocks() {
	for _, node := range knownNodes {
		sendCommandAndPayload(node, CommandGetBlocks, getBlocksData{AddrFrom: nodeAddress})
	}
}

func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if closeErr := ln.Close(); closeErr != nil {
			log.Panic(closeErr)
		}
	}()

	bc := NewBlockchain(nodeID)

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, cErr := ln.Accept()
		if cErr != nil {
			log.Panic(cErr)
		}
		go func() {
			defer func() {
				if closeErr := conn.Close(); closeErr != nil {
					log.Panic(err)
				}
			}()
			handleConnection(conn, bc)
		}()
	}
}

// TODO: impl
func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}

	command := bytesToCommand(request[:commandLength])

	log.Printf("Receiver %s command\n", command)

	switch command {
	case CommandVersion:
		handleVersion(request, bc)
	case CommandAddr:
		handleAddr(request)
	case CommandBlock:
		handleBlock(request, bc)
	case CommandInv:
		handleInv(request, bc)
	case CommandGetBlocks:
		handleGetBlocks(request, bc)
	case CommandGetData:
		handleGetData(request, bc)
	case CommandTx:
		handleTx(request, bc)
	default:
		log.Println("Unknown command")
	}
}

// TODO: impl
func handleAddr(request []byte) {
	var payload addrData

	decodeRequestData(&payload, request)
	for _, addr := range payload.AddrList {
		addToKnownNodes(addr)
	}

	requestBlocks()
}

// TODO: impl
func handleBlock(request []byte, bc *Blockchain) {
	var payload blockData
	decodeRequestData(&payload, request)

	block := DeserializeBlock(payload.Block)

	bc.AddBlock(block)

	if hasBlockInTransit() {
		sendCommandAndPayload(payload.AddrFrom, CommandGetData,
			getDataData{AddrFrom: nodeAddress, Type: CommandGetDataTypeBlock, ID: blocksInTransit[0]})

		blocksInTransit = blocksInTransit[1:]
	} else {
		NewUTXOSet(bc).Reindex()
	}
}

// hasBlockInTransit returns if having blocks in transit
func hasBlockInTransit() bool {
	return len(blocksInTransit) != 0
}

// TODO: impl
func handleInv(request []byte, bc *Blockchain) {

}

// TODO: impl
func handleGetBlocks(request []byte, bc *Blockchain) {

}

// TODO: impl
func handleGetData(request []byte, bc *Blockchain) {

}

// TODO: impl
func handleTx(request []byte, bc *Blockchain) {

}

// handleVersion handles CommandVersion request
func handleVersion(request []byte, bc *Blockchain) {
	var payload versionData
	decodeRequestData(payload, request)

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendCommandAndPayload(payload.AddrFrom, CommandGetBlocks, getBlocksData{AddrFrom: nodeAddress})
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	addToKnownNodes(payload.AddrFrom)
}

// addToKnownNodes checks whether address is in the known nodes list and adds to list if not.
func addToKnownNodes(addr string) {
	if !nodeIsKnow(addr) {
		knownNodes = append(knownNodes, addr)
	}
}

func decodeRequestData(data interface{}, request []byte) {
	var buff bytes.Buffer
	buff.Write(request[commandLength:])

	dec := gob.NewDecoder(&buff)
	if err := dec.Decode(data); err != nil {
		log.Panic(err)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnow(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}
