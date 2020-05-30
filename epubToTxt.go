package main

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type inputParams struct {
	epubDir      string
	outputFolder string
	regexFile    string
}

type containerXMLParams struct {
	RootFiles struct {
		RootFile struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type opfXMLParams struct {
	Manifest struct {
		Items []struct {
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
			ID        string `xml:"id,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			Idref string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

type regexValues struct {
	expression  string
	replacement string
}

func main() {
	start := time.Now()

	// read Arguments
	inputParams := readUserInput()
	// Open the zip file
	r, err := zip.OpenReader(inputParams.epubDir)
	checkError(err)
	defer r.Close()

	// Read the regex file
	regexes := readRegexFile(inputParams.regexFile)

	// read container.xml
	containerData := getContainerData(r)
	opfFilePath := filepath.Join(containerData.RootFiles.RootFile.FullPath)

	// read and parse the opf file
	opfData := getOPFData(r, opfFilePath)

	// get relevant filePaths from the opf data
	HTMLfileList := getHTMLFileList(opfData, filepath.Dir(opfFilePath))
	allText := readHTMLFiles(r, HTMLfileList, regexes)

	//Save the txt file
	createTextFile(inputParams.outputFolder+".txt", allText.String())
	fmt.Printf("Saved to the text file : %s \r\n Conversion process took %s", inputParams.outputFolder+".txt", time.Since(start))

}

func readFileFromZip(src *zip.ReadCloser, path string) (string, error) {
	pathOppositeSlash := strings.Replace(path, "\\", "/", -1)
	for _, f := range src.File {
		if f.FileHeader.Name == path || f.FileHeader.Name == pathOppositeSlash {
			rc, err := f.Open()
			buf := new(strings.Builder)
			if err != nil {
				return "", err
			}
			io.Copy(buf, rc)
			return buf.String(), nil
		}
	}
	return "", errors.New("no such file")
}

func createTextFile(filePath, text string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(text)
	if err != nil {
		log.Fatal(err)
	}
}

func readRegexFile(regexFile string) []regexValues {

	var regexes []regexValues
	if regexFile == "" {
		return nil
	}
	file, err := os.Open(regexFile)
	checkError(err)

	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		if (i+1)%2 == 1 {
			regexes = append(regexes, regexValues{scanner.Text(), ""})
		} else {
			regexes[(i-1)/2].replacement = scanner.Text()
		}
		i++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return regexes
}

func applyRegex(regex regexValues, doc []byte) []byte {
	re := regexp.MustCompile(regex.expression)
	return re.ReplaceAll(doc, []byte(regex.replacement))
}

func readHTMLFiles(src *zip.ReadCloser, fileList []string, regexes []regexValues) strings.Builder {

	var allData strings.Builder
	for _, file := range fileList {
		htmlCon, err := readFileFromZip(src, file)
		htmlContent := []byte(htmlCon)
		if regexes != nil {
			for _, regex := range regexes {
				htmlContent = applyRegex(regex, (htmlContent))
			}
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
		checkError(err)

		doc.Find("body").Each(func(i int, s *goquery.Selection) {
			allData.WriteString(s.Text())
		})
	}
	return allData
}

func getHTMLFileList(opfData opfXMLParams, opfFileDir string) []string {

	var fileList []string
	for _, idref := range opfData.Spine.ItemRefs {
		for _, item := range opfData.Manifest.Items {
			if item.ID == idref.Idref {
				fileList = append(fileList, filepath.Join(opfFileDir, item.Href))
				break
			}
		}
	}
	return fileList
}

func getOPFData(src *zip.ReadCloser, opfFilePath string) opfXMLParams {

	opfContent, err := readFileFromZip(src, opfFilePath)
	checkError(err)

	opfData := opfXMLParams{}
	err = xml.Unmarshal([]byte(opfContent), &opfData)
	checkError(err)

	return opfData
}

func getContainerData(src *zip.ReadCloser) containerXMLParams {

	containerFile := filepath.Join("META-INF", "container.xml")
	containerContent, err := readFileFromZip(src, containerFile)
	checkError(err)

	containerData := containerXMLParams{}
	err = xml.Unmarshal([]byte(containerContent), &containerData)
	checkError(err)

	return containerData
}

func readUserInput() inputParams {

	epubDir, regexFile := os.Args[1], ""
	if len(os.Args) == 3 {
		regexFile = os.Args[2]
	}
	
	fileName := string([]rune(filepath.Base(epubDir))[:len(filepath.Base(epubDir))-len(filepath.Ext(epubDir))])
	outputFolder := filepath.Join(filepath.Dir(epubDir), fileName)
	return inputParams{epubDir, outputFolder, regexFile}

}

func checkError(err error) {

	if err != nil {
		log.Fatal(err)
	}
}
