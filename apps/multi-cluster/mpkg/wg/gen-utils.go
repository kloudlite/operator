package wg

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func Ptr[T any](t T) *T {
	return &t
}

func GenerateWgKeys() ([]byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	return []byte(key.PublicKey().String()), []byte(key.String()), nil
}

func GeneratePublicKey(privateKey string) ([]byte, error) {
	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return nil, err
	}

	return []byte(key.PublicKey().String()), nil
}
