package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// Ruby's simplecov coveraage.json struct types list
type Rspec struct {
	Rspec RubyCoverage `json:"RSpec"`
}
type RubyCoverage struct {
	Coverage  LineBranchMap `json:"coverage"`
	Timestamp int64         `json:"timestamp"`
}
type LineBranchMap map[string]RubyLineBranch

type RubyLineBranch struct {
	Lines        []int64            `json:"lines"`
	RubyBranches RubyBranchStartMap `json:"branches"`
}
type RubyBranchStartMap map[string]RubyBranchMap
type RubyBranchMap map[string]int

// struct to create generic branch coerage report for sonarqube
type coverage struct {
	Version string         `xml:"version,attr"`
	File    []FileCoverage `xml:"file"`
}
type FileCoverage struct {
	Path        string        `xml:"path,attr"`
	LineToCover []LineToCover `xml:"lineToCover"`
}
type LineToCover struct {
	LineNumber      string `xml:"lineNumber,attr"`
	Covered         string `xml:"covered,attr"`
	BranchesToCover string `xml:"branchesToCover,attr"`
	CoveredBranches string `xml:"coveredBranches,attr"`
}

// struct for creation of all-branch-cover.json file to find full branch coverage rate
type BranchCoverageData struct {
	Start      string `json:"Start"`
	Code       string `json:"Code"`
	TrueCount  int    `json:"TrueCount"`
	FalseCount int    `json:"FalseCount"`
}
type BranchCoverageDataCoverage struct {
	Filename        string                `json:"filename"`
	Covered         int                   `json:"covered"`
	BranchesToCover int64                 `json:"total"`
	CoverageRate    float64               `json:"coverageRate"`
	CoverageData    []*BranchCoverageData `json:"coverageData"`
}
type BranchCoverage struct {
	CoverageRate       float64                      `json:"coverageRate"`
	BranchesToCover    int64                        `json:"totalBranch"`
	TotalCoveredBranch int                          `json:"totalCoveredBranch"`
	AllData            []BranchCoverageDataCoverage `json:"allData"`
}

const EMPTY_CHECK = 0

