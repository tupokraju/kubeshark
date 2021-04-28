package inserter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/antoniodipinto/ikisocket"
	"github.com/google/martian/har"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"mizuserver/pkg/database"
	"mizuserver/pkg/models"
	"mizuserver/pkg/utils"
	"net/url"
	"os"
	"path"
	"sort"
	"time"
)

func StartReadingFiles(workingDir string) {
	err := os.MkdirAll(workingDir, 777)
	utils.CheckErr(err)

	for true {
		dir, _ := os.Open(workingDir)
		dirFiles, _ := dir.Readdir(-1)
		sort.Sort(utils.ByModTime(dirFiles))

		if len(dirFiles) == 0{
			fmt.Printf("Waiting for new files\n")
			time.Sleep(3 * time.Second)
			continue
		}
		fileInfo := dirFiles[0]
		inputFilePath := path.Join(workingDir, fileInfo.Name())
		file, err := os.Open(inputFilePath)
		utils.CheckErr(err)

		var inputHar har.HAR
		decErr := json.NewDecoder(bufio.NewReader(file)).Decode(&inputHar)
		utils.CheckErr(decErr)

		for _, entry := range inputHar.Log.Entries {
			SaveHarToDb(*entry, "")
		}
		rmErr := os.Remove(inputFilePath)
		utils.CheckErr(rmErr)
	}
}

func SaveHarToDb(entry har.Entry, source string) {
	entryBytes, _ := json.Marshal(entry)
	serviceName, urlPath := getServiceNameFromUrl(entry.Request.URL)
	entryId := primitive.NewObjectID().Hex()
	mizuEntry := models.MizuEntry{
		EntryId:   entryId,
		Entry:     string(entryBytes), // simple way to store it and not convert to bytes
		Service:   serviceName,
		Url:       entry.Request.URL,
		Path:      urlPath,
		Method:    entry.Request.Method,
		Status:    entry.Response.Status,
		Source:    source,
		Timestamp: entry.StartedDateTime.Unix(),
	}
	database.GetEntriesTable().Create(&mizuEntry)

	baseEntry := &models.BaseEntryDetails{
		Id:         entryId,
		Url:        entry.Request.URL,
		Service:    serviceName,
		Path:       urlPath,
		StatusCode: entry.Response.Status,
		Method:     entry.Request.Method,
		Timestamp:  entry.StartedDateTime.Unix(),
	}
	baseEntryBytes, _ := json.Marshal(&baseEntry)
	ikisocket.Broadcast(baseEntryBytes)

}

func getServiceNameFromUrl(inputUrl string) (string, string) {
	parsed, err := url.Parse(inputUrl)
	utils.CheckErr(err)
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), parsed.Path
}
