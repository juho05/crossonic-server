package responses

import (
	"encoding/json"
	"encoding/xml"
	"io"

	"github.com/juho05/crossonic-server"
)

const (
	apiVersion = "1.16.1"
	xmlns      = "http://subsonic.org/restapi"
)

type status string

const (
	statusOK     = "ok"
	statusFailed = "failed"
)

type Response struct {
	Status        status `xml:"status,attr" json:"status"`
	Version       string `xml:"version,attr" json:"version"`
	XMLNS         string `xml:"xmlns,attr" json:"-"`
	Type          string `xml:"type,attr" json:"type"`
	ServerVersion string `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool   `xml:"openSubsonic,attr" json:"openSubsonic"`
	Crossonic     bool   `xml:"crossonic,attr" json:"crossonic"`

	Error                  *Error                  `xml:"error" json:"error,omitempty"`
	OpenSubsonicExtensions *OpenSubsonicExtensions `xml:"openSubsonicExtensions" json:"openSubsonicExtensions,omitempty"`
}

func New() Response {
	return Response{
		Status:        statusFailed,
		Version:       apiVersion,
		XMLNS:         xmlns,
		Type:          crossonic.ServerName,
		ServerVersion: crossonic.Version,
		OpenSubsonic:  true,
		Crossonic:     true,
	}
}

func EncodeError(w io.Writer, format, msg string, code SubsonicError) error {
	r := Response{
		Status:        statusFailed,
		Version:       apiVersion,
		XMLNS:         xmlns,
		Type:          crossonic.ServerName,
		ServerVersion: crossonic.Version,
		OpenSubsonic:  true,
		Crossonic:     true,
		Error: &Error{
			Code:    code,
			Message: msg,
		},
	}
	return r.Encode(w, format)
}

func (r Response) Encode(w io.Writer, format string) error {
	if format == "json" {
		return json.NewEncoder(w).Encode(r)
	}
	encoder := xml.NewEncoder(w)
	err := encoder.Encode(r)
	if err != nil {
		return err
	}
	return encoder.Close()
}