func main() {

	// Get command line arguments
	packageName := flag.String("packageName", "", "")
	isUpdateJson := flag.Bool("isUpdateJson", false, "")
	flag.Parse()
	if !fileExists(*packageName+"/branch-cover.json") && *packageName != "c1-ruby" {
		return
	}
	var currentContent []byte
	var err error
	if *packageName == "c1-ruby" {
		currentContent, err = ioutil.ReadFile("coverage/.resultset.json")
	} else {
		currentContent, err = ioutil.ReadFile(*packageName + "/branch-cover.json")
	}
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	if len(currentContent) == EMPTY_CHECK {
		log.Printf("Empty content")
		var filelemList []FileCoverage
		if *packageName == "./test-results" {
			createXmlFile(filelemList, "test-results")
		} else if *packageName == "c1-ruby" {
			createXmlFile(filelemList, "coverage")
		}
		return
	}

	var payloadCurrent []*BranchCoverageData
	var newpayloadCurrent []*BranchCoverageData
	var allDataCover []BranchCoverageDataCoverage

	if !(*isUpdateJson) {
		var filenamecover string
		var totalBranchInFile int64
		var coveredBranchInFile int
		var totalBranch int64
		var coveredBranch int
		if *packageName == "./test-results" {

			err = json.NewDecoder(strings.NewReader(string(currentContent[:]))).Decode(&payloadCurrent)
			if err != nil {
				log.Fatal("Error while reading .resultset.json file\n", err)
			}
			if err != nil {
				log.Fatal("Error during Unmarshal() of currentContent: ", err)
			}
			// To generate xml file atlast
			var fileElementList []FileCoverage

			for i := 0; i < len(payloadCurrent); i++ {
				lineNumber := payloadCurrent[i].Start
				fileName := strings.Split(lineNumber, ":")[0]

				if (len(filenamecover) > 0 && (fileName != filenamecover)) || i == len(payloadCurrent)-1 {
					var branchCoverageDataCoverage BranchCoverageDataCoverage
					branchCoverageDataCoverage.Filename = filenamecover
					branchCoverageDataCoverage.Covered = coveredBranchInFile
					branchCoverageDataCoverage.BranchesToCover = totalBranchInFile
					branchCoverageDataCoverage.CoverageRate = getScore(totalBranchInFile, coveredBranchInFile)
					branchCoverageDataCoverage.CoverageData = append(branchCoverageDataCoverage.CoverageData, newpayloadCurrent...)
					allDataCover = append(allDataCover, branchCoverageDataCoverage)
					xmlLineToCoverElemList := createDataForXmlfile(branchCoverageDataCoverage)
					fileElement := FileCoverage{
						Path:        filenamecover,
						LineToCover: xmlLineToCoverElemList,
					}
					fileElementList = append(fileElementList, fileElement)
					totalBranch = totalBranch + totalBranchInFile
					coveredBranch = coveredBranch + coveredBranchInFile
					totalBranchInFile = EMPTY_CHECK
					coveredBranchInFile = EMPTY_CHECK
					newpayloadCurrent = nil
					filenamecover = fileName
				}

				newpayloadCurrent = append(newpayloadCurrent, payloadCurrent[i])
				totalBranchInFile = totalBranchInFile + 2
				if payloadCurrent[i].TrueCount > EMPTY_CHECK {
					coveredBranchInFile++
				}
				if payloadCurrent[i].FalseCount > EMPTY_CHECK {
					coveredBranchInFile++
				}
				if i == EMPTY_CHECK {
					filenamecover = fileName
				}
			}

			createXmlFile(fileElementList, "test-results")

			/* To print coverage rate from gobco output*/
			var branchCoverage BranchCoverage
			branchCoverage.AllData = allDataCover
			branchCoverage.CoverageRate = getScore(totalBranch, coveredBranch)
			branchCoverage.BranchesToCover = totalBranch
			branchCoverage.TotalCoveredBranch = coveredBranch
			jsonfileall, _ := json.Marshal(branchCoverage)
			_ = ioutil.WriteFile("test-results/all-branch-cover.json", jsonfileall, 0644)
			// log.Println("Total Branches: ", totalBranch)
			// log.Println("Total Covered: ", coveredBranch)
			// log.Println("Coverage= ", getScore(totalBranch, coveredBranch))

		} else if *packageName == "c1-ruby" {
			log.Println("C1 Coverage report for ruby")
			var rspec Rspec
			err = json.NewDecoder(strings.NewReader(string(currentContent[:]))).Decode(&rspec)
			if err != nil {
				log.Fatal("Error while reading .resultset.json file\n", err)
			}

			ParseJsonForRubyAndPrepareXmlData(rspec)
		} else {

			err = json.NewDecoder(strings.NewReader(string(currentContent[:]))).Decode(&payloadCurrent)
			if err != nil {
				log.Fatal("Error during Unmarshal() of currentContent for packagename update: ", err)
			}
			// To update package name of the files in gobco output json file(it only has filename without package)
			for i := 0; i < len(payloadCurrent); i++ {
				lineNumber := payloadCurrent[i].Start
				fileName := strings.Split(lineNumber, ":")[0]
				lineNumberStartSplitted, _ := strconv.Atoi(strings.Split(lineNumber, ":")[1])
				ColNumberStartSplitted, _ := strconv.Atoi(strings.Split(lineNumber, ":")[2])

				payloadCurrent[i].Start = *packageName + "/" + fileName + ":" + strconv.Itoa(lineNumberStartSplitted) + ":" + strconv.Itoa(ColNumberStartSplitted)
			}
			jsonfile, _ := json.Marshal(payloadCurrent)
			_ = ioutil.WriteFile(*packageName+"/branch-cover.json", jsonfile, 0644)
		}
	} else if *isUpdateJson {
		// Copy branch-cover.json file from each package and write it in test-results folder
		contentAll, err := ioutil.ReadFile(*packageName + "/branch-cover.json")

		if err != nil {
			log.Fatal("Error when opening file: ", err)
		}

		if len(contentAll) > 0 {
			var payloadAll []*BranchCoverageData
			err = json.NewDecoder(strings.NewReader(string(contentAll[:]))).Decode(&payloadAll)
			if err != nil {
				log.Fatal("Error during Unmarshal() of contentAll: ", err)
			}
			payloadAll = append(payloadAll, payloadCurrent...)
			jsonfile, _ := json.Marshal(payloadAll)
			_ = ioutil.WriteFile("test-results/branch-cover.json", jsonfile, 0644)

		} else {
			jsonfile, _ := json.Marshal(payloadCurrent)
			_ = ioutil.WriteFile("test-results/branch-cover.json", jsonfile, 0644)
		}
	}

}

