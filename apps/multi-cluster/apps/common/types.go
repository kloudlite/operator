package common

import "encoding/json"

type Peer struct {
	PublicKey string `json:"publicKey"`
	IpAddress  string   `json:"ip"`
	AllowedIPs []string `json:"allowedIPs,omitempty"`
	Endpoint   string   `json:"endpoint,omitempty"`
}

func (p *Peer) parseJson(b []byte) error {
	return json.Unmarshal(b, p)
}

type PeerReq struct {
	PublicKey  string   `json:"publicKey"`
	IpAddress  string   `json:"ipAddress"`
	Endpoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowedIPs,omitempty"`
}

func (s *PeerReq) ToJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s *PeerReq) ParseJson(b []byte) error {
	return json.Unmarshal(b, s)
}

type PeerResp struct {
	PublicKey  string   `json:"publicKey"`
	EndPoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowedIPs,omitempty"`
}

func (s *PeerResp) ToJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s *PeerResp) ParseJson(b []byte) error {
	return json.Unmarshal(b, s)
}
