package main

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (
	config struct {
		CallBrowser bool

		Debug   bool
		Log     bool
		LogPath string

		EventDefinition string
		DatabasePath    string
		DataPath        string

		Port       int
		PublicPath string

		AppVersion string
	}

	// definition.json
	RawCategory struct {
		ID   int      `json:"id"`
		Desc []string `json:"desc"`
	}
	RawOption struct {
		ID        int      `json:"id"`
		EventID   int      `json:"event_id"`
		GroupID   int      `json:"group_id"`
		GroupType string   `json:"group_type"`
		Desc      []string `json:"desc"`
	}
	RawDefinition struct {
		Category []RawCategory `json:"category"`
		Option   []RawOption   `json:"option"`
	}

	// response definition struct of events
	definition struct {
		EventID      int           `json:"event_id"`
		Desc         []string      `json:"desc"`
		OptionGroups []OptionGroup `json:"option_groups"`
	}
	OptionGroup struct {
		GroupID   int      `json:"group_id"`
		GroupType string   `json:"group_type"`
		Options   []Option `json:"options"`
	}
	Option struct {
		OptionID int      `json:"option_id"`
		Desc     []string `json:"desc"`
	}

	// file
	File struct {
		Name    string    `json:"name"`
		ExtName string    `json:"ext_name"`
		Date    time.Time `json:"date"`
	}

	Folder struct {
		Name      string    `json:"name"`
		Path      string    `json:"path"`
		Date      time.Time `json:"date"`
		SubFile   []File    `json:"sub_file"`
		SubFolder []Folder  `json:"sub_folder"`
	}
)

func writeLog(level string, content interface{}) {
	level = strings.ToUpper(level)
	if C.Debug || level != "DEBUG" {
		_, _ = fmt.Fprintln(gin.DefaultWriter,
			fmt.Sprintf("[GIN:%s] %v |%s \n", level, time.Now().Format("2006/01/02 - 15:04:05"), content))
	}

	if level == "FATAL" {
		os.Exit(1)
	}
}

func insertOption(groups *[]OptionGroup, option RawOption) {
	for i, v := range *groups {
		if v.GroupID == option.GroupID {
			(*groups)[i].Options = append(v.Options, Option{option.ID, option.Desc})
			return // existing group was found and inserted successfully
		}
	}
	// group not exist
	*groups = append(*groups, OptionGroup{
		option.GroupID,
		option.GroupType,
		[]Option{{option.ID, option.Desc}},
	})
}

func parseEventDefinition(rd *RawDefinition) (definitions []definition) {
	// Option allows out-of-order thus dual loops is required
	for _, c := range rd.Category {
		optionGroups := &[]OptionGroup{}
		for _, o := range rd.Option {
			if o.EventID == c.ID {
				insertOption(optionGroups, o)
			}
		}
		definitions = append(definitions, definition{c.ID, c.Desc, *optionGroups})
	}
	return
}

func loadDefinition() []definition {
	// read definition file
	bytes, err := ioutil.ReadFile(C.EventDefinition)
	if err != nil {
		fmt.Println("Error when read definition file: ", err.Error())
		os.Exit(1)
	}
	// parse raw definition
	rd := &RawDefinition{}
	err = json.Unmarshal(bytes, rd)
	if err != nil {
		fmt.Println("Error when unmarshal definition: ", err.Error())
		os.Exit(1)
	}
	return parseEventDefinition(rd)
}

func readConf() *config {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	err := viper.ReadInConfig()
	if err != nil { // Handle errors reading the config file
		fmt.Println(fmt.Errorf("Fatal error config file: %s \n", err))
		os.Exit(1)
	}
	c := &config{
		viper.GetBool("main.call_borwser"),
		viper.GetBool("dev.debug"),
		viper.GetBool("dev.log"),
		viper.GetString("dev.log_path"),
		viper.GetString("data.definition_path"),
		viper.GetString("data.database_path"),
		viper.GetString("data.data_path"),
		viper.GetInt("server.port"),
		viper.GetString("server.public_path"),
		_VERSION_,
	}
	// Set the database in memory when debug is on
	if c.Debug {
		c.DatabasePath = ":memory:"
		fmt.Println("Warning: you are in Debug mode and no data will be saved.")
	}
	if !c.Log {
		c.LogPath = "unavailable"
	}
	fmt.Println("Configurations read successfully.")
	data := [][]string{
		{"Version", c.AppVersion},
		{"Debug", strconv.FormatBool(c.Debug)},
		{"CallBrowser", strconv.FormatBool(c.CallBrowser)},
		{"EventDefinitionPath", c.EventDefinition},
		{"DatabasePath", c.DatabasePath},
		{"PublicPath", c.PublicPath},
		{"DataPath", c.DataPath},
		{"ServerPort", strconv.Itoa(c.Port)},
		{"Log", strconv.FormatBool(c.Log)},
		{"LogPath", c.LogPath},
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Option", "Value"})
	//table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold},
	//	tablewriter.Colors{tablewriter.Bold})
	//table.SetColumnColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
	//	tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()
	return c
}

func connectDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", C.DatabasePath)
	if err != nil {
		writeLog("FATAL", err)
	}
	if err := db.DB().Ping(); err != nil {
		writeLog("FATAL", err)
	}
	fmt.Println("Database connected.")
	db.AutoMigrate(&Event{}, &Video{}, &Trajectory{}, &Rating{})
	return db
}

