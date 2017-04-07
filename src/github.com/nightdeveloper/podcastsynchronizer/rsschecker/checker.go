package rsschecker;

import (
	"log"
	"github.com/nightdeveloper/podcastsynchronizer/settings"
	"github.com/nightdeveloper/podcastsynchronizer/structs"
	"io/ioutil"
	"encoding/xml"
	"time"
	"strings"
	"errors"
	"net/http"
	"os"
	"io"
)

type Checker struct {
	config	*settings.Config
}

func NewChecker(config *settings.Config) (c *Checker) {
	ch := new(Checker);
	ch.config = config;
	return ch;
}

func (c *Checker) downloadPodcast(p *settings.Podcast, i *structs.ItemStruct) error {

	if !strings.Contains(i.Enclosure.Type, "audio") && i.Enclosure.Type != "" {
		log.Println("type " + i.Enclosure.Type + " is not mp3");
		return errors.New("enclosure type is not mp3 (" + i.Enclosure.Type + ")");
	}

	log.Println("found file " + i.Enclosure.URL);
	if (i.Enclosure.URL == "") {
		log.Println("empty file :(");
		return nil;
	}

	var parts = strings.Split( i.Enclosure.URL, "/");

	if len(parts) == 0 {
		log.Println("url parts are zero length - " + i.Enclosure.URL)
		return errors.New("url error");
	}

	var pf = parts[len(parts) - 1];
	if (pf == "") {
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
	bytes, _ := ioutil.ReadAll(response.Body)
	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " read error")
		return
	}

	var rss structs.RSSStruct;

	err = xml.Unmarshal(bytes, &rss)

	if err != nil {
		p.Status = err.Error()
		log.Println(p.Url + " unmarshal error: ", err.Error())
		return
	}

	if len(rss.Channel.Item) == 0 {
		if (err != nil ) {
			p.Status = err.Error()
		} else { p.Status = "no error supplied"; }
		log.Println(p.Url + " no items in channel")
		return;
	}

	var firstGuid = "";
	var depth = 5
	var wasErrors = false;
	for  _, i := range rss.Channel.Item {

		if firstGuid == "" {
			firstGuid = i.Guid;
		}

		if i.Guid != "" && i.Guid == p.LastGuid {
			break;
		}

		if depth > 0 {
			if (i.Enclosure.URL != "") {
				log.Println("url " + i.Enclosure.URL);
				err := c.downloadPodcast(p, &i);
				if err == nil {
					p.Status = "downloaded ok"
					log.Println("downloaded ok")
				} else {
					p.Status = "download error: " + err.Error()
					wasErrors = true
				}
				depth--
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

	p.Name = rss.Channel.Title

	log.Println("check finished: " + rss.Channel.Title)

	c.config.Save()
}

func (c *Checker) StartLoop() {
	log.Println("checker loop started")

	//for{
		log.Println("tick (", len(c.config.Podcasts), " podcasts)");

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
	//}
}
