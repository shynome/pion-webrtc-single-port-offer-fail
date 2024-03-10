package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/pion/ice/v3"
	"github.com/pion/webrtc/v4"
	"github.com/shynome/err0/try"
)

func main() {
	index := os.Args[1]
	log.Println("index", index)
	eg := webrtc.SettingEngine{}
	conn := try.To1(net.ListenUDP("udp", &net.UDPAddr{Port: 7799}))
	defer conn.Close()
	udp := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: conn})
	eg.SetICEUDPMux(udp)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(eg))

	wcfg := webrtc.Configuration{}
	pc := try.To1(api.NewPeerConnection(wcfg))
	// defer pc.Close()
	dc := try.To1(pc.CreateDataChannel("xhe", nil))
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
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
	log.Println("offer write")
	try.To1(io.WriteString(os.Stdout, offer.SDP))
	log.Println("offer writed")
	//
	log.Println("answer read")
	answerRaw := try.To1(io.ReadAll(os.Stdin))
	log.Println("answer readed")
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(answerRaw),
	}
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
