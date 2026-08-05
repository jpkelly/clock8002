package picturall

import (
	"encoding/xml"
	"fmt"
	"github.com/jpkelly/clock8002/app/debug"
	"io"
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
		return nil, fmt.Errorf("picturall: Error downloading media xml: %w", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("picturall: Error reading response body: %w", err)
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
		return nil, fmt.Errorf("picturall: parseXML error: %w", err)
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
