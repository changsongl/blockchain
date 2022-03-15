package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	// CommandVersion version
	CommandVersion = "version"
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
)

type addr struct {
	AddrList []string
}

type block struct {
	AddrFrom string
	Block    []byte
}

type versionData struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func commandToBytes(command string) []byte {
	var byteArr [commandLength]byte

	for i, c := range command {
		byteArr[i] = byte(c)
	}

	return byteArr[:]
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

func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()

	v := versionData{
		Version:    nodeVersion,
		BestHeight: bestHeight,
		AddrFrom:   nodeAddress,
	}
	payload := gobEncode(v)

	request := append(commandToBytes(CommandVersion), payload...)

	sendData(addr, request)
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
		go handleConnection(conn, bc)
	}
}

// TODO: impl
func handleConnection(conn net.Conn, bc *Blockchain) {

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
