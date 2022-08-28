package picturall

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"gitlab.com/clock-8001/clock-8001/v4/debug"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net/http"
)

// MediaCollections is the parsed picturall media collection
type MediaCollections struct {
	XMLName     xml.Name        `xml:"mediacollections"`
	Timestamp   string          `xml:"timestamp,attr"`
	CreatorID   string          `xml:"creatorid,attr"`
	DataID      string          `xml:"dataid,attr"`
	Collections []XMLCollection `xml:"collection"`
}

// XMLCollection is the <collection> element of the picturall media collection xml
type XMLCollection struct {
	XMLName xml.Name   `xml:"collection"`
	Name    string     `xml:"name,attr"`
	Medias  []XMLMedia `xml:"media"`
}

// XMLMedia is the <media> element from picturall media collection xml
type XMLMedia struct {
	XMLName  xml.Name `xml:"media"`
	Name     string   `xml:"name,attr"`
	File     string   `xml:"file,attr"`
	PlayMode int      `xml:"default_play_mode,attr"`
	Type     string   `xml:"type,attr"`
	Index    int      `xml:"index,attr"`
	Duration float64  `xml:"duration,attr"`
}

// FetchMedia connects to a picturall and tries to fetch the media collection
func FetchMedia(ip string) (*MediaCollections, error) {
	url := fmt.Sprintf("http://%s/filedownload/media_collection.xml", ip)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Picturall: Error downloading media xml: %w", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Picturall: Error reading response body: %w", err)
	}

	mc, err := parseXML(bytes)
	if err != nil {
		return nil, err
	}
	debug.Printf("Picturall: Updated media collection")
	return mc, nil
}

// parseXML takes a byte slice and attempts to parse it as a picturall media collection xml
func parseXML(x []byte) (*MediaCollections, error) {
	mc := &MediaCollections{}
	err := xml.Unmarshal(x[:len(x)-1], mc)
	if err != nil {
		return nil, fmt.Errorf("Picturall: parseXML error: %w", err)
	}
	return mc, nil
}

func decodeXML(data []byte) ([]byte, error) {
	// Make an tranformer that converts MS-Win default to UTF8:
	utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	// Make a transformer that is like win16be, but abides by BOM:
	utf16bom := unicode.BOMOverride(utf16le.NewDecoder())

	// Make a Reader that uses utf16bom:
	unicodeReader := transform.NewReader(bytes.NewReader(data), utf16bom)

	// decode and print:
	x, err := ioutil.ReadAll(unicodeReader)

	if err != nil {
		return nil, fmt.Errorf("Picturall: XML parsing error: %w", err)
	}
	return x, err
}

// parseMediaCollections takes the raw byte slice payload from port 11001 and attempts to parse it
func parseMediaCollections(data []byte) (*MediaCollections, error) {
	x, err := decodeXML(data)
	if err != nil {
		return nil, err
	}
	mc, err := parseXML(x)
	if err != nil {
		return nil, err
	}
	return mc, nil
}

func (mc *MediaCollections) String() string {
	ret := "Media collection:\n"
	for i, c := range mc.Collections {
		if len(c.Medias) == 0 {
			continue
		}
		ret += fmt.Sprintf(" -> Collection %d: %s\n", i, c.Name)
		for _, m := range c.Medias {
			ret += fmt.Sprintf("   -> %d Media: %s type: %s play mode: %d\n", m.Index, m.Name, m.Type, m.PlayMode)
		}
	}
	return ret
}
