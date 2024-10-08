package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	MESSAGES_JSON_PATH string = "/messages.json"
	CHANNEL_JSON_PATH  string = "/channel.json"
	MESSAGES_PATH      string = "messages"
	DUMP_FILE          string = "messages.txt"
	FORMATINT_BASE     int    = 10
	// TIMESTAMP_LAYOUT   string = "2006-01-02 15:04:05"
)

type MessagesJSON []struct {
	ID        int64  `json:"ID,omitempty"`
	Timestamp string `json:"Timestamp,omitempty"`
}

type ChannelJSON struct {
	ID string `json:"id,omitempty"`
}

type FlagValues struct {
	Path        string
	Exclude     bool
	Include     bool
	AllMessages bool
	ByYear      string
	ByChannels  string
}

var (
	channelJson  ChannelJSON
	messagesJson MessagesJSON
	flagValues   FlagValues
	channelsList map[string]struct{} = make(map[string]struct{})
	// wg       sync.WaitGroup
)

func parseFlags() {
	flag.StringVar(&flagValues.Path, "path", MESSAGES_PATH, "path to 'messages' directory")
	flag.BoolVar(&flagValues.AllMessages, "all", true, "dump every message")
	flag.StringVar(&flagValues.ByYear, "year", "", "dump every message from a specified year")
	flag.StringVar(&flagValues.ByChannels, "channels", "", "channels to exclude or include from the dump. [comma,separate,the,input]")
	flag.BoolVar(&flagValues.Exclude, "exclude", false, "exclude specified channels from the dump")
	flag.BoolVar(&flagValues.Include, "include", false, "only include specified channels from the dump")
	flag.Parse()
}

func pathWalk(dumpFile *os.File) error {
	defer dumpFile.Close()

	err := filepath.Walk(flagValues.Path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() != flagValues.Path && info.IsDir() {
			readDirs(path, dumpFile)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func readDirs(path string, dumpFile *os.File) {
	cFile, err := os.ReadFile(path + CHANNEL_JSON_PATH)
	if err != nil {
		log.Fatalf("error reading channel file. ERROR: %v\n", err)
	}
	if err := json.Unmarshal(cFile, &channelJson); err != nil {
		log.Fatalf("error unmarshalling channel json file. ERROR: %v\n", err)
	}

	mFile, err := os.ReadFile(path + MESSAGES_JSON_PATH)
	if err != nil {
		log.Fatalf("error reading messages file. ERROR: %v\n", err)
	}
	if err := json.Unmarshal(mFile, &messagesJson); err != nil {
		log.Fatalf("error unmarshalling messages json file. ERROR: %v\n", err)
	}

	messagesJsonLength := len(messagesJson)
	if messagesJsonLength == 0 {
		return
	}

	if flagValues.ByYear != "" {
		dumpByYear(dumpFile)
	} else if flagValues.ByChannels != "" {
		dumpByChannels(messagesJsonLength, dumpFile)
	} else {
		dumpAllMessages(messagesJsonLength, dumpFile)
	}
}

func dumpAllMessages(messagesJsonLength int, dumpFile *os.File) {
	dumpToFile(dumpFile, (channelJson.ID + ":\n"))
	for i, m := range messagesJson {
		if i+1 == messagesJsonLength {
			dumpToFile(dumpFile, (strconv.FormatInt(m.ID, FORMATINT_BASE) + "\n\n"))
		} else {
			dumpToFile(dumpFile, (strconv.FormatInt(m.ID, FORMATINT_BASE) + ", "))
		}
	}
}

func dumpByYear(dumpFile *os.File) {
	channelIDDumped := false
	for _, m := range messagesJson {
		// parsedTimestamp, err := time.Parse(TIMESTAMP_LAYOUT, m.Timestamp)
		// if err != nil {
		// 	log.Fatalf("error parsing timestamp. ERROR: %v\n", err)
		// }
		// fmt.Println(parsedTimestamp.Year(), i)

		timestampSplit := strings.SplitN(m.Timestamp, "-", 2)
		timestampYear := timestampSplit[0]

		if timestampYear != flagValues.ByYear {
			continue
		}

		if !channelIDDumped {
			dumpToFile(dumpFile, (channelJson.ID + ":\n"))
			channelIDDumped = true
		}

		dumpToFile(dumpFile, (strconv.FormatInt(m.ID, FORMATINT_BASE) + ", "))
	}
	if channelIDDumped {
		dumpToFile(dumpFile, "\n\n")
	}
}

func dumpByChannels(messagesJsonLength int, dumpFile *os.File) {
	_, channelIDExists := channelsList[channelJson.ID]
	if flagValues.Exclude && channelIDExists {
		return
	}
	if flagValues.Include && !channelIDExists {
		return
	}

	dumpToFile(dumpFile, (channelJson.ID + ":\n"))
	for i, m := range messagesJson {
		if i+1 == messagesJsonLength {
			dumpToFile(dumpFile, (strconv.FormatInt(m.ID, FORMATINT_BASE) + "\n\n"))
		} else {
			dumpToFile(dumpFile, (strconv.FormatInt(m.ID, FORMATINT_BASE) + ", "))
		}
	}
}

func dumpToFile(dumpFile *os.File, content string) {
	_, err := dumpFile.WriteString(content)
	if err != nil {
		log.Fatalf("error dumping into file. ERROR: %v\n", err)
	}
}

func createFile() (*os.File, error) {
	file, err := os.Create(DUMP_FILE)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func fillChannelsList() {
	if flagValues.ByChannels != "" {
		excludedChannels := strings.Split(flagValues.ByChannels, ",")
		for _, e := range excludedChannels {
			channelsList[e] = struct{}{}
		}
	}
}

func main() {
	parseFlags()
	fillChannelsList()

	dumpFile, err := createFile()
	if err != nil {
		log.Fatalf("error creating dump file. ERROR: %v\n", err)
	}

	if err := pathWalk(dumpFile); err != nil {
		log.Fatalf("error walking path. ERROR: %v\n", err)
	}

	fmt.Printf("dumped to '%s'\n", DUMP_FILE)
}
