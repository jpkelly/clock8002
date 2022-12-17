package millumin

import (
	"fmt"
	// "gitlab.com/Depili/go-osc/osc"
	"github.com/chabad360/go-osc/osc"

	"regexp"
)

func parseAddressLayer(msg *osc.Message, layerPtr *string) error {
	return parseAddress(msg, regexp.MustCompile("/layer:(.*?)/"), layerPtr)
}

func parseAddress(msg *osc.Message, pattern *regexp.Regexp, values ...*string) error {
	if matches := pattern.FindStringSubmatch(msg.Address); matches == nil {
		return fmt.Errorf("address did not match pattern %v: %v", pattern, msg.Address)
	} else if len(matches) != len(values)+1 {
		return fmt.Errorf("address pattern values count %d mismatch: %d", len(values), len(matches)-1)
	} else {
		for i, value := range matches {
			if i == 0 {
				continue
			}

			*values[i-1] = value
		}

		return nil
	}
}
