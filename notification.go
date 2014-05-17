package wapsnmp

import (
	"fmt"
	"log"
	"net"
	//"time"
)

type Notification struct {
	Origin *net.UDPAddr
	Oids   map[string]interface{}
}

type SnmpNotificationsTarget struct {
	Target string
	Port   uint
}

type WapSNMPListener struct {
	IpAddress string
	Port      uint
	Version   SNMPVersion
	target    string
	conn      *net.UDPConn // Cache the UDP connection in the object.
	connected bool
	channel   *chan Notification
}

type NotificationListener interface {
	Received(notification Notification)
}

func NewSnmpNotificationsTarget(bindAddress string) SnmpNotificationsTarget {
	return SnmpNotificationsTarget{bindAddress, 162}
}

func NewSnmpNotificationsTargetWithPort(bindAddress string, port uint) SnmpNotificationsTarget {
	return SnmpNotificationsTarget{bindAddress, port}
}

func NewWapSNMPListener(version SNMPVersion) (*WapSNMPListener, error) {
	return NewWapSNMPListenerBind(NewSnmpNotificationsTarget(""), version)
}

func NewWapSNMPListenerBind(bindTarget SnmpNotificationsTarget, version SNMPVersion) (*WapSNMPListener, error) {
	return NewWapSNMPListenerBindAndPort(bindTarget.Target, bindTarget.Port, version)
}

func (listener *WapSNMPListener) GetChannel() *chan Notification {
	return listener.channel
}

func NewWapSNMPListenerBindAndPort(bindIpAddress string, port uint, version SNMPVersion) (*WapSNMPListener, error) {
	target := fmt.Sprintf("%s:%v", bindIpAddress, port)
	udpAddress, err := net.ResolveUDPAddr("udp4", target)
	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		return nil, fmt.Errorf(`error listening on ("udp", "%s") : %s`, target, conn)
	}
	channel := make(chan Notification)
	listener := WapSNMPListener{bindIpAddress, port, version, target, conn, true, &channel}
	go func(myListener *WapSNMPListener) {
		for {
			log.Printf("Awaiting traps [%v]", myListener.target)
			buffer := make([]byte, bufSize, bufSize)
			readLen, address, err := myListener.conn.ReadFromUDP(buffer)
			if err != nil {
				//log.Printf(`error reading from ("udp", "%s") : %s`, myListener.target, err)
				if myListener.connected {
					continue
				} else {
					break
				}
			}
			filledBuffer := buffer[:readLen]
			log.Printf("Received bytes [%v]: %v ", address, filledBuffer)
			decodedResponse, err := DecodeSequence(filledBuffer)
			if err != nil {
				log.Printf(`error decoding notification from ("udp", "%s") : %s`, address, err)
				continue
			}
			log.Printf("Decoded response: %v", decodedResponse)
			result := extractMultipleOids(decodedResponse)
			notification := Notification{address, result}
			*myListener.channel <- notification
			log.Printf("Trap consumed")
		}
		log.Printf(`finished listening on ("udp", "%s")`, myListener.target)
	}(&listener)

	return &listener, nil
}

func (snmpListener *WapSNMPListener) Close() error {
	snmpListener.connected = false
	return snmpListener.conn.Close()
}
