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
	"time"
)

func fib() func() int {
	a, b := 0, 1
	return func() int {
		a, b = b, a+b
		return a
	}
}

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
			evSrv.conn.SetReadDeadline(time.Now().Add(time.Duration(1) * time.Second))
			if readBytes, origAddr, err := evSrv.conn.ReadFromUDP(buf[0:]); err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() && readBytes == 0 { // Not real error but our enforcement, continue reading events
					continue
				}
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
	mi := &OsipsMiDatagramConnector{osipsAddr: addrStr, reconnects: reconnects, delayFunc: fib(), connMutex: new(sync.RWMutex)}
	if err := mi.connect(); err != nil {
		return nil, err
	}
	return mi, nil
}

// Represents connection to OpenSIPS mi_datagram
type OsipsMiDatagramConnector struct {
	osipsAddr  string
	reconnects int
	delayFunc  func() int
	conn       *net.UDPConn
	connMutex  *sync.RWMutex
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
	mi.connMutex.Lock()
	defer mi.connMutex.Unlock()
	mi.conn.Close()
	mi.conn = nil
}

func (mi *OsipsMiDatagramConnector) connected() bool {
	mi.connMutex.RLock()
	defer mi.connMutex.RUnlock()
	return mi.conn != nil
}

// Connect with re-connect and start also listener for inbound replies
func (mi *OsipsMiDatagramConnector) connect() error {
	var err error
	if mi.connected() {
		mi.disconnect()
	}
	udpAddr, err := net.ResolveUDPAddr("udp4", mi.osipsAddr)
	if err != nil {
		return err
	}
	mi.connMutex.Lock()
	defer mi.connMutex.Unlock()
	mi.conn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}
	return nil
}

func (mi *OsipsMiDatagramConnector) reconnectIfNeeded() error {
	if mi.connected() { // No need to reconnect
		return nil
	}
	var err error
	i := 0
	for {
		if mi.reconnects != -1 && i >= mi.reconnects { // Maximum reconnects reached, -1 for infinite reconnects
			break
		}
		if err = mi.connect(); err == nil || mi.connected() {
			mi.delayFunc = fib() // Reset the reconnect delay
			break                // No error or unrelated to connection
		}
		time.Sleep(time.Duration(mi.delayFunc()) * time.Second)
		i++
	}
	if err == nil && !mi.connected() {
		return errors.New("NOT_CONNECTED")
	}
	return err // nil or last error in the loop
}

// Send a command, re-connect in background if needed
func (mi *OsipsMiDatagramConnector) SendCommand(cmd []byte) ([]byte, error) {
	if err := mi.reconnectIfNeeded(); err != nil {
		return nil, err
	}
	mi.connMutex.RLock()
	if _, err := mi.conn.Write(cmd); err != nil {
		mi.connMutex.RUnlock()
		return nil, err
	}
	mi.connMutex.RUnlock()
	return mi.readDatagram()
}

// Useful to find out from outside the local IP/Port connected
func (mi *OsipsMiDatagramConnector) LocallAddr() net.Addr {
	if !mi.connected() {
		return nil
	}
	mi.connMutex.RLock()
	defer mi.connMutex.RUnlock()
	return mi.conn.LocalAddr()
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
	if mipool == nil {
		return nil, errors.New("UNCONFIGURED_OPENSIPS_POOL")
	}
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
