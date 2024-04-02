package server

import (
	"fmt"
	"os"
	"time"

	"github.com/kloudlite/operator/apps/multi-cluster/apps/common"
	"github.com/kloudlite/operator/apps/multi-cluster/constants"
	"github.com/kloudlite/operator/apps/multi-cluster/templates"
	"sigs.k8s.io/yaml"
)

const (
	PEER_CONFIG_PATH = "/tmp/peer-config.json"
)

type PeerMap map[string]struct {
	time time.Time
	common.Peer
}

var peerMap = make(PeerMap)
var config Config

type Config struct {
	Endpoint   string `json:"endpoint"`
	PrivateKey string `json:"privateKey"`
	IpAddress  string `json:"ip"`

	Peers         []common.Peer `json:"peers,omitempty"`
	InternalPeers []common.Peer `json:"internal_peers,omitempty"`
}

func (s *Config) load(cPath string) error {

	if _, err := os.Stat(cPath); err != nil {
		return fmt.Errorf("config file not found: %s", cPath)
	}

	b, err := os.ReadFile(cPath)
	if err != nil {
		return err
	}

	if err := s.ParseYaml(b); err != nil {
		return err
	}

	return nil
}

func (s Config) getAllAllowedIPs() []string {
	var ips []string
	for _, p := range s.Peers {
		ips = append(ips, p.AllowedIPs...)
	}

	ips = append(ips, s.IpAddress)

	return ips
}

func (s *Config) upsertPeer(p common.Peer) bool {
	p.AllowedIPs = []string{fmt.Sprintf("%s/32", p.IpAddress)}

	defer func() {
		peerMap[p.PublicKey] = struct {
			time time.Time
			common.Peer
		}{
			time: time.Now(),
			Peer: p,
		}
	}()

	for i, peer := range s.InternalPeers {
		if peer.PublicKey == p.PublicKey {
			s.InternalPeers[i] = p
			return true
		}
	}

	s.InternalPeers = append(s.InternalPeers, p)
	return true
}

func (s *Config) cleanPeers() {
	for k, v := range peerMap {
		if time.Since(v.time) > constants.ExpiresIn*time.Second {
			delete(peerMap, k)

			for i, p := range s.InternalPeers {
				if p.PublicKey == k {
					s.InternalPeers = append(s.InternalPeers[:i], s.InternalPeers[i+1:]...)
					return
				}
			}
		}
	}

	return
}

func (s *Config) ToYaml() ([]byte, error) {
	return yaml.Marshal(s)
}

func (s *Config) ParseYaml(b []byte) error {
	return yaml.Unmarshal(b, s)
}

func (s *Config) toConfigBytes() ([]byte, error) {
	b, err := templates.ParseTemplate(templates.ServerConfg, *s)
	if err != nil {
		return nil, err
	}

	return b, nil
}
