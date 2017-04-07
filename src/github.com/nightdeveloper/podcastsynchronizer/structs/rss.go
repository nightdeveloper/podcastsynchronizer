package structs

type EnclosureStruct struct {
	Type	string		`xml:"type,attr"`
	URL	string		`xml:"url,attr"`
}

type ItemStruct struct {
	Title	string		`xml:"title"`
	Guid	string		`xml:"guid"`
	PubDate	string		`xml:"pubDate"`
	Enclosure EnclosureStruct `xml:"enclosure"`
}

type ChannelStruct struct {
	Title	string		`xml:"title"`
	Link	string		`xml:"link"`
	Item	[]ItemStruct	`xml:"item"`
}

type RSSStruct struct {
	Channel	ChannelStruct	`xml:"channel"`
}
