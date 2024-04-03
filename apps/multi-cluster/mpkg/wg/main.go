package wg

import (
	"fmt"
	"os"
	"strings"

	"github.com/kloudlite/operator/apps/multi-cluster/constants"
	"github.com/kloudlite/operator/pkg/logging"
)

func (c *client) Stop() error {
	return nil
}

func (c *client) startWg(conf []byte) error {
	if err := os.WriteFile(fmt.Sprintf("/etc/wireguard/%s.conf", c.ifName), conf, 0644); err != nil {
		return err
	}

	b, err := c.execCmd(fmt.Sprintf("wg-quick up %s", c.ifName), nil)
	if err != nil {
		c.logger.Error(err)
		return err
	}

	c.logger.Infof("wireguard started: %s", string(b))
	return nil
}

var prevConf []byte

func (c *client) updateWg(conf []byte) error {

	fName := fmt.Sprintf("/etc/wireguard/%s.conf", c.ifName)

	if prevConf != nil && string(prevConf) == string(conf) {
		c.logger.Infof("configuration is the same, skipping update")
		return nil
	}

	prevConf = conf

	if err := os.WriteFile(fName, conf, 0644); err != nil {
		return err
	}

	b, err := c.execCmd(fmt.Sprintf("wg-quick down %s", c.ifName), nil)
	if err != nil {
		c.logger.Error(err)
		fmt.Println(string(b))
		return err
	}

	if err := os.WriteFile(fName, conf, 0644); err != nil {
		return err
	}

	b, err = c.execCmd(fmt.Sprintf("wg-quick up %s", c.ifName), nil)

	if err != nil {
		c.logger.Error(err)
		fmt.Println(string(b))
		return err
	}

	c.logger.Infof("wireguard updated")

	return nil
}

func (c *client) Sync(conf []byte) error {
	b, err := c.execCmd(fmt.Sprintf("wg show %s", c.ifName), nil)
	if err != nil && strings.Contains(string(b), "No such device") || !strings.Contains(string(b), fmt.Sprintf("interface: %s", c.ifName)) {
		c.logger.Infof("wireguard is not running, starting it")
		if err := c.startWg(conf); err != nil {
			c.logger.Error(err)
		}
		return nil
	}

	c.logger.Infof("wireguard is already running, updating configuration")
	if err := c.updateWg(conf); err != nil {
		c.logger.Error(err)
	}
	return nil
}

type client struct {
	logger  logging.Logger
	ifName  string
	verbose bool
}

type Client interface {
	Sync(conf []byte) error
	Stop() error
}

func NewClient() (Client, error) {
	l, err := logging.New(&logging.Options{})
	if err != nil {
		return nil, err
	}

	return &client{
		logger:  l.WithName("wgc"),
		ifName:  constants.IfaceName,
		verbose: false,
	}, nil
}
