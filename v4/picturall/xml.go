package picturall

import (
	"encoding/xml"
)

type MediaCollections struct {
	XMLName     xml.Name        `xml:"mediacollections"`
	Timestamp   string          `xml:"timestamp,attr"`
	CreatorID   string          `xml:"creatorid,attr"`
	DataID      string          `xml:"dataid,attr"`
	Collections []XMLCollection `xml:"collection"`
}

type XMLCollection struct {
	XMLName xml.Name   `xml:"collection"`
	Name    string     `xml:"name,attr"`
	Medias  []XMLMedia `xml:"media"`
}

type XMLMedia struct {
	XMLName  xml.Name `xml:"media"`
	Name     string   `xml:"name,attr"`
	File     string   `xml:"file,attr"`
	PlayMode int      `xml:"default_play_mode,attr"`
	Type     string   `xml:"type,attr"`
	Index    int      `xml:"index,attr"`
	Duration float64  `xml:"duration,attr"`
}
