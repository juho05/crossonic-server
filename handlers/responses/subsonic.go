package responses

type Error struct {
	Code    SubsonicError `xml:"code,attr" json:"code"`
	Message string        `xml:"message,attr" json:"message"`
}

type OpenSubsonicExtensions []OpenSubsonicExtension

type OpenSubsonicExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}

type License struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

type ScanStatus struct {
	Scanning bool `xml:"scanning,attr" json:"scanning"`
	Count    *int `xml:"count,attr,omitempty" json:"count,omitempty"`
}
