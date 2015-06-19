package main

import (
	// "fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	// "time"

	"ivanzoid/sshTunnel/connect"
)

func main() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, syscall.SIGTERM)

	quit := make(chan bool)

	go func() {
		<-sigCh
		quit <- true
		log.Println("Program should quit now")
	}()

	sshKey, err := ioutil.ReadFile("/Users/ivan/.ssh/id_rsa")
	if err != nil {
		log.Fatalln("Can't read ssh key file:", err)
		return
	}

	sshTunnelConfig := &connect.SSHTunnelConfig{
		RemoteAddr: "ifconfig.me",
		RemotePort: "80",
		LocalPort:  "9090",
		SSHHost:    "do.zoid.cc",
		SSHPort:    "22",
		SSHUser:    "ivan",
		SSHKey:     sshKey,
	}

	log.Println("Running ssh tunnel", sshTunnelConfig)

	err = connect.RunSSHTunnel(sshTunnelConfig, quit)
	if err != nil {
		log.Fatalln("Error running ssh tunnel:", err)
		return
	}
}
