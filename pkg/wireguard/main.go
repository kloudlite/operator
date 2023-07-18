package wireguard

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GenerateWgKeys() (string, string, error) {

	key, err := wgtypes.GenerateKey()
	if err != nil {
		return "", "", err
	}

	return key.PublicKey().String(), key.String(), nil
}
