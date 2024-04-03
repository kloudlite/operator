package wg

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"os/exec"
	"strings"
)

func (c *client) execCmd(cmdString string, env map[string]string) ([]byte, error) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}

	r := csv.NewReader(strings.NewReader(cmdString))
	r.Comma = ' '
	cmdArr, err := r.Read()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	if c.verbose {
		c.logger.Infof(strings.Join(cmdArr, " "))

		cmd.Stdout = out
	}

	if env == nil {
		env = map[string]string{}
	}

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return errOut.Bytes(), err
	}
	return out.Bytes(), nil
}
