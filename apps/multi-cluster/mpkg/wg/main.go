package wg

import (
	"fmt"
	"os"
	"strings"

	"github.com/kloudlite/operator/pkg/logging"
)

const interfaceName = "kl-wg"

func startWg(logger logging.Logger, conf []byte) error {
	if err := os.WriteFile(fmt.Sprintf("/etc/wireguard/%s.conf", interfaceName), conf, 0644); err != nil {
		return err
	}

	b, err := ExecCmd(fmt.Sprintf("wg-quick up %s", interfaceName), nil, logger, true)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Infof("wireguard started: %s", string(b))
	return nil
}

func updateWg(logger logging.Logger, conf []byte) error {

	fName := fmt.Sprintf("/etc/wireguard/%s.conf", interfaceName)

	if _, err := os.Stat(fName); os.IsNotExist(err) {
		if err := os.WriteFile(fName, conf, 0644); err != nil {
			return err
		}
	} else {
		b, err := os.ReadFile(fName)
		if err != nil {
			return err
		}

		if string(b) == string(conf) {
			logger.Infof("configuration is the same, skipping update")
			return nil
		}

		if err := os.WriteFile(fName, conf, 0644); err != nil {
			return err
		}
	}

	b, err := ExecCmd(fmt.Sprintf("wg-quick down %s", interfaceName), nil, logger, true)
	if err != nil {
		logger.Error(err)
		fmt.Println(string(b))
		return err
	}

	if err := os.WriteFile(fName, conf, 0644); err != nil {
		return err
	}

	b, err = ExecCmd(fmt.Sprintf("wg-quick up %s", interfaceName), nil, logger, true)

	if err != nil {
		logger.Error(err)
		fmt.Println(string(b))
		return err
	}

	logger.Infof("wireguard updated")

	// fName := fmt.Sprintf("/etc/wireguard/%s.conf", interfaceName)
	// if _, err := os.Stat(fName); os.IsNotExist(err) {
	// 	if err := os.WriteFile(fName, conf, 0644); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	b, err := os.ReadFile(fName)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	if string(b) == string(conf) {
	// 		logger.Infof("configuration is the same, skipping update")
	// 		return nil
	// 	}
	//
	// 	if err := os.WriteFile(fName, conf, 0644); err != nil {
	// 		return err
	// 	}
	// }
	//
	// cmd1 := exec.Command("wg-quick", "strip", interfaceName)
	// var out1 bytes.Buffer
	// cmd1.Stdout = &out1
	// if err := cmd1.Run(); err != nil {
	// 	return err
	// }
	//
	// cmd2 := exec.Command("wg", "syncconf", interfaceName, "/dev/stdin")
	// cmd2.Stdin = bytes.NewReader(out1.Bytes())
	// if err := cmd2.Run(); err != nil {
	// 	return err
	// }
	//
	// logger.Infof("configuration updated")
	// return nil

	return nil
}

func ResyncWg(logger logging.Logger, conf []byte) error {
	b, err := ExecCmd(fmt.Sprintf("wg show %s", interfaceName), nil, logger, true)
	if err != nil && strings.Contains(string(b), "No such device") || !strings.Contains(string(b), fmt.Sprintf("interface: %s", interfaceName)) {
		logger.Infof("wireguard is not running, starting it")
		if err := startWg(logger, conf); err != nil {
			logger.Error(err)
		}
		return nil
	}

	logger.Infof("wireguard is already running, updating configuration")
	if err := updateWg(logger, conf); err != nil {
		logger.Error(err)
	}
	return nil
}
