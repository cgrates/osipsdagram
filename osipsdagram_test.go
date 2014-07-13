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
	eOEvent := &OsipsEvent{Name: "E_SCRIPT_EVENT", AttrValues: map[string]string{"param1": "value1", "param2": "value2"}, Values: make([]string, 0)}
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
