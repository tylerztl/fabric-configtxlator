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
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/common/tools/protolator"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"

	"github.com/pkg/errors"
)

const pkgLogID = "common/tools/configtxlator/rest"

var logger = flogging.MustGetLogger(pkgLogID)

func init() {
	flogging.SetModuleLevel(pkgLogID, "info")
}

func OutputGenesisBlock(w http.ResponseWriter, r *http.Request) {
	profile := r.FormValue("profile")
	if profile == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'profile'\n")
		return
	}

	channelID := r.FormValue("channelID")
	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'channelID'\n")
		return
	}

	// ubuntu related golang version
	//configtxStr := r.FormValue("configtx")
	//if configtxStr == "" {
	//	w.WriteHeader(http.StatusBadRequest)
	//	fmt.Fprint(w, "Missing field 'configtx'\n")
	//	return
	//}
	//configtx := []byte(configtxStr)
	// mac
	configtx, err := fieldBytes("configtx", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'configtx'\n")
		return
	}

	configPath := r.FormValue("configPath")
	if configPath == "" {
		configPath = "."
	}

	outputBlock := r.FormValue("outputBlock")
	//if outputBlock == "" {
	//	w.WriteHeader(http.StatusBadRequest)
	//	fmt.Fprint(w, "Missing field 'outputBlock'\n")
	//	return
	//}

	logger.Info("Loading configuration")

	factory.InitFactories(nil)

	//profileConfig := genesisconfig.Load(profile, configPath)
	profileConfig := genesisconfig.LoadFromBytes(profile, configtx, configPath)

	blockBytes, err := doOutputBlock(profileConfig, channelID, outputBlock)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error on outputBlock: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(blockBytes)
}

func OutputChannelCreateTx(w http.ResponseWriter, r *http.Request) {
	profile := r.FormValue("profile")
	if profile == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'profile'\n")
		return
	}

	channelID := r.FormValue("channelID")
	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'channelID'\n")
		return
	}

	// ubuntu related golang version
	//configtxStr := r.FormValue("configtx")
	//if configtxStr == "" {
	//	w.WriteHeader(http.StatusBadRequest)
	//	fmt.Fprint(w, "Missing field 'configtx'\n")
	//	return
	//}
	//configtx := []byte(configtxStr)
	// mac
	configtx, err := fieldBytes("configtx", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'configtx'\n")
		return
	}

	configPath := r.FormValue("configPath")
	if configPath == "" {
		configPath = "."
	}

	outputCreateChannelTx := r.FormValue("outputCreateChannelTx")
	//if outputCreateChannelTx == "" {
	//	w.WriteHeader(http.StatusBadRequest)
	//	fmt.Fprint(w, "Missing field 'outputCreateChannelTx'\n")
	//	return
	//}

	factory.InitFactories(nil)

	//profileConfig := genesisconfig.Load(profile, configPath)
	profileConfig := genesisconfig.LoadFromBytes(profile, configtx, configPath)

	configtxBytes, err := doOutputChannelCreateTx(profileConfig, channelID, outputCreateChannelTx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error on outputChannelCreateTx: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(configtxBytes)
}

func PrintOrg(w http.ResponseWriter, r *http.Request) {
	printOrg := r.FormValue("printOrg")
	if printOrg == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'printOrg'\n")
		return
	}

	// ubuntu related golang version
	//configtxStr := r.FormValue("configtx")
	//if configtxStr == "" {
	//	w.WriteHeader(http.StatusBadRequest)
	//	fmt.Fprint(w, "Missing field 'configtx'\n")
	//	return
	//}
	//configtx := []byte(configtxStr)
	// mac
	configtx, err := fieldBytes("configtx", r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing field 'configtx'\n")
		return
	}

	configPath := r.FormValue("configPath")
	if configPath == "" {
		configPath = "."
	}

	factory.InitFactories(nil)
	//topLevelConfig := genesisconfig.LoadTopLevel()
	topLevelConfig := genesisconfig.LoadTopLevelFromBytes(configtx, configPath)
	orgBytes, err := doPrintOrg(topLevelConfig, printOrg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error on printOrg: %s\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(orgBytes)
}

func doOutputBlock(config *genesisconfig.Profile, channelID string, outputBlock string) ([]byte, error) {
	pgen := encoder.New(config)
	logger.Info("Generating genesis block")
	if config.Consortiums == nil {
		logger.Warning("Genesis block does not contain a consortiums group definition.  This block cannot be used for orderer bootstrap.")
	}
	genesisBlock := pgen.GenesisBlockForChannel(channelID)

	logger.Info("Writing genesis block")
	blockBytes, err := utils.Marshal(genesisBlock)
	if err != nil {
		return blockBytes, err
	}

	if outputBlock != "" {
		err = ioutil.WriteFile(outputBlock, blockBytes, 0644)
	}
	return blockBytes, err
}

func doOutputChannelCreateTx(conf *genesisconfig.Profile, channelID string, outputChannelCreateTx string) ([]byte, error) {
	logger.Info("Generating new channel configtx")

	configtx, err := encoder.MakeChannelCreationTransaction(channelID, nil, conf)
	if err != nil {
		return nil, err
	}

	logger.Info("Writing new channel tx")
	configtxBytes, err := utils.Marshal(configtx)
	if err != nil {
		return configtxBytes, err
	}

	if outputChannelCreateTx != "" {
		err = ioutil.WriteFile(outputChannelCreateTx, configtxBytes, 0644)
	}
	return configtxBytes, err
}

func doPrintOrg(t *genesisconfig.TopLevel, printOrg string) ([]byte, error) {
	for _, org := range t.Organizations {
		if org.Name == printOrg {
			og, err := encoder.NewOrdererOrgGroup(org)
			if err != nil {
				return nil, errors.Wrapf(err, "bad org definition for org %s", org.Name)
			}

			orgBytes, err := protolator.MarshalToJSON(&cb.DynamicConsortiumOrgGroup{ConfigGroup: og})
			if err != nil {
				return nil, errors.Wrapf(err, "malformed org definition for org: %s", org.Name)
			}
			return orgBytes, nil
		}
	}
	return nil, errors.Errorf("organization %s not found", printOrg)
}
