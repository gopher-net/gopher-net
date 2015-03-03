package api

// TODO: refactor tests to work again
import (
//	"crypto/rand"
//	"encoding/hex"
//	"net"
//	"testing"
//	"time"
//
//	"github.com/docker/libchan"
//	"github.com/docker/libchan/spdy"
//    "github.com/gopher-net/gopher-net/api"
)

//// UUID generation for API
//func CreateUUID() (string, error) {
//	uuid := make([]byte, 16)
//	n, err := rand.Read(uuid)
//	if n != len(uuid) || err != nil {
//		return "", err
//	}
//	uuid[8] = 0x80
//	uuid[4] = 0x40
//	return hex.EncodeToString(uuid), nil
//}

//func TestApi(t *testing.T) {
////	d := NewDaemon()
////	go d.ApiListen()
//
//	entry := &BgpDbEntry{
//		Prefix:         net.ParseIP("1.1.1.1"),
//		PrefixLen:      24,
//		PathAttributes: nil,
//	}
//
////	d.BgpDb.AddEntry(net.ParseIP("10.10.10.10"), entry)
////	d.BgpDb.AddEntry(net.ParseIP("20.20.20.20"), entry)
//
//	// wait for main loop
//	time.Sleep(1 * time.Second)
//
//	var client net.Conn
//	var err error
//
//	client, err = net.Dial("tcp", "127.0.0.1:12345")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	transport, err := spdy.NewClientTransport(client)
//	if err != nil {
//		log.Fatal(err)
//	}
//	sender, err := transport.NewSendChannel()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	receiver, remoteSender := libchan.Pipe()
//
//	command := &api.RemoteCommand{
//		api.Cmd:        "show",
//		Args:       nil,
//		StatusChan: remoteSender,
//	}
//
//	err = sender.Send(command)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	response := &CommandResponse{}
//	err = receiver.Receive(response)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	log.Print(response.Data)
//}
//
//func TestUUID(t *testing.T) {
//    uuid, err := CreateUUID()
//    if  err != nil {
//        t.Fatalf("create UUID error %s",err)
//    }
//    t.Logf("uuid[%s]\n",uuid)
//}
//