//Find branch coveage rate
func getScore(total int64, covered int) float64 {
	return math.Floor(float64(covered) / float64(total) * 100)
}

// Create lineToCover element list for a file
func createDataForXmlfile(branchCoverageDataCoverage BranchCoverageDataCoverage) []LineToCover {
	var branchCoverDataForaFile = branchCoverageDataCoverage.CoverageData
	var lineToCoverElemList []LineToCover
	for j := 0; j < len(branchCoverDataForaFile); j++ {
		var coveredBranchXml = 0
		var isCovered = true
		if branchCoverageDataCoverage.CoverageData[j].TrueCount > EMPTY_CHECK {
			coveredBranchXml++
		}
		if branchCoverageDataCoverage.CoverageData[j].FalseCount > EMPTY_CHECK {
			coveredBranchXml++
		}
		if coveredBranchXml < 2 {
			isCovered = false
		}
		codeStart := branchCoverageDataCoverage.CoverageData[j].Start
		lineNumber := strings.Split(codeStart, ":")[1]
		lineToCoverElement := LineToCover{
			LineNumber:      lineNumber,
			Covered:         fmt.Sprint(isCovered),
			BranchesToCover: "2",
			CoveredBranches: fmt.Sprint(coveredBranchXml),
		}

		lineToCoverElemList = append(lineToCoverElemList, lineToCoverElement)
	}
	return lineToCoverElemList
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func ParseJsonForRubyAndPrepareXmlData(rspec Rspec) {

	keys := make([]string, 0, len(rspec.Rspec.Coverage))
	for k := range rspec.Rspec.Coverage {
		keys = append(keys, k)
	}

	var fileElementList []FileCoverage
	// To generate xml file atlast
	for i := 0; i < len(keys); i++ {
		var lineToCoverElemList []LineToCover
		fileName := keys[i]

		rubyBranchMap := rspec.Rspec.Coverage[fileName].RubyBranches
		branchKeys := make([]string, 0, len(rubyBranchMap))

		for c1 := range rubyBranchMap {
			branchKeys = append(branchKeys, c1)
		}

		for k := 0; k < len(branchKeys); k++ {
			outetKey := branchKeys[k]
			newKey := strings.Replace(strings.Replace(outetKey, "[", "", -1), "]", "", -1)
			branchKeyArray := strings.Split(newKey, ",")

			branchMap := rubyBranchMap[outetKey]
			ifelseKeys := make([]string, 0, len(branchMap))
			for k1 := range branchMap {
				ifelseKeys = append(ifelseKeys, k1)
			}
			notCovered := 0
			for branchCount := 0; branchCount < len(ifelseKeys); branchCount++ {
				if branchMap[ifelseKeys[branchCount]] == EMPTY_CHECK {
					notCovered++
				}
			}

			lineToCoverElement := prepareLineElement(branchKeyArray[2], strconv.Itoa(len(ifelseKeys)), strconv.Itoa(len(ifelseKeys)-notCovered),
				(len(ifelseKeys)-notCovered) == len(ifelseKeys))

			lineToCoverElemList = append(lineToCoverElemList, lineToCoverElement)
		}
		if len(lineToCoverElemList) > 0 {
			fileElement := FileCoverage{
				Path:        fileName,
				LineToCover: lineToCoverElemList,
			}
			fileElementList = append(fileElementList, fileElement)
		}

	}

	createXmlFile(fileElementList, "coverage")

}

// Preare LineTocover element
func prepareLineElement(filename string, total string, covered string, isCovered bool) LineToCover {
	lineToCoverElement := LineToCover{
		LineNumber:      strings.Trim(filename, " "),
		BranchesToCover: total,
		CoveredBranches: covered,
		Covered:         fmt.Sprint(isCovered),
	}
	return lineToCoverElement
}

// Pass data and create generic test coverage report for sonaqube
func createXmlFile(fileElementList []FileCoverage, folderPath string) {
	xmlData := coverage{
		Version: "1",
		File:    fileElementList}
	xmlFile, err := os.Create(folderPath + "/branch-coverage.xml")
	enc := xml.NewEncoder(xmlFile)
	enc.Indent("", "\t")
	if err := enc.Encode(xmlData); err != nil {
		fmt.Printf("error: %v\n", err)
	}

	if err != nil {
		fmt.Println("Error creating XML file: ", err)
		return
	}
}
