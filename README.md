OpenSIPS  Datagram Communication using Go
========================================
[![Build Status](https://secure.travis-ci.org/cgrates/cgrates.png)](http://travis-ci.org/cgrates/osipsdagram)

It offers both remote MI commands as well as UDP Event handler Server with auto subscribe.

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
)

func printEvent(ev *osipsdagram.OsipsEvent) {
	fmt.Printf("Got event: %+v\n", ev)
}

func main() {
	evsrv, err := osipsdagram.NewEventServer("localhost:2020",
		map[string][]func(*osipsdagram.OsipsEvent){
			"E_ACC_CDR": []func(*osipsdagram.OsipsEvent){printEvent}})
	if err != nil {
		fmt.Printf("Cannot create new server: %s", err.Error())
		return
	}
	/*
	// Test sending commands and receive raw reply
		miConn, err := osipsdagram.NewOsipsMiDatagramConnector("localhost:8020", 3)
		if err != nil {
			fmt.Printf("Cannot create new mi connector: %s", err.Error())
			return
		}
		cmd := []byte(`:get_statistics:
			dialog:
			tm:

			`)

		if reply, err := miConn.SendCommand(cmd); err != nil {
			fmt.Printf("Got error when executing the command: %s\n", err.Error())
		} else {
			fmt.Printf("Got answer to command: %s\n", string(reply))
		}
	*/
	evsrv.ServeEvents()

}

```
