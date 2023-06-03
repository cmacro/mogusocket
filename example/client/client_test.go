package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	r := strings.NewReader("abc")
	io.Copy(os.Stdout, r)
	fmt.Print("ok")

}
