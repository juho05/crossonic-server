package sonos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/log"
)

type SonosController struct {
	onNewDevice     func(string)
	onDeviceRemoved func(string)

	refreshDevicesTimerRunning bool
	sonosURL                   string

	devicesLock sync.RWMutex
	devices     map[string]string
}

// returns nil if SONOS_CMD is empty
func NewController(onNewDevice func(string), onDeviceRemoved func(string)) *SonosController {
	sonosURL := config.SonosControllerURL()
	if sonosURL == "" {
		return nil
	}
	return &SonosController{
		refreshDevicesTimerRunning: false,
		sonosURL:                   sonosURL,
		devices:                    make(map[string]string),
		onNewDevice:                onNewDevice,
		onDeviceRemoved:            onDeviceRemoved,
	}
}

func (s *SonosController) StartRefreshDevicesTimer() {
	if s == nil {
		return
	}
	if s.refreshDevicesTimerRunning {
		return
	}
	s.refreshDevicesTimerRunning = true
	go func() {
		for s.refreshDevicesTimerRunning {
			err := s.refreshDevices()
			if err != nil {
				log.Error("sonos:", err)
			}
			time.Sleep(5 * time.Minute)
		}
	}()
}

func (s *SonosController) StopRefreshDevicesTimer() {
	if s == nil {
		return
	}
	s.refreshDevicesTimerRunning = false
}

func (s *SonosController) Devices() []string {
	if s == nil {
		return make([]string, 0)
	}
	s.devicesLock.RLock()
	devices := make([]string, 0, len(s.devices))
	for d := range s.devices {
		devices = append(devices, d)
	}
	s.devicesLock.RUnlock()
	return devices
}

func (s *SonosController) refreshDevices() error {
	type device struct {
		Name   string `json:"name"`
		IpAddr string `json:"ip_addr"`
	}
	devices, err := controllerJSONRequest[[]device](s, "/getDevices", nil)
	if err != nil {
		return fmt.Errorf("refresh devices: %w", err)
	}
	s.devicesLock.Lock()
	newDevices := make(map[string]string, len(s.devices))
	for _, d := range devices {
		newDevices[d.Name] = d.IpAddr
		if _, ok := s.devices[d.Name]; !ok {
			s.onNewDevice(d.Name)
		}
	}
	for d := range s.devices {
		if _, ok := newDevices[d]; !ok {
			s.onDeviceRemoved(d)
		}
	}
	s.devices = newDevices
	s.devicesLock.Unlock()
	return nil
}

func controllerJSONRequest[T any](s *SonosController, endpoint string, body any) (T, error) {
	var obj T
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return obj, fmt.Errorf("controller json request: marshal request body: %w", err)
		}
	}

	res, err := http.Post(s.sonosURL+endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return obj, fmt.Errorf("controller json request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return obj, fmt.Errorf("controller json request: unexpected response code: %d", res.StatusCode)
	}

	err = json.NewDecoder(res.Body).Decode(&obj)
	res.Body.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return obj, fmt.Errorf("controller json request: %w", err)
	}
	return obj, nil
}
