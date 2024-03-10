package main

import (
	"context"
	"log"
	"net"
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
	eg := webrtc.SettingEngine{}
	conn := try.To1(net.ListenUDP("udp", &net.UDPAddr{Port: 7799}))
	defer conn.Close()
	udp := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: conn})
	eg.SetICEUDPMux(udp)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(eg))
	var wg sync.WaitGroup
	for _, index := range []int{1, 2, 3} {
		wg.Add(1)
		func() {
			defer wg.Done()
			log.Println("index", index)
			ttt(api, index)
		}()
	}
	wg.Wait()
}

func ttt(api *webrtc.API, index int) {
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
	// log.Println("offer1", index, offer.SDP)
	// <-webrtc.GatheringCompletePromise(pc)
	offer = *pc.LocalDescription()
	offerCh <- offer
	answer := <-answerCh
	// log.Println("offer", index, offer.SDP)
	// log.Println("answer", index, answer.SDP)
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				log.Println("dc state", index, dc.ReadyState())
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
