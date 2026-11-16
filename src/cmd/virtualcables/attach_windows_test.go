//go:build windows

package main

import (
	"reflect"
	"testing"
)

func TestUSBIPArgsDefaultPortUnchanged(t *testing.T) {
	got := usbipArgs(3240, "attach", "-r", "127.0.0.1", "-b", "1-1")
	want := []string{"attach", "-r", "127.0.0.1", "-b", "1-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("usbipArgs(3240) = %v, want %v", got, want)
	}
}

func TestUSBIPArgsCustomPortPrefixed(t *testing.T) {
	got := usbipArgs(3241, "list", "-r", "127.0.0.1")
	want := []string{"--tcp-port", "3241", "list", "-r", "127.0.0.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("usbipArgs(3241) = %v, want %v", got, want)
	}
}
