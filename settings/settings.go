package settings

import (
	"encoding/json"
	"log"
	"time"
	"path/filepath"
	"io/ioutil"
)

type Filter struct {
	Title		string		`json:"title"`
}

type Podcast struct {
	Url		string		`json:"url"`
	MaxDepth	int		`json:"maxDepth,omitempty"`
	Filters		[]*Filter	`json:"filters,omitempty"`
	Name		string		`json:"name,omitempty"`
	LastUpdated	time.Time	`json:"lastUpdated,omitempty"`
	LastChecked	time.Time	`json:"lastChecked,omitempty"`
	LastGuid	string		`json:"lastGuid,omitempty"`
	Status		string		`json:"status,omitempty"`
}

type Config struct {
	DropboxDir	string		`json:"dropboxDir"`
	Podcasts	[]*Podcast	`json:"podcasts,omitempty"`
}

func (c *Config) getFileName() string {
	absPath, _ := filepath.Abs("./");
	return absPath + "/config_ps.json";
}

func (c *Config) Load() {

	file, err := ioutil.ReadFile(c.getFileName())

	if err != nil {
		log.Fatal("Config reading error from " + c.getFileName() + " ", err);
		panic("config reading error");
	}

	err = json.Unmarshal(file, c);

	if err != nil || c == nil {
		log.Fatal("Config decoding error ", err);
		panic("config decoding error");
	}

	if c.DropboxDir == "" || len(c.Podcasts) == 0 {
		log.Fatal("we need dropbox dir and podcasts list to go")
		panic("config content error")
	}

	log.Println("config read");
}

func (c *Config) Save() {

	out, err := json.MarshalIndent(c, "", "	")

	err = ioutil.WriteFile(c.getFileName(), out,0755)

	if err != nil {
		log.Fatal("Config write error ", err);
		panic("config write error");
	}

	log.Println("Config saved");
}