/*
osipsdagram is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides OpenSIPS mi_datagram communication and event server.
*/

package osipsdagram

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
)

type OsipsEvent struct {
	Name              string            // Event name
	AttrValues        map[string]string // Populate AttributeValue pairs here
	Values            []string          // Populate single values here
	OriginatorAddress *net.UDPAddr      // Address of the entity originating the package
}

func NewEventServer(addrStr string, eventHandlers map[string][]func(*OsipsEvent)) (*OsipsEventServer, error) {
	var evSrv *OsipsEventServer
	if addr, err := net.ResolveUDPAddr("udp", addrStr); err != nil {
		return nil, err
	} else if sock, err := net.ListenUDP("udp", addr); err != nil {
		return nil, err
	} else {
		evSrv = &OsipsEventServer{conn: sock, eventsBuffer: bytes.NewBuffer(nil), eventHandlers: eventHandlers}
	}
	return evSrv, nil
}

// Receives events from OpenSIPS server
type OsipsEventServer struct {
	conn          *net.UDPConn
	eventsBuffer  *bytes.Buffer
	eventHandlers map[string][]func(*OsipsEvent)
}

func (evSrv *OsipsEventServer) ServeEvents(stopServing chan struct{}) error {
	var buf [65457]byte
	for {
		select {
		case <-stopServing: // Break this loop from outside
			return nil
		default:
			if readBytes, origAddr, err := evSrv.conn.ReadFromUDP(buf[0:]); err != nil {
				return err
			} else if err := evSrv.processReceivedData(buf[:readBytes], origAddr); err != nil {
				return err
			}
		}
	}
}

// Build event type out of received data
func (evSrv *OsipsEventServer) processReceivedData(rcvData []byte, origAddr *net.UDPAddr) error {
	if idxEndEvent := bytes.Index(rcvData, []byte("\n\n")); idxEndEvent == -1 { // Could not find event delimiter, something went wrong here
		return errors.New("PARSE_ERROR")
	} else { // Try generating event out event data, and start fresh a new one after resetting the buffer
		endEvent, startNewEvent := rcvData[:idxEndEvent+2], rcvData[idxEndEvent+2:]
		if _, err := evSrv.eventsBuffer.Write(endEvent); err != nil { // Possible error here is buffer full
			return err
		}
		if newEvent, err := evSrv.generateEvent(origAddr); err != nil {
			return err
		} else if err := evSrv.dispatchEvent(newEvent); err != nil {
			return err
		}
		evSrv.eventsBuffer.Reset() // Have finished consuming the previous event data, empty write buffer
		if _, err := evSrv.eventsBuffer.Write(startNewEvent); err != nil {
			return err
		}
	}
	return nil
}

// Instantiate event
func (evSrv *OsipsEventServer) generateEvent(origAddr *net.UDPAddr) (*OsipsEvent, error) {
	ev := &OsipsEvent{AttrValues: make(map[string]string), OriginatorAddress: origAddr}
	if eventName, err := evSrv.eventsBuffer.ReadBytes('\n'); err != nil {
		return nil, err
	} else {
		ev.Name = string(eventName[:len(eventName)-1])
	}
	for {
		valByte, err := evSrv.eventsBuffer.ReadBytes('\n')
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		valByte = valByte[:len(valByte)-1] // Remove \n in the end
		if len(valByte) == 0 {             // Have reached second \n, end of event processing
			break
		}
		if idxSep := bytes.Index(valByte, []byte("::")); idxSep == -1 {
			ev.Values = append(ev.Values, string(valByte))
		} else {
			ev.AttrValues[string(valByte[:idxSep])] = string(valByte[idxSep+2:])
		}
	}
	return ev, nil
}

func (evSrv *OsipsEventServer) dispatchEvent(ev *OsipsEvent) error {
	if handlers, hasHandler := evSrv.eventHandlers[ev.Name]; hasHandler {
		for _, handlerFunc := range handlers {
			go handlerFunc(ev)
		}
	}
	return nil
}

func NewOsipsMiDatagramConnector(addrStr string, reconnects int) (*OsipsMiDatagramConnector, error) {
	mi := &OsipsMiDatagramConnector{osipsAddr: addrStr, reconnects: reconnects, procLock: new(sync.Mutex)}
	if err := mi.connect(); err != nil {
		return nil, err
	}
	return mi, nil
}

// Represents connection to OpenSIPS mi_datagram
type OsipsMiDatagramConnector struct {
	osipsAddr  string
	reconnects int
	conn       *net.UDPConn
	procLock   *sync.Mutex
}

// Read from network buffer
func (mi *OsipsMiDatagramConnector) readDatagram() ([]byte, error) {
	var buf [65457]byte
	readBytes, _, err := mi.conn.ReadFromUDP(buf[0:])
	if err != nil {
		mi.disconnect()
		return nil, err
	}
	return buf[:readBytes], nil
}

func (mi *OsipsMiDatagramConnector) disconnect() {
	mi.conn.Close()
	mi.conn = nil
}

func (mi *OsipsMiDatagramConnector) connected() bool {
	return mi.conn != nil
}

// Connect with re-connect and start also listener for inbound replies
func (mi *OsipsMiDatagramConnector) connect() error {
	var err error
	if mi.conn != nil {
		mi.disconnect()
	}
	udpAddr, err := net.ResolveUDPAddr("udp4", mi.osipsAddr)
	if err != nil {
		return err
	}
	mi.conn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}
	return nil
}

// Send a command, re-connect in background if needed
func (mi *OsipsMiDatagramConnector) SendCommand(cmd []byte) ([]byte, error) {
	mi.procLock.Lock()
	defer mi.procLock.Unlock()
	if mi.conn == nil {
		for i := 0; i < mi.reconnects; i++ {
			if err := mi.connect(); err == nil {
				break
			}
		}
		if mi.conn == nil {
			return nil, errors.New("NOT_CONNECTED")
		}
	}
	if _, err := mi.conn.Write(cmd); err != nil {
		return nil, err
	}
	return mi.readDatagram()
}

func NewOsipsMiConPool(address string, reconnects int, maxConnections int) (*OsipsMiConPool, error) {
	miPool := &OsipsMiConPool{osipsAddr: address, reconnects: reconnects, mis: make(chan *OsipsMiDatagramConnector, maxConnections)}
	for i := 0; i < maxConnections; i++ {
		miPool.mis <- nil // Empty instantiate so we do not need to wait later when we pop
	}
	return miPool, nil
}

type OsipsMiConPool struct {
	osipsAddr  string
	reconnects int
	mis        chan *OsipsMiDatagramConnector // Here will be a reference towards the available connectors
}

func (mipool *OsipsMiConPool) PopMiConn() (*OsipsMiDatagramConnector, error) {
	var err error
	mi := <-mipool.mis
	if mi == nil {
		mi, err = NewOsipsMiDatagramConnector(mipool.osipsAddr, mipool.reconnects)
		if err != nil {
			return nil, err
		}
		return mi, nil
	} else {
		return mi, nil
	}
}

func (mipool *OsipsMiConPool) PushMiConn(mi *OsipsMiDatagramConnector) {
	if mi.connected() { // We only add it back if the socket is still connected
		mipool.mis <- mi
	}
}

func (mipool *OsipsMiConPool) SendCommand(cmd []byte) ([]byte, error) {
	miConn, err := mipool.PopMiConn()
	if err != nil {
		return nil, err
	} else {
		defer mipool.PushMiConn(miConn)
	}
	return miConn.SendCommand(cmd)
}
