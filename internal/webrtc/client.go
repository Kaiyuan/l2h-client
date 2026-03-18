package webrtc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pion/webrtc/v3"
)

type SignalMessage struct {
	APIKey string `json:"api_key"`
	SDP    string `json:"sdp"`
	Type   string `json:"type"`
}

type ProxyRequest struct {
	RequestId  string            `json:"requestId"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	TargetPort int               `json:"targetPort"`
}

type ProxyResponse struct {
	RequestId string            `json:"requestId"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

func ConnectToServer(serverURL, apiKey string) error {
	// 1. Create WebRTC PeerConnection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}

	// 2. Create DataChannel
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return err
	}

	dataChannel.OnOpen(func() {
		fmt.Println("Data channel is open")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		var req ProxyRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			fmt.Printf("Failed to unmarshal DC message: %v\n", err)
			return
		}

		fmt.Printf("Proxying request: %s %s -> localhost:%d\n", req.Method, req.Path, req.TargetPort)

		// Create local request
		url := fmt.Sprintf("http://localhost:%d%s", req.TargetPort, req.Path)
		clientReq, err := http.NewRequest(req.Method, url, nil) // Body support can be added later
		if err != nil {
			sendError(dataChannel, req.RequestId, 500, err.Error())
			return
		}

		// Copy headers
		for k, v := range req.Headers {
			if !isForbiddenHeader(k) {
				clientReq.Header.Set(k, v)
			}
		}

		resp, err := http.DefaultClient.Do(clientReq)
		if err != nil {
			sendError(dataChannel, req.RequestId, 502, err.Error())
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[k] = strings.Join(v, ", ")
		}

		proxyResp := ProxyResponse{
			RequestId: req.RequestId,
			Status:    resp.StatusCode,
			Headers:   headers,
			Body:      string(body),
		}

		respData, _ := json.Marshal(proxyResp)
		dataChannel.Send(respData)
	})

	// 3. Create Offer
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return err
	}

	// 4. Wait for ICE gathering
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return err
	}
	<-gatherComplete
	
	offer = *peerConnection.LocalDescription()

	// 5. Send Offer to Server via Signaling
	signalReq := SignalMessage{
		APIKey: apiKey,
		SDP:    offer.SDP,
		Type:   "offer",
	}

	payload, err := json.Marshal(signalReq)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/webrtc/signal", serverURL), "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signaling failed: %s", string(body))
	}

	var signalResp SignalMessage
	if err := json.NewDecoder(resp.Body).Decode(&signalResp); err != nil {
		return err
	}

	if signalResp.Type == "answer" {
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  signalResp.SDP,
		}
		if err := peerConnection.SetRemoteDescription(answer); err != nil {
			return err
		}
		fmt.Println("Answer received and set")
	}

	fmt.Println("WebRTC connection established (signaling phase done)")
	return nil
}

func sendError(dc *webrtc.DataChannel, requestId string, status int, msg string) {
	resp := ProxyResponse{
		RequestId: requestId,
		Status:    status,
		Body:      msg,
	}
	data, _ := json.Marshal(resp)
	dc.Send(data)
}

func isForbiddenHeader(h string) bool {
	h = strings.ToLower(h)
	return h == "host" || h == "connection" || h == "upgrade" || h == "proxy-connection" || h == "transfer-encoding"
}
