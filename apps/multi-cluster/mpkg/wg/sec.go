package wg

// import (
// 	"fmt"
// 	"net"
// 	"os"
//
// 	"github.com/kloudlite/operator/pkg/logging"
// 	"golang.zx2c4.com/wireguard/wgctrl"
// 	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
// )
//
// func (c *client) UpsertPeer(pk string, AllowedIPs []net.IPNet, endpoint *string) error {
// 	c.logger.Infof("upserting peer: %s", pk)
//
// 	publicKey, err := wgtypes.ParseKey(pk)
// 	if err != nil {
// 		return err
// 	}
//
// 	peerConfig := wgtypes.PeerConfig{
// 		PublicKey:  publicKey,
// 		AllowedIPs: AllowedIPs,
// 		Endpoint: func() *net.UDPAddr {
// 			if endpoint == nil {
// 				return nil
// 			}
//
// 			addr, err := net.ResolveUDPAddr("udp", *endpoint)
// 			if err != nil {
// 				c.logger.Error(err)
// 				return nil
// 			}
//
// 			fmt.Println("endpoint", addr)
//
// 			return addr
// 		}(),
// 		ReplaceAllowedIPs: true,
// 	}
//
// 	if err := c.client.ConfigureDevice(c.ifName, wgtypes.Config{Peers: []wgtypes.PeerConfig{peerConfig}}); err != nil {
// 		return err
// 	}
//
// 	return nil
// }
//
// func (c *client) RemovePeer(pk string) error {
// 	publicKey, err := wgtypes.ParseKey(pk)
// 	if err != nil {
// 		return err
// 	}
//
// 	if err := c.client.ConfigureDevice(c.ifName,
// 		wgtypes.Config{Peers: []wgtypes.PeerConfig{{PublicKey: publicKey, Remove: true}}},
// 	); err != nil {
// 		return err
// 	}
//
// 	return nil
// }
//
// func (c *client) GetAllowedIps(pk string) ([]net.IPNet, error) {
// 	publicKey, err := wgtypes.ParseKey(pk)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	dev, err := c.client.Device(c.ifName)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	for _, peer := range dev.Peers {
// 		if peer.PublicKey == publicKey {
// 			return peer.AllowedIPs, nil
// 		}
// 	}
//
// 	return nil, fmt.Errorf("peer not found")
// }
//
// func (c *client) IsConfigured() bool {
// 	dev, err := c.client.Device(c.ifName)
// 	if err != nil {
// 		if c.verbose {
// 			c.logger.Error(err)
// 		}
// 		return false
// 	}
//
// 	return len(dev.Peers) > 0
// }
//
// func (c *client) Start(conf []byte) error {
// 	fName := fmt.Sprintf("/etc/wireguard/%s.conf", c.ifName)
// 	if err := os.WriteFile(fName, conf, 0644); err != nil {
// 		return err
// 	}
//
// 	b, err := c.execCmd(fmt.Sprintf("wg-quick up %s", c.ifName), nil)
// 	if err != nil {
// 		c.logger.Infof(string(b))
// 		return err
// 	}
//
// 	return nil
// }
//
// func (c *client) Stop() error {
// 	b, err := c.execCmd(fmt.Sprintf("ip link delete %s", c.ifName), nil)
// 	if err != nil {
// 		c.logger.Infof(string(b))
// 		return err
// 	}
//
// 	return nil
// }
//
// func (c *client) GetPrivateKey() (string, error) {
// 	dev, err := c.client.Device(c.ifName)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return dev.PrivateKey.String(), nil
// }
//
// func (c *client) GetPublicKey() (string, error) {
// 	dev, err := c.client.Device(c.ifName)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return dev.PublicKey.String(), nil
// }
//
// func (c *client) GetKeys() (string, string, error) {
// 	privateKey, err := c.GetPrivateKey()
// 	if err != nil {
// 		return "", "", err
// 	}
//
// 	publicKey, err := c.GetPublicKey()
// 	if err != nil {
// 		return "", "", err
// 	}
//
// 	return publicKey, privateKey, nil
// }
//
// func (c *client) Close() error {
// 	return c.client.Close()
// }
//
// type client struct {
// 	client  *wgctrl.Client
// 	logger  logging.Logger
// 	verbose bool
// 	ifName  string
// }
//
// func NewClient(ifName string, logger logging.Logger, verbose bool) (*client, error) {
//
// 	if ifName == "" {
// 		return nil, fmt.Errorf("ifName is required")
// 	}
//
// 	if logger == nil {
// 		var err error
// 		logger, err = logging.New(&logging.Options{})
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	c, err := wgctrl.New()
// 	if err != nil {
// 		if verbose {
// 			logger.Error(err)
// 		}
// 		return nil, err
// 	}
//
// 	return &client{
// 		logger:  logger,
// 		ifName:  ifName,
// 		verbose: verbose,
// 		client:  c,
// 	}, nil
// }
