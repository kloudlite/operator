package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"

	wireguardv1 "github.com/kloudlite/operator/apis/wireguard/v1"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	corev1 "k8s.io/api/core/v1"
)

func getNs(obj *wireguardv1.Server) string {
	return fmt.Sprintf("wg-%s", obj.Name)
}

func parseWgSec(sec *corev1.Secret) (pub []byte, priv []byte, err error) {
	var ok bool
	if pub, ok = sec.Data["public-key"]; !ok {
		return nil, nil, fmt.Errorf("can't fetch public key from secret")
	}
	if priv, ok = sec.Data["private-key"]; !ok {
		return nil, nil, fmt.Errorf("can't fetch private key from secret")
	}

	return pub, priv, nil
}

func parseDevSec(sec corev1.Secret) (ip []byte, pub []byte, err error) {
	var ok bool
	ip, ok = sec.Data["ip"]
	if !ok {
		return nil, nil, fmt.Errorf("can't get ip from device secret")
	}

	pub, ok = sec.Data["public-key"]
	if !ok {
		return nil, nil, fmt.Errorf("can't get public-key from device secret")
	}

	return ip, pub, nil
}

func isContains(svce map[string]*configService, port int32) bool {
	for _, s := range svce {
		if s.ServicePort == port {
			return true
		}
	}
	return false
}

func getTempPort(svcs map[string]*configService, id string, configData map[string]*configService) int32 {
	if svcs[id] != nil {
		return svcs[id].ProxyPort
	}

	return func() int32 {
		min, max := 3000, 6000

		count := 0
		var r int

		for {
			r = rand.Intn(max-min) + min
			if !isContains(configData, int32(r)) || count > max-min {
				break
			}

			count++
		}

		return int32(r)
	}()
}

func getRemoteDeviceIp(deviceOffcet int64) ([]byte, error) {
	deviceRange := ipaddr.NewIPAddressString("10.13.0.0/16")

	if address, addressError := deviceRange.ToAddress(); addressError == nil {
		increment := address.Increment(deviceOffcet + 2)
		return []byte(ipaddr.NewIPAddressString(increment.GetNetIP().String()).String()), nil
	} else {
		return nil, addressError
	}
}

func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

func JSONStringsEqual(a, b string) (bool, error) {
	return JSONBytesEqual([]byte(a), []byte(b))
}
