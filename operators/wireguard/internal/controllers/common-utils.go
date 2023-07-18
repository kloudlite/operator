package wgctrl_utils

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GenerateWgKeys() ([]byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	return []byte(key.PublicKey().String()), []byte(key.String()), nil
}
