package utils

import (
	"fmt"
	"net/http"
)

// ProbePort custom port for probe port
var ProbePort int

func livenessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func InitProbe() error {
	http.HandleFunc("/healthz", livenessProbe)
	return http.ListenAndServe(fmt.Sprintf(":%d", ProbePort), nil)
}
