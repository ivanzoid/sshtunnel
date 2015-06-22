package connect

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/juju/errgo/errors"
	"golang.org/x/crypto/ssh"
)

const (
	localAddress = "localhost"
)

// Equivalent ssh command given this config will be:
// ssh -L LocalPort:RemoteAddr:RemotePort SSHUser@SSHHost
type SSHTunnelConfig struct {
	RemoteAddr string
	RemotePort string
	LocalPort  string
	SSHHost    string
	SSHPort    string
	SSHUser    string
	SSHKey     []byte
}

func (c *SSHTunnelConfig) String() string {
	return fmt.Sprintf("{%v@%v %v:%v:%v}", c.SSHUser, c.SSHHost, c.LocalPort, c.RemoteAddr, c.RemotePort)
}

func sshConfig(tunnelConfig *SSHTunnelConfig) (*ssh.ClientConfig, error) {
	signer, err := ssh.ParsePrivateKey(tunnelConfig.SSHKey)
	if err != nil {
		return nil, errors.Mask(err)
	}
	authMethod := ssh.PublicKeys(signer)

	sshConfig := &ssh.ClientConfig{
		User: tunnelConfig.SSHUser,
		Auth: []ssh.AuthMethod{authMethod},
	}

	return sshConfig, nil
}

func handleConnection(localListener net.Listener, sshServerAddrString, remoteAddrString string, sshConfig *ssh.ClientConfig, quit chan bool) error {

	localConn, err := localListener.Accept()
	if err != nil {
		return errors.NoteMask(err, "listen.Accept failed")
	}

	log.Println("Accepted local connection")

	// Establish connection to ssh server

	sshConn, err := ssh.Dial("tcp", sshServerAddrString, sshConfig)
	if err != nil {
		return errors.NoteMask(err, "ssh.Dial failed")
	}

	log.Println("Established ssh connection")

	go func() {
		<-quit
		log.Println("Got quit message for connection!")
		log.Println("Closing local connection")
		localConn.Close()
		log.Println("Closing ssh connection")
		sshConn.Close()
	}()

	// Initiate remote connection from ssh host to target host

	remoteConn, err := sshConn.Dial("tcp", remoteAddrString)
	if err != nil {
		return errors.NoteMask(err, "ssh connection .Dial failed")
	}

	log.Println("Established remote connection from ssh host to target host")

	go func() {
		_, err = io.Copy(remoteConn, localConn)
		if err != nil {
			log.Println("io.Copy from local to remote connection returned error:", err)
		} else {
			log.Println("io.Copy from local to remote connection finished")
		}
		log.Println("Closing local connection")
		localConn.Close()
		log.Println("Closing ssh connection")
		sshConn.Close()
	}()

	go func() {
		_, err = io.Copy(localConn, remoteConn)
		if err != nil {
			log.Println("io.Copy from remote to local connection returned error:", err)
		} else {
			log.Println("io.Copy from remote to local connection finished")
		}
		log.Println("Closing local connection")
		localConn.Close()
		log.Println("Closing ssh connection")
		sshConn.Close()
	}()

	return nil
}

// Is blocking call.
func RunSSHTunnel(tunnelConfig *SSHTunnelConfig, quit chan bool) error {

	sshConfig, err := sshConfig(tunnelConfig)

	localAddrString := fmt.Sprintf("%v:%v", localAddress, tunnelConfig.LocalPort)
	remoteAddrString := fmt.Sprintf("%v:%v", tunnelConfig.RemoteAddr, tunnelConfig.RemotePort)
	sshServerAddrString := fmt.Sprintf("%v:%v", tunnelConfig.SSHHost, tunnelConfig.SSHPort)

	log.Println("localAddrString: ", localAddrString)
	log.Println("remoteAddrString: ", remoteAddrString)
	log.Println("sshServerAddrString: ", sshServerAddrString)

	// Create listening socket

	localListener, err := net.Listen("tcp", localAddrString)
	if err != nil {
		return errors.NoteMask(err, "net.Listen failed")
	}

	log.Println("Created listening socket")

	connQuit := make(chan bool)

	go func() {
		<-quit
		log.Println("Got quit:")
		log.Println("Closing listening socket")
		localListener.Close()
		log.Println("Broadcasting quit to connections")
		close(connQuit)
	}()

	done := false
	for !done {
		select {
		case <-connQuit:
			log.Println("Got listening loop break message!")
			done = true
		default:
			err = handleConnection(localListener, sshServerAddrString, remoteAddrString, sshConfig, connQuit)
			if err != nil {
				log.Println("handleConnection error:", err)
			}
		}
	}

	log.Println("RunSSHTunnel done")

	return nil
}
