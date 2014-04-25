package wapsnmp

import (
	"fmt"
	"log"
	"net"
)

type WapSNMPListener struct {
	IpAddress string
	Port      uint
	Version   SNMPVersion
	target    string
	conn      *net.UDPConn // Cache the UDP connection in the object.
	connected bool
	channel   chan []byte
}

func NewWapSNMPListener(version SNMPVersion) (*WapSNMPListener, error) {
	return NewWapSNMPListenerBind("", version)
}

func NewWapSNMPListenerBind(bindIpAddress string, version SNMPVersion) (*WapSNMPListener, error) {
	return NewWapSNMPListenerBindAndPort(bindIpAddress, 162, version)
}

func (listener *WapSNMPListener) GetChannel() chan []byte {
	return listener.channel
}

func NewWapSNMPListenerBindAndPort(bindIpAddress string, port uint, version SNMPVersion) (*WapSNMPListener, error) {
	target := fmt.Sprintf("%s:%v", bindIpAddress, port)
	udpAddress, err := net.ResolveUDPAddr("udp4", target)
	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		return nil, fmt.Errorf(`error listening on ("udp", "%s") : %s`, target, conn)
	}

	listener := WapSNMPListener{bindIpAddress, port, version, target, conn, true, make(chan []byte)}
	go func() {
		for {
			buffer := make([]byte, bufSize, bufSize)
			log.Printf("Awaiting traps [%v]", target)
			readLen, address, err := listener.conn.ReadFromUDP(buffer)
			if err != nil {
				log.Printf(`error reading from ("udp", "%s") : %s`, target, err)
				if listener.connected {
					continue
				} else {
					break
				}
			}
			filledBuffer := buffer[0:readLen]
			log.Printf("Received bytes [%v]: %v ", address, filledBuffer)
			listener.channel <- filledBuffer
		}
		log.Printf(`finished listening on ("udp", "%s")`, listener.target)
	}()

	return &listener, nil
}

func (snmpListener *WapSNMPListener) Close() error {
	snmpListener.connected = false
	return snmpListener.conn.Close()
}
