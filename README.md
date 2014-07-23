OpenSIPS  Datagram Communication using Go
========================================
[![Build Status](https://secure.travis-ci.org/cgrates/cgrates.png)](http://travis-ci.org/cgrates/osipsdagram)

It offers both remote MI commands as well as UDP Event handler Server with auto subscribe. One can instantiate a signle connection or a pool of connections if concurrency is required.

## Installation ##

`go get github.com/cgrates/osipsdagram`

## Support ##
Join [CGRateS](http://www.cgrates.org/ "CGRateS Website") on Google Groups [here](https://groups.google.com/forum/#!forum/cgrates "CGRateS on GoogleGroups").

## License ##
OsipsDagram is released under the [MIT License](http://www.opensource.org/licenses/mit-license.php "MIT License").
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

## Sample usage code ##
```
package main

import (
	"fmt"
	"github.com/cgrates/osipsdagram"
	"time"
)

func printEvent(ev *osipsdagram.OsipsEvent) {
	fmt.Printf("Got event: %+v\n", ev)
}

func main() {
	cmd := []byte(`:get_statistics:
dialog:
tm:

`)

	// Test sending command over single connection
	miConn, err := osipsdagram.NewOsipsMiDatagramConnector("localhost:8020", 2)
	if err != nil {
		fmt.Printf("Cannot create new mi pool: %s", err.Error())
		return
	}
	startTime := time.Now()
	for i := 0; i < 10500; i++ {
		go func(i int) {
			if reply, err := miConn.SendCommand(cmd); err != nil {
				fmt.Printf("Got error when executing the command: %s\n", err.Error())
			} else {
				fmt.Printf("Request nr: %d, got answer to command: %s\n", i, string(reply))
			}
		}(i)
	}

	if reply, err := miConn.SendCommand(cmd); err != nil {
		fmt.Printf("Got error when executing the command: %s\n", err.Error())
	} else {
		fmt.Printf("Got answer to command: %s\n", string(reply))
	}
	fmt.Printf("Finished executing commands, total time: %v\n", time.Now().Sub(startTime))

	// Test sending command over pool of connections
	miPool, err := osipsdagram.NewOsipsMiConPool("localhost:8020", 2, 3)
	if err != nil {
		fmt.Printf("Cannot create new mi pool: %s", err.Error())
		return
	}
	startTime = time.Now()
	for i := 0; i < 10500; i++ {
		go func(i int) {
			if reply, err := miPool.SendCommand(cmd); err != nil {
				fmt.Printf("Got error when executing the command: %s\n", err.Error())
			} else {
				fmt.Printf("Request nr: %d, got answer to command: %s\n", i, string(reply))
			}
		}(i)
	}
	if reply, err := miPool.SendCommand(cmd); err != nil {
		fmt.Printf("Got error when executing the command: %s\n", err.Error())
	} else {
		fmt.Printf("Got answer to command: %s\n", string(reply))
	}
	fmt.Printf("Finished executing commands, total time: %v\n", time.Now().Sub(startTime))

	// Event server
	evsrv, err := osipsdagram.NewEventServer("localhost:2020",
		map[string][]func(*osipsdagram.OsipsEvent){
			"E_ACC_CDR": []func(*osipsdagram.OsipsEvent){printEvent}})
	if err != nil {
		fmt.Printf("Cannot create new server: %s", err.Error())
		return
	}
	evsrv.ServeEvents()

}


```
