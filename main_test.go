package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"

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

	l := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	defer l.Close()
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offerRaw := try.To1(io.ReadAll(r.Body))
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  string(offerRaw),
		}
		offerCh <- offer
		answer := <-answerCh
		try.To1(io.WriteString(w, answer.SDP))
	}))

	addr := l.Addr().String()

	var wg sync.WaitGroup
	for _, index := range []int{1, 2, 3} {
		wg.Add(1)
		func() {
			defer wg.Done()
			cmd := exec.Command("go", "run", ".", fmt.Sprintf("%d", index), addr)
			cmd.Stderr = os.Stderr
			cmd.Start()
			err := cmd.Wait()
			if err != nil {
				t.Error(err)
			}
			// time.Sleep(30 * time.Second)
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
