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
	Start      string
	Code       string
	TrueCount  int
	FalseCount int
}
type BranchCoverageDataCoverage struct {
	Filename        string                `json:"filename"`
	Covered         int64                 `json:"covered"`
	BranchesToCover int64                 `json:"total"`
	CoverageRate    float64               `json:"coverageRate"`
	CoverageData    []*BranchCoverageData `json:"coverageData"`
}
type BranchCoverage struct {
	CoverageRate       float64                      `json:"coverageRate"`
	BranchesToCover    int64                        `json:"totalBranch"`
	TotalCoveredBranch int64                        `json:"totalCoveredBranch"`
	AllData            []BranchCoverageDataCoverage `json:"allData"`
}

func main() {

	// Get command line argument1
	packageName := flag.String("packageName", "", "")
	// Get command line argument2
	isUpdateJson := flag.Bool("isUpdateJson", false, "")
	flag.Parse()

	currentContent, err := ioutil.ReadFile(*packageName + "/branch-cover.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	// Now let's unmarshall the data into `currentContent`
	var payloadCurrent []*BranchCoverageData
	var newpayloadCurrent []*BranchCoverageData
	var allDataCover []BranchCoverageDataCoverage
	err = json.Unmarshal([]byte(currentContent), &payloadCurrent)
	if err != nil {
		log.Fatal("Error during Unmarshal() of currentContent: ", err)
	}

	if !(*isUpdateJson) {
		var filenamecover string
		var totalBranchInFile int64
		var coveredBranchInFile int64
		var totalBranch int64
		var coveredBranch int64
		if *packageName == "./test-results" {
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
					totalBranchInFile = 0
					coveredBranchInFile = 0
					newpayloadCurrent = nil
					filenamecover = fileName
				}

				newpayloadCurrent = append(newpayloadCurrent, payloadCurrent[i])
				totalBranchInFile = totalBranchInFile + 2
				if payloadCurrent[i].TrueCount > 0 {
					coveredBranchInFile++
				}
				if payloadCurrent[i].FalseCount > 0 {
					coveredBranchInFile++
				}
				if i == 0 {
					filenamecover = fileName
				}
			}
			xmlData := &coverage{
				Version: "1",
				File:    fileElementList}

			xmlFile, err := os.Create("test-results/branch-coverage.xml")
			enc := xml.NewEncoder(xmlFile)
			enc.Indent("", "\t")
			if err := enc.Encode(xmlData); err != nil {
				fmt.Printf("error: %v\n", err)
			}

			if err != nil {
				fmt.Println("Error creating XML file: ", err)
				return
			}

			/* */
			var branchCoverage BranchCoverage
			branchCoverage.AllData = allDataCover
			branchCoverage.CoverageRate = getScore(totalBranch, coveredBranch)
			branchCoverage.BranchesToCover = totalBranch
			branchCoverage.TotalCoveredBranch = coveredBranch
			jsonfileall, _ := json.Marshal(branchCoverage)
			_ = ioutil.WriteFile("test-results/all-branch-cover.json", jsonfileall, 0644)
			log.Println("Total Branches: ", totalBranch)
			log.Println("Total Covered: ", coveredBranch)
			log.Println("Coverage= ", getScore(totalBranch, coveredBranch))

			fmt.Println("done writing to C1 as C0 data")

		} else {
			// To update package name of the files in gobco output json file(it only has filename without package)
			for i := 0; i < len(payloadCurrent); i++ {
				lineNumber := payloadCurrent[i].Start
				fileName := strings.Split(lineNumber, ":")[0]
				lineNumberStartSplitted, _ := strconv.Atoi(strings.Split(lineNumber, ":")[1])
				ColNumberStartSplitted, _ := strconv.Atoi(strings.Split(lineNumber, ":")[2])

				payloadCurrent[i].Start = *packageName + "/" + fileName + ":" + strconv.Itoa(lineNumberStartSplitted) + ":" + strconv.Itoa(ColNumberStartSplitted)
			}
		}
		jsonfile, _ := json.Marshal(payloadCurrent)
		_ = ioutil.WriteFile(*packageName+"/branch-cover.json", jsonfile, 0644)
	} else if *isUpdateJson {
		// Copy branch-cover.json file from each package and write it in test-results folder
		contentAll, err := ioutil.ReadFile("test-results/branch-cover.json")
		if err != nil {
			log.Fatal("Error when opening file: ", err)
		}

		if len(contentAll) > 0 {
			var payloadAll []*BranchCoverageData
			err = json.Unmarshal([]byte(contentAll), &payloadAll)
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
func getScore(total int64, covered int64) float64 {
	return math.Floor(float64(covered) / float64(total) * 100)
}

// Create lineToCover element list for a file
func createDataForXmlfile(branchCoverageDataCoverage BranchCoverageDataCoverage) []LineToCover {
	var branchCoverDataForaFile = branchCoverageDataCoverage.CoverageData
	var lineToCoverElemList []LineToCover
	for j := 0; j < len(branchCoverDataForaFile); j++ {
		var coveredBranchXml = 0
		var isCovered = true
		if branchCoverageDataCoverage.CoverageData[j].TrueCount > 0 {
			coveredBranchXml++
		}
		if branchCoverageDataCoverage.CoverageData[j].FalseCount > 0 {
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
