package webrtc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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
	Body       string            `json:"body"` // Base64 encoded
}

type ProxyResponse struct {
	RequestId string            `json:"requestId"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

// ConnectWithRetry maintains the connection with exponential backoff
func ConnectWithRetry(serverURL, apiKey string) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		fmt.Printf("正在连接服务器: %s\n", serverURL)
		err := ConnectToServer(serverURL, apiKey)
		if err != nil {
			fmt.Printf("连接错误: %v\n", err)
		}
		
		fmt.Printf("连接丢失或失败。正在重试，等待时间 %v...\n", backoff)
		time.Sleep(backoff)
		
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func ConnectToServer(serverURL, apiKey string) error {
	// 0. 从服务端动态获取 ICE 配置
	iceConfig, err := getRemoteICEConfig(serverURL)
	if err != nil {
		fmt.Printf("警告：无法从服务端获取动态 ICE 配置，将使用默认值: %v\n", err)
		iceConfig = []string{"stun:stun.cloudflare.com:3478"}
	}

	// 1. Create WebRTC PeerConnection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: iceConfig,
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}
	defer peerConnection.Close()

	done := make(chan struct{})
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("连接状态已变更: %s \n", s.String())
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})

	// 2. DataChannel handle (处理数据通道)
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		fmt.Printf("新建 DataChannel %s %d\n", dc.Label(), dc.ID())
		dc.OnOpen(func() {
			fmt.Println("数据通道已打开")
		})
		
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			handleMessage(dc, msg)
		})
		
		dc.OnClose(func() {
			fmt.Println("数据通道已关闭")
		})
	})

	// Client creates DataChannel to initiate the first one
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return err
	}
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		handleMessage(dataChannel, msg)
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
		fmt.Println("已收到并设置 Answer")
	}

	fmt.Println("WebRTC 连接已建立（信令阶段完成）。正在等待连接...")
	
	// 等待直到连接关闭或失败
	<-done
	return nil
}

func handleMessage(dc *webrtc.DataChannel, msg webrtc.DataChannelMessage) {
	fmt.Printf("收到 DC 消息: %s\n", string(msg.Data))
	var req ProxyRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		fmt.Printf("解析 DC 消息失败: %v (内容: %s)\n", err, string(msg.Data))
		return
	}

	fmt.Printf("[%s] 正在代理请求: %s %s -> localhost:%d\n", req.RequestId, req.Method, req.Path, req.TargetPort)

	// Decode body
	var bodyReader io.Reader
	if req.Body != "" {
		bodyBytes, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			fmt.Printf("解码 Body 失败: %v\n", err)
			sendError(dc, req.RequestId, 400, "无效的 Base64 Body")
			return
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create local request
	url := fmt.Sprintf("http://localhost:%d%s", req.TargetPort, req.Path)
	clientReq, err := http.NewRequest(req.Method, url, bodyReader)
	if err != nil {
		sendError(dc, req.RequestId, 500, err.Error())
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
		sendError(dc, req.RequestId, 502, err.Error())
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
	dc.Send(respData)
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
	// Some headers are hop-by-hop or managed by http.Client
	return h == "connection" || h == "upgrade" || h == "proxy-connection" || h == "transfer-encoding"
}

func getRemoteICEConfig(serverURL string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/webrtc/config", serverURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		IceServers []string `json:"iceServers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.IceServers, nil
}

