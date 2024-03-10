package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/pion/ice/v3"
	"github.com/pion/webrtc/v4"
	"github.com/shynome/err0/try"
)

var offerCh = make(chan webrtc.SessionDescription)
var answerCh = make(chan webrtc.SessionDescription)

func TestMain(m *testing.M) {
	eg := webrtc.SettingEngine{}
	conn := try.To1(net.ListenUDP("udp", &net.UDPAddr{Port: 0}))
	udp := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: conn})
	eg.SetICEUDPMux(udp)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(eg))
	go func() {
		for offer := range offerCh {
			go handle(api, offer)
		}
	}()
	m.Run()
}

func TestDC(t *testing.T) {

	var wg sync.WaitGroup
	for _, index := range []int{1, 2, 3} {
		wg.Add(1)
		func() {
			defer wg.Done()
			cmd := exec.Command("go", "run", ".", fmt.Sprintf("%d", index))
			stdin, stdinWriter := io.Pipe()
			cmd.Stdin = stdin
			stdout := new(bytes.Buffer)
			cmd.Stdout = stdout
			cmd.Stderr = os.Stderr
			try.To(cmd.Start())
			log.Println("offer read")
			time.Sleep(3 * time.Second)
			offerRaw := stdout.String()
			log.Println("offer readed")
			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  string(offerRaw),
			}
			offerCh <- offer
			answer := <-answerCh
			log.Println("answer write")
			try.To1(io.WriteString(stdinWriter, answer.SDP))
			try.To(stdin.Close())
			log.Println("answer writed")
			err := cmd.Wait()
			if err != nil {
				t.Error(err)
			}
			time.Sleep(30 * time.Second)
		}()
	}
	wg.Wait()
}

func handle(api *webrtc.API, sdp webrtc.SessionDescription) {
	wcfg := webrtc.Configuration{}
	pc := try.To1(api.NewPeerConnection(wcfg))
	try.To(pc.SetRemoteDescription(sdp))
	answer := try.To1(pc.CreateAnswer(nil))
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	try.To(pc.SetLocalDescription(answer))
	<-gatherComplete
	answer = *pc.LocalDescription()
	answerCh <- answer
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			dc.SendText("world")
		})
	})
}
