package main

import (
	"fmt"
	"errors"
	"net/rpc"
	"net"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/exec"
)

//Args ...
type Args struct {
	A, B int
}

//Quotient ...
type Quotient struct {
	Quo, Rem int
}

//Arith ...
type Arith int
//Multiply ...
func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}
//Divide ...
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

/*
There is a test application
 */
func main() {
	arith := new(Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":33771")
	if e != nil {
		log.Fatalf("listen error: %v", e)
	}

	go http.Serve(l, nil)
	Sayhello()

	sch := make(chan os.Signal, 1)
	signal.Notify(sch, os.Interrupt, os.Kill)

	go func() {
		client, err := rpc.DialHTTP("tcp", "127.0.0.1:33771")
		if err != nil {
			log.Fatal("dial failed.", err)
		}

		args := &Args{7,8}
		var reply int

		err = client.Call("Arith.Multiply", args, &reply)
		if err != nil {
			log.Fatal("arith error:", err)
		}

		fmt.Printf("Arith: %d*%d=%d\n",args.A, args.B, reply)
		close(sch)
	}()

	<-sch
	fmt.Println("Server down.")
}

//Sayhello function Sample of the application`s function
func Sayhello() {
	fmt.Println("hello world")
	s, e := exec.LookPath("golint")
	if e != nil {
		fmt.Printf("%v\n", e)
		return
	}

	fmt.Printf("# %s\n", s)
}