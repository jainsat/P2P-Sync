package main

import (
	"fmt"
	"lib"
)

func main() {
	dcl := lib.NewDCL()
	dcl.Append("1")
	dcl.Append("2")
	dcl.Append("3")
	dcl.Append("4")
	dcl.Append("5")

	dcl.Print()
	dcl.RPrint()
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())
	fmt.Println(dcl.Next())

	fmt.Println(dcl.Next())

}
