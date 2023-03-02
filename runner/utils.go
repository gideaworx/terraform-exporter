package runner

import (
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
)

func GetHTTPClient() *http.Client {
	skipVerify, _ := strconv.ParseBool(os.Getenv("TF_EXPORTER_INSTALL_SKIP_TLS_VERIFY"))

	hc := http.DefaultClient
	hc.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
	}

	return hc
}
