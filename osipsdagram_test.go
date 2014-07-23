/*
osipsdagram is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH. All Rights Reserved.
*/

package osipsdagram

import (
	"bytes"
	"reflect"
	"testing"
)

var ONLY_VALUES = ``

func TestParseAttrValue(t *testing.T) {
	rawEvent := []byte(`E_SCRIPT_EVENT
param2::value2
param1::value1

`)
	oes := OsipsEventServer{eventsBuffer: new(bytes.Buffer)}
	eOEvent := &OsipsEvent{Name: "E_SCRIPT_EVENT", AttrValues: map[string]string{"param1": "value1", "param2": "value2"}}
	oes.eventsBuffer.Write(rawEvent)
	if oEvent, err := oes.generateEvent(); err != nil {
		t.Error("Unexpected error: ", err)
	} else if !reflect.DeepEqual(eOEvent, oEvent) {
		t.Errorf("Expecting: %+v, received: %+v", eOEvent, oEvent)
	}
}

func TestParseValues(t *testing.T) {
	rawEvent := []byte(`E_SCRIPT_EVENT
value2
value1
value2
value1

`)
	oes := OsipsEventServer{eventsBuffer: new(bytes.Buffer)}
	eOEvent := &OsipsEvent{Name: "E_SCRIPT_EVENT", AttrValues: make(map[string]string), Values: []string{"value2", "value1", "value2", "value1"}}
	oes.eventsBuffer.Write(rawEvent)
	if oEvent, err := oes.generateEvent(); err != nil {
		t.Error("Unexpected error: ", err)
	} else if !reflect.DeepEqual(eOEvent, oEvent) {
		t.Errorf("Expecting: %+v, received: %+v", eOEvent, oEvent)
	}
}

func TestParseSimpleCdr(t *testing.T) {
	rawEvent := []byte(`E_ACC_CDR
method::INVITE
from_tag::2059db25
to_tag::0e481c57
callid::MTlhYmU5MTVkM2FlY2NmOTRjZWIwNzg0ZjNjM2UwYzc.
sip_code::200
sip_reason::OK
time::1405347930
cgr_reqtype::prepaid
cgr_destination::dan
cgr_account::dan
cgr_subject::dan
duration::6
setuptime::2
created::1405347928

`)
	oes := OsipsEventServer{eventsBuffer: new(bytes.Buffer)}
	eOEvent := &OsipsEvent{Name: "E_ACC_CDR",
		AttrValues: map[string]string{"method": "INVITE", "from_tag": "2059db25", "to_tag": "0e481c57", "callid": "MTlhYmU5MTVkM2FlY2NmOTRjZWIwNzg0ZjNjM2UwYzc.",
			"sip_code": "200", "sip_reason": "OK", "time": "1405347930", "cgr_reqtype": "prepaid", "cgr_destination": "dan", "cgr_account": "dan", "cgr_subject": "dan",
			"duration": "6", "setuptime": "2", "created": "1405347928"},
	}
	oes.eventsBuffer.Write(rawEvent)
	if oEvent, err := oes.generateEvent(); err != nil {
		t.Error("Unexpected error: ", err)
	} else if !reflect.DeepEqual(eOEvent, oEvent) {
		t.Errorf("Expecting: %+v, received: %+v", eOEvent, oEvent)
	}
}

func TestMiProcessReceivedData(t *testing.T) {
	rawEvent := []byte(`200 OK
dialog:active_dialogs = 0
dialog:early_dialogs = 0
dialog:processed_dialogs = 0
dialog:expired_dialogs = 0
dialog:failed_dialogs = 0
dialog:create_sent = 0
dialog:update_sent = 0
dialog:delete_sent = 0
dialog:create_recv = 0
dialog:update_recv = 0
dialog:delete_recv = 0
tm:received_replies = 0
tm:relayed_replies = 0
tm:local_replies = 0
tm:UAS_transactions = 0
tm:UAC_transactions = 0
tm:2xx_transactions = 0
tm:3xx_transactions = 0
tm:4xx_transactions = 0
tm:5xx_transactions = 0
tm:6xx_transactions = 0
tm:inuse_transactions = 0

`)
	mi := OsipsMiDatagramConnector{dagramBuffer: bytes.NewBuffer(make([]byte, 0, 65457)), datagramReply: make(chan []byte, 10)}
	if err := mi.processReceivedData(rawEvent); err != nil {
		t.Error("Unexpected error: ", err)
	}
	if rcvDatagram := <-mi.datagramReply; !reflect.DeepEqual(rcvDatagram, rawEvent) {
		t.Errorf("Expected %q, received: %q", rawEvent, rcvDatagram)
	}
}
