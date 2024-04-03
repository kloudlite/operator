package client

import (
	// "fmt"
	// "os"

	"github.com/kloudlite/operator/apps/multi-cluster/apps/client/env"
	"github.com/kloudlite/operator/apps/multi-cluster/mpkg/wg"
	"github.com/kloudlite/operator/pkg/logging"
)

func Run() error {
	env := env.GetEnvOrDie()

	// ip, ok := os.LookupEnv("MY_IP_ADDRESS")
	// if !ok {
	// 	return fmt.Errorf("MY_IP_ADDRESS not set")
	// }

	pub, priv, err := wg.GenerateWgKeys()
	if err != nil {
		return err
	}

	l, err := logging.New(&logging.Options{})
	if err != nil {
		return err
	}

	c := &client{
		logger:     l,
		env:        env,
		privateKey: priv,
		publicKey:  pub,
		// IpAddress:  generateIp(ip),
	}

	return c.start()
}
