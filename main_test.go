package main

import (
	"context"
	"log"
	"net"
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
	c := try.To1(net.ListenUDP("udp", &net.UDPAddr{Port: 0}))
	udp := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: c})
	// udp := try.To1(ice.NewMultiUDPMuxFromPort(0))
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
	eg := webrtc.SettingEngine{}
	if true {
		c := try.To1(net.ListenUDP("udp", &net.UDPAddr{Port: 7799}))
		defer c.Close()
		udp := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: c})
		eg.SetICEUDPMux(udp)
	}
	// udp := try.To1(ice.NewMultiUDPMuxFromPort(7799))
	// defer udp.Close()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(eg))
	for _, index := range []int{1, 2} {
		log.Println("index", index)
		ttt(api)
	}
}

func ttt(api *webrtc.API) {
	wcfg := webrtc.Configuration{}
	pc := try.To1(api.NewPeerConnection(wcfg))
	dc := try.To1(pc.CreateDataChannel("xhe", nil))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dc.OnOpen(func() {
		log.Println("opennnnn")
		dc.SendText("hello")
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Println("msg", string(msg.Data))
		cancel()
	})
	offer := try.To1(pc.CreateOffer(nil))
	try.To(pc.SetLocalDescription(offer))
	offer = *pc.LocalDescription()
	offerCh <- offer
	answer := <-answerCh
	// log.Println("answer", answer.SDP)
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				log.Println("dc state", dc.ReadyState())
			case <-ctx.Done():
				return
			}
		}
	}()
	try.To(pc.SetRemoteDescription(answer))

	<-ctx.Done()
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
