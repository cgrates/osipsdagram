/*
osipsdagram is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

Provides OpenSIPS mi_datagram communication and event server.
*/

package osipsdagram

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

type OsipsEvent struct {
	Name       string            // Event name
	AttrValues map[string]string // Populate AttributeValue pairs here
	Values     []string          // Populate single values here
}

func NewEventServer(addrStr string) (*OsipsEventServer, error) {
	var evSrv *OsipsEventServer
	if addr, err := net.ResolveUDPAddr("udp", addrStr); err != nil {
		return nil, err
	} else if sock, err := net.ListenUDP("udp", addr); err != nil {
		return nil, err
	} else {
		maxBuf := make([]byte, 0, 65457) // Given by opensips maximum datagram size
		evSrv = &OsipsEventServer{conn: sock, eventsBuffer: bytes.NewBuffer(maxBuf)}
	}
	return evSrv, nil
}

// Receives events from OpenSIPS server
type OsipsEventServer struct {
	conn         *net.UDPConn
	eventsBuffer *bytes.Buffer
}

func (evSrv *OsipsEventServer) ServeEvents() error {
	for {
		buf := make([]byte, 512)
		if _, _, err := evSrv.conn.ReadFromUDP(buf); err != nil {
			return err
		}
		if err := evSrv.processReceivedData(buf); err != nil {
			return err
		}

	}
}

// Build event type out of received data
func (evSrv *OsipsEventServer) processReceivedData(rcvData []byte) error {
	if idxEndEvent := bytes.Index(rcvData, []byte("\n\n")); idxEndEvent == -1 { // Did not find end of event, write in the content buffer without triggering dispatching
		if _, err := evSrv.eventsBuffer.Write(rcvData); err != nil { // Possible error here is buffer full
			return err
		}
	} else { // Try generating event out event data, and start fresh a new one after resetting the buffer
		endEvent, startNewEvent := rcvData[:idxEndEvent+2], rcvData[idxEndEvent+2:]
		if _, err := evSrv.eventsBuffer.Write(endEvent); err != nil { // Possible error here is buffer full
			return err
		}
		if newEvent, err := evSrv.generateEvent(); err != nil {
			return err
		} else if err := evSrv.processEvent(newEvent); err != nil {
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
func (evSrv *OsipsEventServer) generateEvent() (*OsipsEvent, error) {
	ev := &OsipsEvent{AttrValues: make(map[string]string), Values: make([]string, 0)}
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

func (evSrv *OsipsEventServer) processEvent(ev *OsipsEvent) error {
	fmt.Printf("Got event: %+v\n", ev)
	return nil
}
