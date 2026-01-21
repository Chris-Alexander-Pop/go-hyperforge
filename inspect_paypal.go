package main

import (
	"fmt"
	"reflect"

	"github.com/plutov/paypal/v4"
)

func main() {
	var c paypal.Client
	t := reflect.TypeOf(&c)
	m, ok := t.MethodByName("CapturedDetail")
	if ok {
		fmt.Println("Found CapturedDetail:", m.Type)
	}
}
