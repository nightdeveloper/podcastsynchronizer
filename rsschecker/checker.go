package rsschecker;

import (
	"log"
	"github.com/nightdeveloper/podcastsynchronizer/settings"
	"github.com/nightdeveloper/podcastsynchronizer/structs"
	"io/ioutil"
	"encoding/xml"
	"net/url"
	"time"
	"strings"
	"errors"
	"net/http"
	"os"
	"io"
)

var DEFAULT_SEARCH_DEPTH = 5

type Checker struct {
	config	*settings.Config
	chatChannel chan string
}

func NewChecker(config *settings.Config) (c *Checker) {
	ch := new(Checker);
	ch.config = config;
	return ch;
}

func (c *Checker) SetChatChannel(cc chan string) {
	c.chatChannel = cc;
}

func (c *Checker) downloadPodcast(p *settings.Podcast, i *structs.ItemStruct) error {

	if !strings.Contains(i.Enclosure.Type, "audio") && i.Enclosure.Type != "" {

		log.Println("type " + i.Enclosure.Type + " is not mp3");
		return errors.New("enclosure type is not mp3 (" + i.Enclosure.Type + ")");
	}

	log.Println("found file " + i.Enclosure.URL);
	if i.Enclosure.URL == "" {
		log.Println("empty file :(");
		return nil;
	}

	var fileUrl, _ = url.QueryUnescape(i.Enclosure.URL);

	var parts = strings.Split( fileUrl, "/");

	if len(parts) == 0 {
		log.Println("url parts are zero length - " + i.Enclosure.URL)
		return errors.New("url error");
	}

	var pf = parts[len(parts) - 1];
	if pf == "" {
		pf = parts[len(parts) - 2];
	}

	var destinationFilePath = c.config.DropboxDir + "/" + pf;

	parts = strings.Split(destinationFilePath, "?")
	destinationFilePath = parts[0];

	log.Println( i.Title, ": downloading file " + i.Enclosure.URL + " to " + destinationFilePath);


	out, err := os.Create(destinationFilePath)
	defer out.Close()

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " file write error: " + err.Error())
		return errors.New("write error: " + err.Error());
	}


	client := http.Client{ Timeout: time.Duration(10 * time.Minute) }
	resp, err := client.Get(i.Enclosure.URL)

	defer resp.Body.Close()

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " file get error: " + err.Error())
		return errors.New("get error: " + err.Error());
	}

	log.Println("saving file to " + destinationFilePath);

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " file download error: " + err.Error())
		return errors.New("download error: " + err.Error());
	} else {
		if (c.chatChannel != nil) {
			c.chatChannel <- "New podcast: " + i.Title;
		}
	}

	return nil;
}

func (c *Checker) checkPodcast(p *settings.Podcast) {

	p.LastChecked = time.Now()

	client := http.Client{ Timeout: time.Duration(10 * time.Minute) }
	response, err := client.Get(p.Url)

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " download error: " + err.Error())
		return;
	}

	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " read error")
		return
	}

	var rss structs.RSSStruct
	err = xml.Unmarshal(bytes, &rss)

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " unmarshal error: ", err.Error())
		return
	}

	c.ProcessEntry(rss, p)
	c.ProcessChannel(rss, p)

	p.Status = "Checked successfully"
	p.Name = rss.Channel.Title
	log.Println("check finished: " + rss.Channel.Title + " (last updated: " + p.LastUpdated.String() + ")");
	c.config.Save()
}

func (c *Checker) ProcessEntry(rss structs.RSSStruct, p *settings.Podcast) {
	if len(rss.Entry) == 0 {
		log.Println(p.Url + " no items in entry")
		return
	}

	var depth = DEFAULT_SEARCH_DEPTH

	var firstGuid = ""
	if p.MaxDepth > 0 {
		depth = p.MaxDepth
	}

	log.Println("checking with depth", depth)

	for  _, i := range rss.Entry {

		log.Println("entry item hit")
		var guid = i.VideoId

		if firstGuid == "" {
			firstGuid = guid
		}

		if guid != "" && guid == p.LastGuid {
			log.Println("last guid hit")
			break
		}

		if depth > 0 {
			log.Println("title " + i.Title)
			c.chatChannel <- "New entry: " + i.Title
			depth--
		}
	}

	if firstGuid != p.LastGuid {
		p.LastGuid = firstGuid
		p.LastUpdated = time.Now()
	}
}

func (c *Checker) ProcessChannel(rss structs.RSSStruct, p *settings.Podcast) {
	if len(rss.Channel.Item) == 0 {
		log.Println(p.Url + " no items in channel")
		return
	}

	var depth = DEFAULT_SEARCH_DEPTH

	var firstGuid = "";
	if p.MaxDepth > 0 {
		depth = p.MaxDepth
	}

	log.Println("checking with depth", depth)

	var wasErrors = false;
	for  _, i := range rss.Channel.Item {

		log.Println("channel item hit")

		var guid = i.Guid

		if guid == "" {
			guid = i.PubDate + " " + i.Title
		}

		if firstGuid == "" {
			firstGuid = guid
		}

		if guid != "" && guid == p.LastGuid {
			log.Println("last guid hit")
			break
		}

		if depth > 0 {
			if i.Enclosure.URL != "" {

				log.Println("url " + i.Enclosure.URL)

				var isFilteredOk = true

				if p.Filters != nil {
					log.Println("filtering [", i.Title, "]...")
					var isMatch = false
					for _, f := range p.Filters {
						if strings.Contains(i.Title, f.Title) {
							isMatch = true
						}
					}
					isFilteredOk = isMatch

					log.Println("filtered", isFilteredOk)
				}

				if isFilteredOk {
					err := c.downloadPodcast(p, &i)
					if err == nil {
						p.Status = "downloaded ok"
						log.Println("downloaded ok")
					} else {
						p.Status = "download error: " + err.Error()
						wasErrors = true
					}
				}
				depth--
			} else {
				log.Println("rss item with no url")
				c.chatChannel <- "New rss: " + i.Title
			}
		}

		if wasErrors {
			break
		}
	}

	if firstGuid != p.LastGuid && !wasErrors {
		p.LastGuid = firstGuid
		p.LastUpdated = time.Now()
	}
}

func (c *Checker) StartLoop() {
	log.Println("checker loop started")

	for{
		c.config.Load()

		log.Printf("tick (%d podcasts)", len(c.config.Podcasts));

		for  _, p := range c.config.Podcasts {
			log.Println("want to check podcast " + p.Url)

			c.checkPodcast(p);
		}

		c.config.Save()

		log.Println("tick finished")

		startTime := time.Now();
		for time.Since(startTime).Hours() < 2 {
			time.Sleep(time.Duration(1) * time.Minute);
		}
		time.Sleep(time.Duration(1) * time.Minute); // additional sleep for wake up time network interface
	}
}