func callUserInterface(callBrowser bool) {
	fmt.Printf("\nPlease visit \"http://localhost:%v\" in your browser on the local computer.\n", C.Port)
	fmt.Printf("Or through an intranet address for other devices in the LAN:\n"+
		"\t1. Run \"ipconfig /all\" in your terminal and view IP addresses.\n"+
		"\t2. Find the local address of the intranet where the target device is located.\n"+
		"\t3. Visit \"http://[This PC's IP address]:%v\" on the target device.\n", C.Port)
	ifaces, _ := net.Interfaces()
	fmt.Printf("\t(Maybe you can try: ")
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			ipnet, _ := addr.(*net.IPNet)
			if ipnet.IP.IsGlobalUnicast() {
				fmt.Printf(" \"http://%v:%v\"", ipnet.IP.To4(), C.Port)
			}
		}
	}
	fmt.Printf(")\n")

	if callBrowser {
		cmd := fmt.Sprintf("/c start http://localhost:%v", C.Port)
		err := exec.Command("cmd", cmd).Start()
		if err != nil {
			writeLog("WARN", err)
			return
		}
	}
}

func TraverseDirectoriesRecursively(folder *Folder, rootDir string) (err error) {
	var files []os.FileInfo
	if rootDir == "" {
		files, err = ioutil.ReadDir(folder.Path)
	} else {
		files, err = ioutil.ReadDir(rootDir)
		folder.Path = rootDir
	}
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	for _, f := range files {
		currentFilePath := fmt.Sprintf("%v/%v", folder.Path, f.Name())
		if f.IsDir() { // traverse directory recursively
			var date time.Time
			date, err = time.Parse("060102", f.Name())
			if err != nil {
				date = time.Time{}
			}
			newSubFolder := Folder{
				f.Name(),
				currentFilePath,
				date,
				[]File{},
				[]Folder{},
			}
			err = TraverseDirectoriesRecursively(&newSubFolder, "")
			if err != nil {
				writeLog("FATAL", err)
			}
			folder.SubFolder = append(folder.SubFolder, newSubFolder)
		} else {
			file := File{
				f.Name(),
				strings.ToLower(filepath.Ext(f.Name())),
				folder.Date,
			}
			folder.SubFile = append(folder.SubFile, file)
			StoreFileToDatabase(file, currentFilePath)
		}
	}
	return
}

func StoreFileToDatabase(file File, path string) {
	if file.ExtName == ".mp4" {
		videoSplitName := strings.Split(file.Name, "_")
		var videoBeginTime time.Time
		var videoEndTime time.Time
		var videoType string
		var err error
		if len(videoSplitName) == 3 {
			formatDate := file.Date.Format("20060102")
			videoType = videoSplitName[3]
			videoBeginTime, err = time.Parse("20060102T150405Z07:00",
				fmt.Sprintf("%sT%s+08:00", formatDate, videoSplitName[1]))
			if err != nil {
				writeLog("WARN", err)
				videoBeginTime = file.Date
			}

			videoEndTime, err = time.Parse("20060102T150405Z07:00",
				fmt.Sprintf("%sT%s+08:00", formatDate, videoSplitName[2]))
			if err != nil {
				writeLog("WARN", err)
				videoEndTime = file.Date
			}
		} else {
			videoBeginTime = time.Time{}
			videoEndTime = time.Time{}
			videoType = "Unknown"
		}
		if err := insertVideoIfNotExist(&Video{
			FileName:         file.Name,
			Path:             path,
			BeginTime:        videoBeginTime,
			EndTime:          videoEndTime,
			Type:             videoType,
			VideoGPSTimeDiff: 0,
		}); err != nil {
			writeLog("FATAL", err)
		}
	}
	if file.ExtName == ".vbo" {
		if err := insertTrajectoryIfNotExist(&Trajectory{
			FileName:  file.Name,
			Path:      path,
			BeginTime: file.Date,
			EndTime:   file.Date,
		}); err != nil {
			writeLog("FATAL", err)
		}
	}
}

func InitDataStorageFiles() (root *Folder) {
	root = &Folder{"Public", "", time.Time{}, []File{}, []Folder{}}
	_ = TraverseDirectoriesRecursively(root, C.DataPath)
	return
}