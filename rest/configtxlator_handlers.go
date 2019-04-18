/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rest

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/tools/configtxlator/sanitycheck"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	cb "github.com/hyperledger/fabric/protos/common"
)

func fieldBytes(fieldName string, r *http.Request) ([]byte, error) {
	fieldFile, _, err := r.FormFile(fieldName)
	if err != nil {
		return nil, err
	}
	defer fieldFile.Close()

	return ioutil.ReadAll(fieldFile)
}

func fieldConfigProto(fieldName string, r *http.Request) (*cb.Config, error) {
	fieldBytes, err := fieldBytes(fieldName, r)
	if err != nil {
		return nil, fmt.Errorf("error reading field bytes: %s", err)
	}

	config := &cb.Config{}
	err = proto.Unmarshal(fieldBytes, config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling field bytes: %s", err)
	}

	return config, nil
}

func ComputeUpdateFromConfigs(w http.ResponseWriter, r *http.Request) {
	originalConfig, err := fieldConfigProto("original", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'original': %s\n", err)
		return
	}

	updatedConfig, err := fieldConfigProto("updated", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error with field 'updated': %s\n", err)
		return
	}

	configUpdate, err := update.Compute(originalConfig, updatedConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error computing update: %s\n", err)
		return
	}

	configUpdate.ChannelId = r.FormValue("channel")

	encoded, err := proto.Marshal(configUpdate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling config update: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(encoded)
}

func SanityCheckConfig(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	config := &cb.Config{}
	err = proto.Unmarshal(buf, config)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error unmarshaling data to common.Config': %s\n", err)
		return
	}

	fmt.Printf("Sanity checking %+v\n", config)
	sanityCheckMessages, err := sanitycheck.Check(config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error performing sanity check: %s\n", err)
		return
	}

	resBytes, err := json.Marshal(sanityCheckMessages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error marshaling result to JSON: %s\n", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return err
		}

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			fileReader.Close()
			targetFile.Close()
			return err
		}
		fileReader.Close()
		targetFile.Close()
	}

	return nil
}

func isZip(zipPath string) bool {
	f, err := os.Open(zipPath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 4)
	if n, err := f.Read(buf); err != nil || n < 4 {
		return false
	}

	return bytes.Equal(buf, []byte("PK\x03\x04"))
}

func createDir(filePath string) error {
	if !isExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		return err
	}
	return nil
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func UploadMspFiles(w http.ResponseWriter, r *http.Request) {
	destination := r.FormValue("destination")
	if destination == "" {
		w.WriteHeader(http.StatusBadRequest)
		errMsg := "Lack of field 'destination'\n"
		fmt.Println(errMsg)
		fmt.Fprint(w, errMsg)
		return
	}
	if err := createDir(destination); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("Create dir err: ", err)
		fmt.Fprint(w, "Create dir err: %s", err)
		return
	}

	r.ParseForm()
	// Store uploaded files in memory and temporary files
	r.ParseMultipartForm(32 << 20)
	// Gets the file handle, stores the file
	file, handler, err := r.FormFile("msp")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("Form file err: ", err)
		fmt.Fprint(w, "Form file err: %s", err)
		return
	}
	defer file.Close()
	fmt.Fprintf(w, "%v", handler.Header)
	// Create the uploaded destination file
	zipFile := destination + handler.Filename
	f, err := os.OpenFile(zipFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("Open file err: ", err)
		fmt.Fprint(w, "Open file err: %s", err)
		return
	}
	defer f.Close()

	if _, err = io.Copy(f, file); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("Copy file err: ", err)
		fmt.Fprint(w, "Copy file err: %s", err)
		return
	}

	if isZip(zipFile) {
		if err = unzip(zipFile, destination); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("Unzip file err: ", err)
			fmt.Fprint(w, "Unzip file err: %s", err)
			return
		}

		if err = os.Remove(zipFile); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("Remove file err: ", err)
			fmt.Fprint(w, "Remove file err: %s", err)
			return
		}
	}
}
