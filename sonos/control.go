package sonos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/log"
)

var ErrNotFound = errors.New("not found")

func (s *SonosController) Play(device string) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: play: %w", ErrNotFound)
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/play", ip), nil)
	if err != nil {
		return fmt.Errorf("sonos: play: %w", err)
	}
	return nil
}

func (s *SonosController) Pause(device string) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: pause: %w", ErrNotFound)
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/pause", ip), nil)
	if err != nil {
		return fmt.Errorf("sonos: pause: %w", err)
	}
	return nil
}

func (s *SonosController) Stop(device string) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: stop: %w", ErrNotFound)
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/stop", ip), nil)
	if err != nil {
		return fmt.Errorf("sonos: stop: %w", err)
	}
	return nil
}

func (s *SonosController) SetCurrent(device, url string, nextURL *string) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: set current: %w", ErrNotFound)
	}
	type req struct {
		URI     string  `json:"uri"`
		NextURI *string `json:"next_uri"`
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/setCurrent", ip), req{
		URI:     url,
		NextURI: nextURL,
	})
	if err != nil {
		return fmt.Errorf("sonos: current: %w", err)
	}
	return nil
}

func (s *SonosController) SetNext(device string, url *string) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: set next: %w", ErrNotFound)
	}
	type req struct {
		URI *string `json:"uri"`
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/setNext", ip), req{
		URI: url,
	})
	if err != nil {
		return fmt.Errorf("sonos: set next: %w", err)
	}
	return nil
}

func (s *SonosController) GetPosition(device string) (time.Duration, error) {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return 0, fmt.Errorf("sonos: get position: %w", ErrNotFound)
	}
	type response struct {
		Seconds int `json:"seconds"`
	}
	res, err := controllerJSONRequest[response](s, fmt.Sprintf("/%s/getPosition", ip), nil)
	if err != nil {
		return 0, fmt.Errorf("sonos: get position: %w", err)
	}
	return time.Duration(res.Seconds) * time.Second, nil
}

func (s *SonosController) GetVolume(device string) (int, error) {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return 0, fmt.Errorf("sonos: get volume: %w", ErrNotFound)
	}
	type response struct {
		Volume int `json:"volume"`
	}
	res, err := controllerJSONRequest[response](s, fmt.Sprintf("/%s/getVolume", ip), nil)
	if err != nil {
		return 0, fmt.Errorf("sonos: get volume: %w", err)
	}
	return res.Volume, nil
}

func (s *SonosController) SetVolume(device string, volume int) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: set volume: %w", ErrNotFound)
	}
	type req struct {
		Volume int `json:"volume"`
	}
	_, err := controllerJSONRequest[struct{}](s, fmt.Sprintf("/%s/setVolume", ip), req{
		Volume: volume,
	})
	if err != nil {
		return fmt.Errorf("sonos: set volume: %w", err)
	}
	return nil
}

func (s *SonosController) OnEvent(ctx context.Context, device string, callback func(state string)) error {
	s.devicesLock.RLock()
	ip, ok := s.devices[device]
	s.devicesLock.RUnlock()
	if !ok {
		return fmt.Errorf("sonos: on event: %w", ErrNotFound)
	}
	conn, _, _, err := ws.Dial(ctx, fmt.Sprintf("ws%s/%s/events", strings.TrimPrefix(config.SonosControllerURL(), "http"), ip))
	if err != nil {
		return fmt.Errorf("sonos: on event: %w", err)
	}
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	type event struct {
		State string `json:"state"`
	}
	go func() {
		defer conn.Close()
		for {
			data, err := wsutil.ReadServerText(conn)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return
				}
				log.Errorf("sonos: on event: %s", err)
				return
			}

			var ev event
			err = json.Unmarshal(data, &ev)
			if err != nil {
				log.Errorf("sonos: decode message: %s", err)
				continue
			}
			callback(ev.State)
		}
	}()
	return nil
}
